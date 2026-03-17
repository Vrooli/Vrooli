package main

import "testing"

func TestExtractAssistantText_AssistantOutputText(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"response_item","payload":{"role":"assistant","content":[{"type":"output_text","text":"Hello, world!"}]}}`)
	got := ExtractAssistantText(line)
	if got != "Hello, world!" {
		t.Errorf("expected %q, got %q", "Hello, world!", got)
	}
}

func TestExtractAssistantText_UserResponseItem(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"response_item","payload":{"role":"user","content":[{"type":"output_text","text":"user text"}]}}`)
	got := ExtractAssistantText(line)
	if got != "" {
		t.Errorf("expected empty string for user role, got %q", got)
	}
}

func TestExtractAssistantText_SessionMeta(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"session_meta","payload":{"session_id":"abc123"}}`)
	got := ExtractAssistantText(line)
	if got != "" {
		t.Errorf("expected empty string for session_meta, got %q", got)
	}
}

func TestExtractAssistantText_EventMsg(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"event_msg","payload":{"message":"something happened"}}`)
	got := ExtractAssistantText(line)
	if got != "" {
		t.Errorf("expected empty string for event_msg, got %q", got)
	}
}

func TestExtractAssistantText_MultiContent(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"response_item","payload":{"role":"assistant","content":[{"type":"output_text","text":"Hello"},{"type":"output_text","text":" World"}]}}`)
	got := ExtractAssistantText(line)
	if got != "Hello World" {
		t.Errorf("expected %q, got %q", "Hello World", got)
	}
}

func TestExtractAssistantText_NonOutputTextTypes(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"response_item","payload":{"role":"assistant","content":[{"type":"tool_call","text":"ignored"}]}}`)
	got := ExtractAssistantText(line)
	if got != "" {
		t.Errorf("expected empty string for non-output_text content, got %q", got)
	}
}

func TestExtractAssistantText_EmptyLine(t *testing.T) {
	got := ExtractAssistantText([]byte(""))
	if got != "" {
		t.Errorf("expected empty string for empty input, got %q", got)
	}
}

func TestExtractAssistantText_MalformedJSON(t *testing.T) {
	got := ExtractAssistantText([]byte("{not json"))
	if got != "" {
		t.Errorf("expected empty string for malformed JSON, got %q", got)
	}
}

func TestExtractAssistantText_MixedContentTypes(t *testing.T) {
	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"response_item","payload":{"role":"assistant","content":[{"type":"output_text","text":"visible"},{"type":"tool_use","text":"hidden"},{"type":"output_text","text":" part2"}]}}`)
	got := ExtractAssistantText(line)
	if got != "visible part2" {
		t.Errorf("expected %q, got %q", "visible part2", got)
	}
}
