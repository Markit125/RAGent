package agent

import (
	"chatbot/internal/domain"
	"log"
	"regexp"
	"strings"
)

var cmdRegex = regexp.MustCompile(`(?i)CMD:(SAVE|SEARCH|GOOGLE)\s*\|\s*(.*)`)

func ParseActions(msg domain.Message) ([]domain.BotAction, error) {
	text := strings.TrimSpace(msg.Content)
	var actions []domain.BotAction

	matches := cmdRegex.FindStringSubmatch(text)

	publicText := CleanThinking(text)
	if publicText != "" {
		actions = append(actions, domain.ReplyAction{Content: publicText})
	}

	if matches != nil {

		cmdType := strings.ToUpper(matches[1])
		argsRaw := matches[2]
		parts := strings.Split(argsRaw, "|")

		switch cmdType {
		case "SAVE":

			content := strings.TrimSpace(parts[0])

			var tags string
			if len(parts) > 1 {
				tags = strings.TrimSpace(parts[1])
			}

			log.Printf("Parsed SAVE action: Text='%s', Tags='%s'", content, tags)

			return []domain.BotAction{
				domain.SaveAction{Text: content, Tags: tags},
			}, nil

		case "SEARCH":
			query := strings.TrimSpace(argsRaw)

			log.Printf("Parsed SEARCH action: Query='%s'", query)

			return []domain.BotAction{
				domain.SearchAction{Query: query},
			}, nil

		case "GOOGLE":
			query := strings.TrimSpace(parts[0])
			return []domain.BotAction{
				domain.GoogleAction{Query: query},
			}, nil
		}
	}

	return actions, nil
}

func CleanThinking(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	lastLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "МЫСЛЬ:") || strings.HasPrefix(trimmed, "THOUGHT:") {
			continue
		}

		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, "CMD:") {
			continue
		}

		if strings.HasPrefix(trimmed, "SYSTEM_OBSERVATION:") {
			continue
		}
		if strings.HasPrefix(trimmed, "- [") && strings.Contains(trimmed, "]") {
			continue
		}

		if strings.HasPrefix(upper, "ПРАВИЛО:") || strings.HasPrefix(upper, "RULE:") {
			continue
		}
		if strings.HasPrefix(upper, "ИНСТРУКЦИЯ:") {
			continue
		}

		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "<eos>") ||
			strings.Contains(lower, "user:") ||
			(strings.HasPrefix(lower, "user") && len(lower) < 10) {
			break
		}

		prefixesToRemove := []string{"ОТВЕТ:", "RESPONSE:", "Assistant:", "ASSISTANT:"}
		for _, p := range prefixesToRemove {
			if len(trimmed) >= len(p) && strings.EqualFold(trimmed[:len(p)], p) {
				trimmed = strings.TrimSpace(trimmed[len(p):])
			}
		}

		if trimmed == lastLine {
			continue
		}

		if trimmed != "" {
			result = append(result, trimmed)
			lastLine = trimmed
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}
