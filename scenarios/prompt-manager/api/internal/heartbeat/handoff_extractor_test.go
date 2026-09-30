package heartbeat

import (
	"context"
	"encoding/json"
	"testing"
)

func makeEventsJSON(t *testing.T, events []runEvent) []byte {
	t.Helper()
	envelope := eventsEnvelope{Events: events}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSentinelExtractor_HappyPath(t *testing.T) {
	events := []runEvent{
		{Message: eventMessage{Role: "user", Content: "Do stuff"}},
		{Message: eventMessage{Role: "assistant", Content: "I did stuff.\n\n## HANDOFF\n\n**Status**: Completed\n\n**Completed this heartbeat**:\n- Did the thing\n"}},
	}
	ext := NewSentinelExtractor()
	result, err := ext.Extract(context.Background(), makeEventsJSON(t, events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty handoff")
	}
	if !containsString(result, "Status") || !containsString(result, "Completed") {
		t.Errorf("handoff content missing expected text: %s", result)
	}
}

func TestSentinelExtractor_EmojiHeader(t *testing.T) {
	events := []runEvent{
		{Message: eventMessage{Role: "assistant", Content: "Work done.\n\n## 🔄 HANDOFF\n\n**Status**: In progress\n"}},
	}
	ext := NewSentinelExtractor()
	result, err := ext.Extract(context.Background(), makeEventsJSON(t, events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty handoff with emoji header")
	}
	if !containsString(result, "In progress") {
		t.Errorf("expected 'In progress' in result: %s", result)
	}
}

func TestSentinelExtractor_CaseInsensitive(t *testing.T) {
	for _, header := range []string{"## Handoff", "## handoff", "## HANDOFF"} {
		events := []runEvent{
			{Message: eventMessage{Role: "assistant", Content: "Done.\n\n" + header + "\n\n**Status**: OK\n"}},
		}
		ext := NewSentinelExtractor()
		result, err := ext.Extract(context.Background(), makeEventsJSON(t, events))
		if err != nil {
			t.Fatalf("header %q: unexpected error: %v", header, err)
		}
		if result == "" {
			t.Fatalf("header %q: expected non-empty handoff", header)
		}
	}
}

func TestSentinelExtractor_NoHandoff(t *testing.T) {
	events := []runEvent{
		{Message: eventMessage{Role: "assistant", Content: "I did things but no handoff section."}},
	}
	ext := NewSentinelExtractor()
	result, err := ext.Extract(context.Background(), makeEventsJSON(t, events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got: %s", result)
	}
}

func TestSentinelExtractor_NoAssistantMessages(t *testing.T) {
	events := []runEvent{
		{Message: eventMessage{Role: "user", Content: "Hello"}},
		{Message: eventMessage{Role: "system", Content: "Config"}},
	}
	ext := NewSentinelExtractor()
	result, err := ext.Extract(context.Background(), makeEventsJSON(t, events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got: %s", result)
	}
}

func TestSentinelExtractor_MalformedJSON(t *testing.T) {
	ext := NewSentinelExtractor()
	_, err := ext.Extract(context.Background(), []byte("{invalid"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestSentinelExtractor_HandoffFollowedBySection(t *testing.T) {
	events := []runEvent{
		{Message: eventMessage{Role: "assistant", Content: "Work.\n\n## HANDOFF\n\n**Status**: Done\n\n**Completed**:\n- X\n\n## OTHER SECTION\n\nThis should not be included.\n"}},
	}
	ext := NewSentinelExtractor()
	result, err := ext.Extract(context.Background(), makeEventsJSON(t, events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containsString(result, "OTHER SECTION") || containsString(result, "should not be included") {
		t.Errorf("handoff should not include content after next ## section: %s", result)
	}
	if !containsString(result, "Status") {
		t.Errorf("handoff should include status: %s", result)
	}
}

func TestChainExtractor_FirstEmptySecondReturns(t *testing.T) {
	// First extractor always returns empty
	empty := &stubExtractor{result: ""}
	// Second extractor returns content
	withContent := &stubExtractor{result: "handoff content"}

	chain := NewChainExtractor(empty, withContent)
	result, err := chain.Extract(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "handoff content" {
		t.Errorf("expected 'handoff content', got: %s", result)
	}
	if !withContent.called {
		t.Error("second extractor should have been called")
	}
}

func TestChainExtractor_FirstReturnsContent(t *testing.T) {
	first := &stubExtractor{result: "from first"}
	second := &stubExtractor{result: "from second"}

	chain := NewChainExtractor(first, second)
	result, err := chain.Extract(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "from first" {
		t.Errorf("expected 'from first', got: %s", result)
	}
	if second.called {
		t.Error("second extractor should NOT have been called")
	}
}

// stubExtractor is a test helper that returns a fixed result.
type stubExtractor struct {
	result string
	called bool
}

func (s *stubExtractor) Extract(_ context.Context, _ []byte) (string, error) {
	s.called = true
	return s.result, nil
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
