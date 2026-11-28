package agenttranscript

import (
	"encoding/json"
	"strings"
)

// ParseCursorConversation reads Cursor's native composer document and the
// separately stored bubble documents. Cursor persists headers under
// composerData:<composerId> and each message under
// bubbleId:<composerId>:<bubbleId> in its cursorDiskKV table.
//
// The database adapter supplies the native composer JSON plus bubble JSON
// values keyed by bubble ID. This codec only interprets their message shape;
// it does not open the database or retain the input bytes.
func ParseCursorConversation(composerJSON []byte, bubbles map[string][]byte) []Message {
	_, messages, _ := ParseCursorConversationForID(composerJSON, bubbles, "")
	return messages
}

// ParseCursorConversationForID reads the composer once to return its
// normalized identity and messages, optionally requiring a caller's expected
// composer ID. Message.SessionID retains the raw native ID for compatibility.
func ParseCursorConversationForID(composerJSON []byte, bubbles map[string][]byte, expectedSessionID string) (string, []Message, bool) {
	format, ok := FormatFor("cursor")
	if !ok || format.MessageDialect != "cursor-composer-kv" {
		return "", nil, false
	}
	var composer struct {
		ComposerID string `json:"composerId"`
		Headers    []struct {
			BubbleID string          `json:"bubbleId"`
			Type     json.RawMessage `json:"type"`
		} `json:"fullConversationHeadersOnly"`
	}
	if json.Unmarshal(composerJSON, &composer) != nil || strings.TrimSpace(composer.ComposerID) == "" {
		return "", nil, false
	}
	sessionID := strings.TrimSpace(composer.ComposerID)
	if expectedSessionID != "" && sessionID != expectedSessionID {
		return "", nil, false
	}
	messages := make([]Message, 0, len(composer.Headers))
	for _, header := range composer.Headers {
		role := cursorRole(header.Type)
		if role == "" || strings.TrimSpace(header.BubbleID) == "" {
			continue
		}
		var bubble struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(bubbles[header.BubbleID], &bubble) != nil || strings.TrimSpace(bubble.Text) == "" {
			continue
		}
		messages = append(messages, Message{Role: role, Text: bubble.Text, SessionID: composer.ComposerID})
	}
	return sessionID, messages, true
}

// CursorConversationSessionID reads the native composer ID without consulting
// message content or a database key supplied by the caller.
func CursorConversationSessionID(composerJSON []byte) string {
	var header struct {
		ComposerID string `json:"composerId"`
	}
	if json.Unmarshal(composerJSON, &header) != nil {
		return ""
	}
	return strings.TrimSpace(header.ComposerID)
}

func cursorRole(raw json.RawMessage) string {
	var number int
	if json.Unmarshal(raw, &number) == nil {
		switch number {
		case 1:
			return "user"
		case 2:
			return "assistant"
		default:
			return ""
		}
	}
	var name string
	if json.Unmarshal(raw, &name) != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "human", "user":
		return "user"
	case "ai", "assistant":
		return "assistant"
	default:
		return ""
	}
}
