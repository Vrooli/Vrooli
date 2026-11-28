package codex

import (
	"encoding/json"
	"encoding/xml"
	"strings"
)

// NativeEvent is the provider-owned identity carried by an app-server
// notification. Text is optional because lifecycle and approval events do not
// necessarily contain displayable content.
type NativeEvent struct {
	Kind              string
	ThreadID          string
	TurnID            string
	ItemID            string
	MessageID         string
	BoundaryID        string
	Role              string
	Text              string
	Partial           bool
	VoiceHandoff      bool
	Controllable      bool
	CompactionLineage string
}

// MapNotification translates the stable identity-bearing parts of a Codex
// app-server notification. Unknown notifications are intentionally ignored;
// callers must never invent native provenance from transcript text.
func MapNotification(message Message) (NativeEvent, bool) {
	if message.Method == "" || len(message.Params) == 0 {
		return NativeEvent{}, false
	}
	switch message.Method {
	case "thread/started", "thread/updated", "turn/started", "turn/completed", "item/started", "item/completed", "item/agentMessage/delta", "error", "approval/requested":
	default:
		return NativeEvent{}, false
	}
	var params map[string]json.RawMessage
	if json.Unmarshal(message.Params, &params) != nil {
		return NativeEvent{}, false
	}
	event := NativeEvent{Kind: message.Method}
	event.ThreadID = firstString(params, "threadId", "threadID", "sessionId")
	event.TurnID = firstString(params, "turnId", "turnID")
	event.ItemID = firstString(params, "itemId", "itemID")
	event.MessageID = firstString(params, "messageId", "messageID")
	event.BoundaryID = firstString(params, "boundaryId", "boundaryID", "partId", "partID")
	event.CompactionLineage = firstString(params, "compactionLineage")
	if nested, ok := object(params, "thread"); ok {
		event.ThreadID = firstNonEmpty(event.ThreadID, firstString(nested, "id", "threadId"))
	}
	if nested, ok := object(params, "turn"); ok {
		event.TurnID = firstNonEmpty(event.TurnID, firstString(nested, "id", "turnId"))
	}
	if nested, ok := object(params, "item"); ok {
		event.ItemID = firstNonEmpty(event.ItemID, firstString(nested, "id", "itemId"))
		event.MessageID = firstNonEmpty(event.MessageID, firstString(nested, "messageId"))
		event.Role = firstString(nested, "role", "type")
		event.Text = firstNonEmpty(firstString(nested, "text", "delta"), textValue(nested["content"]))
	}
	event.Role = firstNonEmpty(event.Role, firstString(params, "role"))
	if event.Role == "userMessage" {
		event.Role = "user"
	} else if event.Role == "agentMessage" {
		event.Role = "assistant"
	}
	event.Text = firstNonEmpty(event.Text, firstString(params, "text", "delta"), textValue(params["content"]))
	if event.Role == "user" {
		input := ParseUserInput(event.Text)
		event.Text, event.VoiceHandoff = input.Text, input.VoiceHandoff
	}
	event.Partial = message.Method == "item/agentMessage/delta" || message.Method == "item/started"
	event.Controllable = event.TurnID != "" && (message.Method == "turn/completed" || message.Method == "item/completed" || message.Method == "item/agentMessage/delta")
	if event.BoundaryID == "" {
		event.BoundaryID = firstNonEmpty(event.ItemID, event.MessageID, event.TurnID)
	}
	return event, true
}

// UserInput retains display text and the origin of a native coding input.
type UserInput struct {
	Text         string
	VoiceHandoff bool
}

// ParseUserInput preserves a handoff receipt only when a coding user message
// contains the native realtime delegation envelope. Captions alone are not proof.
func ParseUserInput(text string) UserInput {
	ordinary := UserInput{Text: text}
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "<realtime_delegation>") || !strings.HasSuffix(trimmed, "</realtime_delegation>") {
		return ordinary
	}
	var delegation struct {
		XMLName xml.Name `xml:"realtime_delegation"`
		Source  string   `xml:"source"`
		Input   *string  `xml:"input"`
	}
	if xml.Unmarshal([]byte(trimmed), &delegation) != nil || delegation.Input == nil {
		return ordinary
	}
	if delegation.Source == "transcript_tail_flush" {
		return UserInput{}
	}
	return UserInput{Text: *delegation.Input, VoiceHandoff: strings.TrimSpace(*delegation.Input) != ""}
}

// DisplayUserText hides provider scaffolding without changing the original thread.
func DisplayUserText(text string) string { return ParseUserInput(text).Text }

func object(values map[string]json.RawMessage, key string) (map[string]json.RawMessage, bool) {
	var result map[string]json.RawMessage
	raw, ok := values[key]
	if !ok || json.Unmarshal(raw, &result) != nil {
		return nil, false
	}
	return result, true
}

func firstString(values map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		var value string
		if raw, ok := values[key]; ok && json.Unmarshal(raw, &value) == nil && value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func textValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	var values []map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return ""
	}
	for _, item := range values {
		if text := firstNonEmpty(firstString(item, "text"), textValue(item["content"])); text != "" {
			return text
		}
	}
	return ""
}
