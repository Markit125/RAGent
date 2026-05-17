package telegram

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/telebot.v3"

	"chatbot/internal/agent"
	"chatbot/internal/domain"
	"chatbot/internal/infra/google"
	"chatbot/internal/infra/redis"
)

type Bot struct {
	bot *telebot.Bot

	llm          domain.LLM
	embedder     domain.Embedder
	vectorStore  domain.VectorStore
	redis        *redis.Client
	googleSearch *google.GoogleSearchClient

	systemPrompt string
}

func NewBot(token string, systemPrompt string, llm domain.LLM, embedder domain.Embedder, store domain.VectorStore, redis *redis.Client, googleSearch *google.GoogleSearchClient) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		URL:    "https://api.telegram.org",
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		bot:          b,
		llm:          llm,
		embedder:     embedder,
		vectorStore:  store,
		redis:        redis,
		googleSearch: googleSearch,
		systemPrompt: systemPrompt,
	}

	bot.bot.Handle(telebot.OnText, bot.onText)
	bot.bot.Handle("/reset", bot.resetRedis)

	return bot, nil
}

func (b *Bot) Start() {
	log.Println("Telegram bot started")
	b.bot.Start()
}

func (b *Bot) resetRedis(c telebot.Context) error {
	userID := c.Sender().ID
	err := b.redis.ClearHistory(context.Background(), userID)
	if err != nil {
		return c.Send("Ошибка очистки кэша: " + err.Error())
	}

	return c.Send("🗑 История диалога в Redis очищена! Начинаем с чистого листа.")
}

func (b *Bot) onText(c telebot.Context) error {
	userID := c.Sender().ID
	msgID := c.Message().ID
	userText := c.Text()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	isDup, err := b.redis.IsDuplicate(ctx, userID, msgID)
	if err != nil {
		log.Printf("⚠️ Redis dedup error: %v", err)
	}
	if isDup {
		log.Printf("♻️ [IGNORE] Дубликат сообщения ID %d от user %d", msgID, userID)
		return nil
	}

	vector, err := b.embedder.Embed(context.Background(), userText)
	var ragContext string

	if err == nil {
		memories, err := b.vectorStore.Search(
			context.Background(),
			vector,
			userID,
			3,
			0.75,
		)

		if err == nil && len(memories) > 0 {

			ragContext = "НАЙДЕННЫЕ ФАКТЫ ИЗ ПАМЯТИ:\n"
			for _, m := range memories {
				ragContext += fmt.Sprintf("- %s\n", m.Text)
			}
		}
	}

	log.Printf("[%d] User: %s", userID, userText)
	_ = c.Notify(telebot.Typing)

	history, err := b.redis.GetHistory(ctx, userID)
	if err != nil {
		log.Printf("⚠️ Failed to load history: %v", err)
		history = []domain.Message{}
	}

	currentDate := time.Now().Format("02.01.2006, 15:04")
	dynamicSystemPrompt := strings.ReplaceAll(b.systemPrompt, "{{CURRENT_TIME}}", currentDate)

	if len(history) == 0 {
		sysMsg := domain.Message{Role: domain.RoleSystem, Content: dynamicSystemPrompt}
		history = append(history, sysMsg)

		go b.redis.AddMessage(context.Background(), userID, sysMsg)
	} else {

		if history[0].Role == domain.RoleSystem {
			history[0].Content = dynamicSystemPrompt
		}
	}

	finalUserMessage := userText

	if ragContext != "" {
		finalUserMessage = fmt.Sprintf(
			"Справочная информация из памяти:\n%s\n\n---\nСообщение пользователя:\n%s",
			ragContext,
			userText,
		)
	}

	userMsg := domain.Message{Role: domain.RoleUser, Content: finalUserMessage}
	history = append(history, userMsg)

	go b.redis.AddMessage(context.Background(), userID, userMsg)

	services := domain.ServiceContainer{
		VectorStore:  b.vectorStore,
		Embedder:     b.embedder,
		UserID:       domain.UserID(userID),
		GoogleSearch: b.googleSearch,
	}

	respMsg, err := b.llm.Chat(ctx, history)
	if err != nil {
		log.Printf("LLM error: %+v", err)
		return c.Send("⚠️ Ошибка нейросети")
	}

	actions, err := agent.ParseActions(respMsg)
	if err != nil {
		log.Printf("Parse warning: %+v", err)
	}

	var finalResponseText string

	if len(actions) == 0 {

		finalResponseText = agent.CleanThinking(respMsg.Content)
	} else {

		action := actions[0]
		log.Printf("Executing action: %+v", action)

		execResult, err := action.Execute(ctx, services)
		if err != nil {
			execResult = "❌ Ошибка: " + err.Error()
		}

		_, isSearch := action.(domain.SearchAction)
		_, isGoogle := action.(domain.GoogleAction)

		if isSearch || isGoogle {

			log.Println("🔄 RAG Loop: Получены данные поиска, уточняем ответ...")

			assCmd := domain.Message{Role: domain.RoleAssistant, Content: respMsg.Content}
			history = append(history, assCmd)

			sysObs := domain.Message{Role: domain.RoleSystem, Content: execResult}
			history = append(history, sysObs)

			forceMsg := domain.Message{
				Role:    domain.RoleUser,
				Content: "(ВАЖНО) Используя результаты поиска выше, дай развернутый ответ на мой изначальный вопрос. Не используй команды, пиши обычным текстом.",
			}
			history = append(history, forceMsg)

			go func() {
				b.redis.AddMessage(context.Background(), userID, assCmd)
				b.redis.AddMessage(context.Background(), userID, sysObs)
			}()

			_ = c.Notify(telebot.Typing)
			finalResp, err := b.llm.Chat(ctx, history)
			if err != nil {
				return c.Send("⚠️ Ошибка при генерации финального ответа RAG")
			}

			finalResponseText = agent.CleanThinking(finalResp.Content)

			if (finalResponseText == "") && len(execResult) > 20 {

				cleanFact := strings.Replace(execResult, "SYSTEM_OBSERVATION:", "", 1)
				finalResponseText = "Я нашел в памяти вот это, но не смог сформулировать мысль:\n" + cleanFact
			}

		} else {

			chatPart := agent.CleanThinking(respMsg.Content)

			parts := []string{}

			if chatPart != "" {
				parts = append(parts, chatPart)
			}
			parts = append(parts, execResult)

			finalResponseText = strings.Join(parts, "\n\n")

			assCmd := domain.Message{Role: domain.RoleAssistant, Content: respMsg.Content}
			go b.redis.AddMessage(context.Background(), userID, assCmd)
		}
	}

	finalResponseText = agent.CleanThinking(finalResponseText)

	botMsg := domain.Message{Role: domain.RoleAssistant, Content: finalResponseText}
	go b.redis.AddMessage(context.Background(), userID, botMsg)

	log.Printf("Response sent: %s", finalResponseText)
	return c.Send(finalResponseText)
}
