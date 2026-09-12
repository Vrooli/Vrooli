package execute

import (
	"strings"
	"testing"
)

// extractErrorMessage backs the PreviewPlan error path (the execute run itself
// now flows through the durable runs RPCs, not the blocking REST adapter). These
// tests pin the message extraction that surfaces a scenario API error to the user.

func TestExtractErrorMessage_PrefersTypedEnvelope(t *testing.T) {
	msg := extractErrorMessage([]byte(`{"error":{"message":"boom"}}`))
	if msg != "boom" {
		t.Fatalf("expected typed envelope message, got %q", msg)
	}
}

func TestExtractErrorMessage_FallsBackToRawBody(t *testing.T) {
	msg := extractErrorMessage([]byte("plain text failure"))
	if !strings.Contains(msg, "plain text failure") {
		t.Fatalf("expected raw body fallback, got %q", msg)
	}
}

func TestExtractErrorMessage_DeduplicatesParts(t *testing.T) {
	msg := extractErrorMessage([]byte(`{"error":"dup","message":"dup"}`))
	if msg != "dup" {
		t.Fatalf("expected deduplicated single message, got %q", msg)
	}
}
