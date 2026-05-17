package main

import (
	"context"
	"flag"
	"log"
	"os"

	"chatbot/internal/config"
	"chatbot/internal/infra/google"
	"chatbot/internal/infra/ollama"
	"chatbot/internal/infra/qdrant"
	"chatbot/internal/infra/redis"
	"chatbot/internal/transport/telegram"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	configPath := flag.String("config", "config/bot_config.json", "path to config file")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN is not set")
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Load config error: %v", err)
	}

	log.Printf("Configuration loaded successfully.\nHost: %s\nChatModel: %s\nEmbedModel: %s",
		cfg.LLM.Host,
		cfg.LLM.ChatModel,
		cfg.LLM.EmbedModel,
	)

	log.Println("Initializing infrastructure...")

	ollamaClient := ollama.NewClient(cfg.LLM)

	log.Println("LLM warmup...")

	ctx := context.Background()
	if err := ollamaClient.WarmupEmbed(ctx); err != nil {
		log.Printf("Embed warmup error: %v", err)
	} else {
		log.Println("Embed warmup successful")
	}

	if err := ollamaClient.WarmupChat(ctx); err != nil {
		log.Printf("Chat warmup error: %v", err)
	} else {
		log.Println("Chat warmup successful")
	}

	qdrantStore, err := qdrant.NewStore(cfg.Store.Host, cfg.Store.Port, cfg.Store.CollectionName)
	if err != nil {
		log.Fatal(err)
	}

	if err := qdrantStore.Init(context.Background(), cfg.Store.VectorSize); err != nil {
		log.Fatal(err)
	}

	redisClient, err := redis.NewClient(cfg.Redis.Host, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.HistoryTTLHours)

	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY is not set")
	}

	googleCX := os.Getenv("GOOGLE_CX")
	if googleCX == "" {
		log.Fatal("GOOGLE_CX is not set")
	}

	googleSearchClient := google.NewClient(googleAPIKey, googleCX)

	tgBot, err := telegram.NewBot(
		token,
		cfg.SystemPrompt,
		ollamaClient,
		ollamaClient,
		qdrantStore,
		redisClient,
		googleSearchClient,
	)
	if err != nil {
		log.Fatal("Bot launch error:", err)
	}

	tgBot.Start()
}
