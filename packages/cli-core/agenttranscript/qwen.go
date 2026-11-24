package agenttranscript

import (
	"encoding/json"
	"strings"
)

// ParseQwenMessages reads selectable text from one Qwen Code chat-recording
// JSONL record. System metadata and model thought parts are not user-visible
// conversation content, so they are excluded. Session identity and source
// timestamp are carried from the enclosing native record.
func ParseQwenMessages(line []byte) []Message {
	var record struct {
		Type      string          `json:"type"`
		SessionID string          `json:"sessionId"`
		Timestamp json.RawMessage `json:"timestamp"`
		Message   struct {
			Role  string `json:"role"`
			Parts []struct {
				Text         string `json:"text"`
				Thought      bool   `json:"thought"`
				FunctionCall *struct {
					Name string          `json:"name"`
					ID   string          `json:"id"`
					Args json.RawMessage `json:"args"`
				} `json:"functionCall"`
				FunctionResponse *struct {
					Name     string          `json:"name"`
					ID       string          `json:"id"`
					Response json.RawMessage `json:"response"`
				} `json:"functionResponse"`
			} `json:"parts"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &record) != nil {
		return nil
	}
	role := ""
	switch record.Type {
	case "user":
		role = "user"
	case "assistant":
		role = "assistant"
	case "tool_result":
		role = "assistant"
	default:
		return nil
	}
	if record.Message.Role != "" {
		if record.Type == "user" && record.Message.Role != "user" || record.Type == "assistant" && record.Message.Role != "model" && record.Message.Role != "assistant" || record.Type == "tool_result" && record.Message.Role != "function" {
			return nil
		}
	}
	var messages []Message
	var textParts []string
	flushText := func() {
		text := strings.Join(textParts, "\n")
		if strings.TrimSpace(text) != "" {
			messages = append(messages, Message{Role: role, Text: text, SessionID: strings.TrimSpace(record.SessionID), OccurredAt: ParseTimestampValue(record.Timestamp)})
		}
		textParts = nil
	}
	for _, part := range record.Message.Parts {
		if part.Thought {
			continue
		}
		if part.FunctionCall != nil {
			flushText()
			if record.Type == "assistant" {
				text := FormatToolEvent(part.FunctionCall.Name, "call", part.FunctionCall.ID, part.FunctionCall.Args)
				messages = append(messages, Message{Role: "assistant", Text: text, SessionID: strings.TrimSpace(record.SessionID), OccurredAt: ParseTimestampValue(record.Timestamp)})
			}
			continue
		}
		if part.FunctionResponse != nil {
			flushText()
			if record.Type == "tool_result" {
				text := FormatToolEvent(part.FunctionResponse.Name, "result", part.FunctionResponse.ID, part.FunctionResponse.Response)
				messages = append(messages, Message{Role: "assistant", Text: text, SessionID: strings.TrimSpace(record.SessionID), OccurredAt: ParseTimestampValue(record.Timestamp)})
			}
			continue
		}
		if record.Type != "tool_result" && strings.TrimSpace(part.Text) != "" {
			textParts = append(textParts, part.Text)
		}
	}
	flushText()
	return messages
}
