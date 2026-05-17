package qdrant

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"chatbot/internal/domain"
)

type QdrantStore struct {
	client         *qdrant.Client
	collectionName string
}

func NewStore(host string, port int, collectionName string) (*QdrantStore, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	return &QdrantStore{
		client:         client,
		collectionName: collectionName,
	}, nil
}

func (q *QdrantStore) Init(ctx context.Context, vectorSize uint64) error {
	collections, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	if slices.Contains(collections, q.collectionName) {
		return nil
	}

	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

func (q *QdrantStore) Save(ctx context.Context, m domain.Memory) error {
	log.Printf("Saving memory: %v", m.Text)
	pointID := uuid.New().String()

	vectors := qdrant.NewVectors(m.Vector...)

	tagsAny := make([]any, len(m.Tags))
	for i, v := range m.Tags {
		tagsAny[i] = v
	}

	payloadMap := map[string]any{
		"text":       m.Text,
		"user_id":    int64(m.UserID),
		"tags":       tagsAny,
		"created_at": time.Now().Unix(),
		"date_str":   time.Now().Format("2006-01-02 15:04:05"),
	}
	payload := qdrant.NewValueMap(payloadMap)

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewID(pointID),
				Vectors: vectors,
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to save point: %w", err)
	}

	log.Printf("Memory saved: %s, Tags: %v", m.Text, m.Tags)
	return nil
}

func (q *QdrantStore) Search(ctx context.Context, vector []float32, userID int64, limit int, minScore float32) ([]domain.Memory, error) {
	log.Printf("Searching memory for user %d", userID)

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatchInt("user_id", userID),
		},
	}

	res, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collectionName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		ScoreThreshold: qdrant.PtrOf(float32(minScore)),
		WithPayload:    qdrant.NewWithPayload(true),
		Filter:         filter,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	memories := make([]domain.Memory, 0, len(res))

	for _, item := range res {
		text := ""
		if val, ok := item.Payload["text"]; ok {
			text = val.GetStringValue()
		}

		var tags []string
		if val, ok := item.Payload["tags"]; ok {

			if listVal := val.GetListValue(); listVal != nil {
				for _, v := range listVal.Values {
					tags = append(tags, v.GetStringValue())
				}
			}
		}

		var createdAt time.Time
		if val, ok := item.Payload["date_str"]; ok {
			dateStr := val.GetStringValue()

			parsed, err := time.Parse("2006-01-02 15:04:05", dateStr)
			if err == nil {
				createdAt = parsed
			}
		}

		memories = append(memories, domain.Memory{
			Text:      text,
			UserID:    domain.UserID(userID),
			Tags:      tags,
			CreatedAt: createdAt,
		})
	}

	log.Printf("Searched memory for user %d: found %d items", userID, len(memories))

	return memories, nil
}
