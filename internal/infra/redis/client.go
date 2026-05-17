package redis

import (
	"chatbot/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb        *redis.Client
	historyTTL time.Duration
}

func NewClient(host, password string, db, ttlHours int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Client{
		rdb:        rdb,
		historyTTL: time.Duration(ttlHours) * time.Hour,
	}, nil
}

func (c *Client) AddMessage(ctx context.Context, userID int64, msg domain.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("chat:history:%d", userID)

	pipe := c.rdb.Pipeline()

	pipe.RPush(ctx, key, data)

	pipe.LTrim(ctx, key, -50, -1)

	pipe.Expire(ctx, key, c.historyTTL)

	_, err = pipe.Exec(ctx)
	return err
}

func (c *Client) GetHistory(ctx context.Context, userID int64) ([]domain.Message, error) {
	key := fmt.Sprintf("chat:history:%d", userID)

	rawMessages, err := c.rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	history := make([]domain.Message, 0, len(rawMessages))
	for _, raw := range rawMessages {
		var msg domain.Message
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			continue
		}
		history = append(history, msg)
	}

	return history, nil
}

func (c *Client) ClearHistory(ctx context.Context, userID int64) error {
	return c.rdb.Del(ctx, fmt.Sprintf("chat:history:%d", userID)).Err()
}

func (c *Client) IsDuplicate(ctx context.Context, userID int64, msgID int) (bool, error) {
	key := fmt.Sprintf("dedup:%d:%d", userID, msgID)

	saved, err := c.rdb.SetNX(ctx, key, 1, 1*time.Minute).Result()
	if err != nil {
		return false, err
	}

	return !saved, nil
}
