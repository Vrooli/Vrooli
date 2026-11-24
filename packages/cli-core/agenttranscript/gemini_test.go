package agenttranscript

import (
	"testing"
	"time"
)

func TestParseGeminiCLIConversationDocument(t *testing.T) {
	data := []byte(`{"sessionId":"session-safe","messages":[{"id":"u1","timestamp":"2025-10-17T14:27:00Z","type":"user","content":"question "},{"id":"i1","type":"info","content":"metadata"},{"id":"a1","type":"gemini","content":"answer"}]}`)
	got := ParseConversation("gemini-cli", data)
	if len(got) != 2 || got[0].Role != "user" || got[0].Text != "question " || got[0].SessionID != "session-safe" || !got[0].OccurredAt.Equal(time.Date(2025, 10, 17, 14, 27, 0, 0, time.UTC)) || got[1] != (Message{Role: "assistant", Text: "answer", SessionID: "session-safe"}) {
		t.Fatalf("ParseConversation(gemini-cli) = %#v", got)
	}
	if got := ParseConversation("gemini-cli", []byte(`{"messages":[`)); len(got) != 0 {
		t.Fatalf("malformed conversation returned %#v", got)
	}
}

func TestGeminiOccurrenceTimeKeepsPrecisionAndUnknownValuesZero(t *testing.T) {
	doc, ok := ParseGeminiConversationDocument([]byte(`{"sessionId":"s","messages":[{"type":"user","content":"a","timestamp":"2025-10-17T14:27:00.123456789Z"},{"type":"gemini","content":"b","timestamp":"bad"},{"type":"user","content":"c"}]}`))
	if !ok || len(doc.Messages) != 3 {
		t.Fatalf("parse result = %#v, %v", doc, ok)
	}
	want := time.Date(2025, 10, 17, 14, 27, 0, 123456789, time.UTC)
	if !doc.Messages[0].OccurredAt.Equal(want) || !doc.Messages[1].OccurredAt.IsZero() || !doc.Messages[2].OccurredAt.IsZero() {
		t.Fatalf("Gemini occurrence timestamps = %#v", doc.Messages)
	}
}

func TestParseGeminiConversationDocumentReturnsIdentityAndMessagesTogether(t *testing.T) {
	data := []byte(`{"sessionId":" session-safe ","messages":[{"type":"user","content":"question"},{"type":"info","content":"skip"},{"type":"gemini","content":"answer"}]}`)
	document, ok := ParseGeminiConversationDocument(data)
	if !ok {
		t.Fatal("valid Gemini document was rejected")
	}
	if document.SessionID != "session-safe" || GeminiConversationSessionID(data) != document.SessionID {
		t.Fatalf("native identity = %q, wrapper = %q", document.SessionID, GeminiConversationSessionID(data))
	}
	if len(document.Messages) != 2 || document.Messages[0] != (Message{Role: "user", Text: "question", SessionID: " session-safe "}) || document.Messages[1] != (Message{Role: "assistant", Text: "answer", SessionID: " session-safe "}) {
		t.Fatalf("combined messages = %#v", document.Messages)
	}
}

func TestParseGeminiConversationDocumentDistinguishesMalformedAndMissingIdentity(t *testing.T) {
	if document, ok := ParseGeminiConversationDocument([]byte(`{"messages":[`)); ok || document.SessionID != "" || document.Messages != nil {
		t.Fatalf("malformed document = %#v, %v", document, ok)
	}
	document, ok := ParseGeminiConversationDocument([]byte(`{"messages":[{"type":"user","content":"question"}]}`))
	if !ok || document.SessionID != "" || len(document.Messages) != 1 {
		t.Fatalf("valid document without identity = %#v, %v", document, ok)
	}
}
