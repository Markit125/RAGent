package domain

import "time"

type Memory struct {
	UserID    UserID
	Text      string
	Tags      []string
	Vector    []float32
	CreatedAt time.Time
}

type Message struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"arguments"`
}
