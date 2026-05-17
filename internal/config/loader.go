package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type BotConfig struct {
	LLM              LLMConfig   `json:"llm"`
	Store            StoreConfig `json:"store"`
	Redis            RedisConfig `json:"redis"`
	SystemPromptFile string      `json:"system_prompt_file"`
	SystemPrompt     string      `json:"-"`
}

type LLMConfig struct {
	Host       string     `json:"host"`
	ChatModel  string     `json:"chat_model"`
	EmbedModel string     `json:"embed_model"`
	Options    LLMOptions `json:"options"`
}

type LLMOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumCtx      *int     `json:"num_ctx,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

type StoreConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	CollectionName string `json:"collection_name"`
	VectorSize     uint64 `json:"vector_size"`
}

type RedisConfig struct {
	Host            string `json:"host"`
	Password        string `json:"password"`
	DB              int    `json:"db"`
	HistoryTTLHours int    `json:"history_ttl_hours"`
}

func LoadConfig(configPath string) (*BotConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg BotConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.SystemPromptFile != "" {
		promptData, err := os.ReadFile(cfg.SystemPromptFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read system prompt file '%s': %w", cfg.SystemPromptFile, err)
		}
		cfg.SystemPrompt = string(promptData)
	} else {
		cfg.SystemPrompt = "Ты должен писать, что системный проомпт не указан."
	}

	return &cfg, nil
}
