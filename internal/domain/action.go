package domain

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"chatbot/internal/infra/google"
)

type BotAction interface {
	Execute(ctx context.Context, services ServiceContainer) (string, error)
}

type ServiceContainer struct {
	VectorStore  VectorStore
	Embedder     Embedder
	UserID       UserID
	GoogleSearch *google.GoogleSearchClient
}

type ReplyAction struct {
	Content string
}

func (r ReplyAction) Execute(ctx context.Context, services ServiceContainer) (string, error) {
	return r.Content, nil
}

type SaveAction struct {
	Text string `json:"text"`
	Tags string `json:"tags"`
}

func (s SaveAction) Execute(ctx context.Context, services ServiceContainer) (string, error) {
	if s.Text == "" {
		return "Не понял, что именно нужно сохранить. Уточни, пожалуйста.", nil
	}

	rawTags := strings.Split(s.Tags, ",")
	cleanTags := make([]string, 0, len(rawTags))
	for _, t := range rawTags {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleanTags = append(cleanTags, strings.ToLower(trimmed))
		}
	}

	if len(cleanTags) == 0 {
		cleanTags = []string{"general"}
	}

	contentToEmbed := fmt.Sprintf("%s (теги: %s)", s.Text, strings.Join(cleanTags, ", "))

	vector, err := services.Embedder.Embed(ctx, contentToEmbed)
	if err != nil {
		return "", fmt.Errorf("failed to embed: %w", err)
	}

	memory := Memory{
		UserID:    services.UserID,
		Text:      s.Text,
		Tags:      cleanTags,
		Vector:    vector,
		CreatedAt: time.Now(),
	}

	log.Printf("Trying to save memory: %s, Tags: %v", memory.Text, memory.Tags)
	if err := services.VectorStore.Save(ctx, memory); err != nil {
		return "", fmt.Errorf("failed to save: %w", err)
	}
	log.Println("Saved memory")

	return fmt.Sprintf("✅ Записал в память: %s", s.Text), nil
}

type SearchAction struct {
	Query string `json:"query"`
}

func (s SearchAction) Execute(ctx context.Context, services ServiceContainer) (string, error) {
	if s.Query == "" {
		return "SYSTEM_NOTE: Пустой поисковый запрос.", nil
	}
	vector, err := services.Embedder.Embed(ctx, s.Query)
	if err != nil {
		return "", fmt.Errorf("embed error: %w", err)
	}

	results, err := services.VectorStore.Search(ctx, vector, int64(services.UserID), 3, 0.75)
	if err != nil {
		return "", fmt.Errorf("search error: %w", err)
	}

	if len(results) == 0 {
		return "SYSTEM_OBSERVATION: В памяти ничего не найдено по этому запросу.", nil
	}

	out := "SYSTEM_OBSERVATION: По твоему запросу в базе знаний найдено:\n"
	for _, m := range results {
		date := time.Unix(m.CreatedAt.Unix(), 0).Format("02.01.2006")
		out += fmt.Sprintf("- [%s] %s (Tags: %v)\n", date, m.Text, m.Tags)
	}

	return out, nil
}

type GoogleAction struct {
	Query string
}

func (g GoogleAction) Execute(ctx context.Context, services ServiceContainer) (string, error) {
	if services.GoogleSearch == nil {
		return "SYSTEM_ERROR: Поиск в интернете не настроен.", nil
	}
	return services.GoogleSearch.Search(ctx, g.Query)
}
