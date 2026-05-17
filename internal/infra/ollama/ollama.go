package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"chatbot/internal/config"
	"chatbot/internal/domain"
)

type OllamaClient struct {
	host       string
	client     *http.Client
	embedModel string
	chatModel  string
	options    map[string]interface{}
}

func NewClient(llmConfig config.LLMConfig) *OllamaClient {
	opts := parseOptions(llmConfig.Options)
	return &OllamaClient{
		host:       llmConfig.Host,
		embedModel: llmConfig.EmbedModel,
		chatModel:  llmConfig.ChatModel,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		options: opts,
	}
}

func (o *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model:  o.embedModel,
		Prompt: text,
	}

	respBody := ollamaEmbedResponse{}
	if err := o.makeRequest(ctx, "/api/embeddings", reqBody, &respBody); err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	return respBody.Embedding, nil
}

func (o *OllamaClient) Chat(ctx context.Context, history []domain.Message) (domain.Message, error) {

	msgs := make([]ollamaMessage, 0, len(history))
	for _, m := range history {

		msgs = append(msgs, ollamaMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	reqBody := ollamaChatRequest{
		Model:    o.chatModel,
		Messages: msgs,
		Stream:   false,

		Options: map[string]interface{}{
			"num_ctx":     4096,
			"num_predict": 512,
			"stop": []string{
				"User:",
				"user:",
				"Assistant:",
				"<|endoftext|>",
				"<|im_end|>",
				"<eos>",
				"\nUser",
			},
		},
	}

	respBody := ollamaChatResponse{}
	debugJSON, _ := json.MarshalIndent(reqBody, "", "  ")
	log.Printf("SENDING TO OLLAMA:\n%s\n", string(debugJSON))
	if err := o.makeRequest(ctx, "/api/chat", reqBody, &respBody); err != nil {
		return domain.Message{}, fmt.Errorf("chat failed: %w", err)
	}

	return domain.Message{
		Role:    domain.Role(respBody.Message.Role),
		Content: respBody.Message.Content,
	}, nil
}

func (o *OllamaClient) WarmupEmbed(ctx context.Context) error {
	_, err := o.Embed(ctx, "warmup")
	if err != nil {
		return err
	}
	return nil
}

func (o *OllamaClient) WarmupChat(ctx context.Context) error {
	_, err := o.Chat(ctx, []domain.Message{})
	if err != nil {
		return err
	}
	return nil
}

func (o *OllamaClient) makeRequest(ctx context.Context, endpoint string, payload interface{}, result interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.host+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func parseOptions(options config.LLMOptions) map[string]interface{} {
	opts := make(map[string]interface{})

	if options.Temperature != nil {
		opts["temperature"] = *options.Temperature
	}
	if options.NumCtx != nil {
		opts["num_ctx"] = *options.NumCtx
	}
	if options.NumPredict != nil {
		opts["num_predict"] = *options.NumPredict
	}
	if options.TopK != nil {
		opts["top_k"] = *options.TopK
	}
	if options.TopP != nil {
		opts["top_p"] = *options.TopP
	}

	return opts
}
