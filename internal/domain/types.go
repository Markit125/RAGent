package domain

type UserID int64

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

type ActionType string

const (
	ActionReply  ActionType = "reply"
	ActionSave   ActionType = "save"
	ActionSearch ActionType = "search"
)

type SaveArgs struct {
	Text string `json:"text"`
	Tags string `json:"tags"`
}

type SearchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type BotDecision struct {
	Action ActionType

	ReplyText    string
	SaveParams   *SaveArgs
	SearchParams *SearchArgs
}
