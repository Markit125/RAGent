package domain

import "context"

type VectorStore interface {
	Save(ctx context.Context, m Memory) error
	Search(ctx context.Context, vector []float32, userID int64, limit int, minScore float32) ([]Memory, error)
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type LLM interface {
	Chat(ctx context.Context, history []Message) (Message, error)
}
