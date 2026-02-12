package integrations

import (
	"testing"
	"time"
)

func TestTranslateEvent_MessageUser(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-1",
		"eventType": "message",
		"sequence":  float64(1),
		"timestamp": "2025-01-15T10:30:00Z",
		"data": map[string]interface{}{
			"role":    "user",
			"content": "Hello agent",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ID != "evt-1" {
		t.Errorf("expected ID evt-1, got %s", event.ID)
	}
	if event.Type != "message" {
		t.Errorf("expected type message, got %s", event.Type)
	}
	if event.Role != "user" {
		t.Errorf("expected role user, got %s", event.Role)
	}
	if event.Content != "Hello agent" {
		t.Errorf("expected content 'Hello agent', got %s", event.Content)
	}
	if event.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", event.Sequence)
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	if !event.Timestamp.Equal(expectedTime) {
		t.Errorf("expected timestamp %v, got %v", expectedTime, event.Timestamp)
	}
}

func TestTranslateEvent_MessageAssistant(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-2",
		"eventType": "message",
		"sequence":  float64(2),
		"timestamp": "2025-01-15T10:31:00Z",
		"data": map[string]interface{}{
			"role":    "assistant",
			"content": "I'll help you with that",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Role != "assistant" {
		t.Errorf("expected role assistant, got %s", event.Role)
	}
	if event.Content != "I'll help you with that" {
		t.Errorf("unexpected content: %s", event.Content)
	}
}

func TestTranslateEvent_ToolCall(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-3",
		"eventType": "tool_call",
		"sequence":  float64(3),
		"timestamp": "2025-01-15T10:32:00Z",
		"data": map[string]interface{}{
			"toolName": "read_file",
			"input":    map[string]interface{}{"path": "/tmp/test.txt"},
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "tool_call" {
		t.Errorf("expected type tool_call, got %s", event.Type)
	}
	if event.Role != "assistant" {
		t.Errorf("expected role assistant, got %s", event.Role)
	}
	if event.ToolName != "read_file" {
		t.Errorf("expected toolName read_file, got %s", event.ToolName)
	}
	// Input should be JSON-serialized
	if event.ToolInput == "" {
		t.Error("expected non-empty tool input")
	}
	if event.ToolInput != `{"path":"/tmp/test.txt"}` {
		t.Errorf("unexpected tool input: %s", event.ToolInput)
	}
}

func TestTranslateEvent_ToolResultSuccess(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-4",
		"eventType": "tool_result",
		"sequence":  float64(4),
		"timestamp": "2025-01-15T10:33:00Z",
		"data": map[string]interface{}{
			"toolName": "read_file",
			"output":   "file contents here",
			"success":  true,
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "tool_result" {
		t.Errorf("expected type tool_result, got %s", event.Type)
	}
	if event.Role != "tool" {
		t.Errorf("expected role tool, got %s", event.Role)
	}
	if event.ToolOutput != "file contents here" {
		t.Errorf("expected output 'file contents here', got %s", event.ToolOutput)
	}
	if !event.ToolSuccess {
		t.Error("expected tool_success to be true")
	}
}

func TestTranslateEvent_ToolResultError(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-5",
		"eventType": "tool_result",
		"sequence":  float64(5),
		"timestamp": "2025-01-15T10:34:00Z",
		"data": map[string]interface{}{
			"toolName": "read_file",
			"output":   "original output",
			"success":  true,
			"error":    "permission denied",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	// Error field should override output and set success=false
	if event.ToolOutput != "permission denied" {
		t.Errorf("expected error to override output, got %s", event.ToolOutput)
	}
	if event.ToolSuccess {
		t.Error("expected tool_success to be false when error is set")
	}
}

func TestTranslateEvent_Status(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-6",
		"eventType": "status",
		"sequence":  float64(6),
		"timestamp": "2025-01-15T10:35:00Z",
		"data": map[string]interface{}{
			"newStatus": "running",
			"reason":    "Agent started processing",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "status" {
		t.Errorf("expected type status, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.RunStatus != "running" {
		t.Errorf("expected run_status running, got %s", event.RunStatus)
	}
	if event.Content != "Agent started processing" {
		t.Errorf("expected content 'Agent started processing', got %s", event.Content)
	}
}

func TestTranslateEvent_Error(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-7",
		"eventType": "error",
		"sequence":  float64(7),
		"timestamp": "2025-01-15T10:36:00Z",
		"data": map[string]interface{}{
			"message": "something went wrong",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "error" {
		t.Errorf("expected type error, got %s", event.Type)
	}
	if event.Role != "system" {
		t.Errorf("expected role system, got %s", event.Role)
	}
	if event.Content != "something went wrong" {
		t.Errorf("expected content 'something went wrong', got %s", event.Content)
	}
}

func TestTranslateEvent_LogReturnsNil(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-8",
		"eventType": "log",
		"sequence":  float64(8),
		"timestamp": "2025-01-15T10:37:00Z",
		"data": map[string]interface{}{
			"level":   "debug",
			"message": "internal log",
		},
	}

	event := TranslateEvent(raw)
	if event != nil {
		t.Error("expected nil for log event")
	}
}

func TestTranslateEvent_UnknownTypeReturnsNil(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-9",
		"eventType": "heartbeat",
		"sequence":  float64(9),
		"timestamp": "2025-01-15T10:38:00Z",
		"data":      map[string]interface{}{},
	}

	event := TranslateEvent(raw)
	if event != nil {
		t.Error("expected nil for unknown event type")
	}
}

func TestTranslateEvent_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
	}{
		{
			name: "empty map",
			raw:  map[string]interface{}{},
		},
		{
			name: "no data field",
			raw: map[string]interface{}{
				"id":        "evt-10",
				"eventType": "message",
				"sequence":  float64(10),
			},
		},
		{
			name: "nil data",
			raw: map[string]interface{}{
				"id":        "evt-11",
				"eventType": "message",
				"sequence":  float64(11),
				"data":      nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			event := TranslateEvent(tc.raw)
			// Empty/nil eventType defaults to unknown → returns nil,
			// or "message" with nil data is still safe
			_ = event
		})
	}
}

func TestTranslateEvent_EmptyTimestamp(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-12",
		"eventType": "message",
		"sequence":  float64(1),
		"timestamp": "",
		"data": map[string]interface{}{
			"role":    "user",
			"content": "test",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if !event.Timestamp.IsZero() {
		t.Errorf("expected zero timestamp for empty string, got %v", event.Timestamp)
	}
}

func TestTranslateEvent_SequenceConversion(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-13",
		"eventType": "message",
		"sequence":  float64(9999999),
		"timestamp": "2025-01-15T10:30:00Z",
		"data": map[string]interface{}{
			"role":    "user",
			"content": "test",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Sequence != 9999999 {
		t.Errorf("expected sequence 9999999, got %d", event.Sequence)
	}
}

func TestTranslateEvent_ToolCallNoInput(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-14",
		"eventType": "tool_call",
		"sequence":  float64(1),
		"timestamp": "2025-01-15T10:30:00Z",
		"data": map[string]interface{}{
			"toolName": "list_files",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.ToolName != "list_files" {
		t.Errorf("expected toolName list_files, got %s", event.ToolName)
	}
	// No input field → ToolInput stays empty
	if event.ToolInput != "" {
		t.Errorf("expected empty tool input, got %s", event.ToolInput)
	}
}

func TestTranslateEvent_ToolResultEmptyError(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-15",
		"eventType": "tool_result",
		"sequence":  float64(1),
		"timestamp": "2025-01-15T10:30:00Z",
		"data": map[string]interface{}{
			"toolName": "read_file",
			"output":   "file content",
			"success":  true,
			"error":    "", // Empty error string should NOT override
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	// Empty error string should not override output
	if event.ToolOutput != "file content" {
		t.Errorf("expected output 'file content', got %s", event.ToolOutput)
	}
	if !event.ToolSuccess {
		t.Error("expected tool_success to remain true for empty error")
	}
}

func TestTranslateEvent_StatusNoReason(t *testing.T) {
	raw := map[string]interface{}{
		"id":        "evt-16",
		"eventType": "status",
		"sequence":  float64(1),
		"timestamp": "2025-01-15T10:30:00Z",
		"data": map[string]interface{}{
			"newStatus": "complete",
		},
	}

	event := TranslateEvent(raw)
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.RunStatus != "complete" {
		t.Errorf("expected run_status complete, got %s", event.RunStatus)
	}
	if event.Content != "" {
		t.Errorf("expected empty content when no reason, got %s", event.Content)
	}
}
