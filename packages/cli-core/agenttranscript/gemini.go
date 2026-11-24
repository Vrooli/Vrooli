package agenttranscript

import (
	"encoding/json"
	"strings"
)

// GeminiConversationDocument is the shared decode of one complete native
// Gemini CLI chat archive. SessionID is normalized for identity comparisons;
// each Message retains the native sessionId value used by ParseConversation.
type GeminiConversationDocument struct {
	SessionID string
	Messages  []Message
}

// ParseGeminiConversationDocument decodes one complete native archive once,
// returning both its identity and searchable messages. A valid document may
// have an empty session ID; callers decide whether that identity is required.
func ParseGeminiConversationDocument(data []byte) (GeminiConversationDocument, bool) {
	var transcript struct {
		SessionID string `json:"sessionId"`
		Messages  []struct {
			Type      string          `json:"type"`
			Content   string          `json:"content"`
			Timestamp json.RawMessage `json:"timestamp"`
		} `json:"messages"`
	}
	if json.Unmarshal(data, &transcript) != nil {
		return GeminiConversationDocument{}, false
	}
	document := GeminiConversationDocument{
		SessionID: strings.TrimSpace(transcript.SessionID),
		Messages:  make([]Message, 0, len(transcript.Messages)),
	}
	for _, message := range transcript.Messages {
		role := ""
		switch message.Type {
		case "user":
			role = "user"
		case "gemini":
			role = "assistant"
		default:
			continue
		}
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		occurredAt := ParseTimestampValue(message.Timestamp)
		document.Messages = append(document.Messages, Message{Role: role, Text: message.Content, SessionID: transcript.SessionID, OccurredAt: occurredAt})
	}
	return document, true
}

// ParseConversation reads one complete native conversation document. Gemini
// CLI stores a JSON document containing a messages array rather than JSONL.
// The caller owns the input bytes; this function does not retain content.
func ParseConversation(harness string, data []byte) []Message {
	format, ok := FormatFor(harness)
	if !ok || format.MessageDialect != "gemini-cli-chat" {
		return nil
	}
	document, ok := ParseGeminiConversationDocument(data)
	if !ok {
		return nil
	}
	return document.Messages
}

// GeminiConversationSessionID returns the native ID from the archive header.
// It accepts the complete document and never derives identity from message
// content or a path supplied by the caller.
func GeminiConversationSessionID(data []byte) string {
	var header struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(data, &header) != nil {
		return ""
	}
	return strings.TrimSpace(header.SessionID)
}
