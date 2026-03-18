package services

import (
	"testing"
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// TestContainsString verifies slice membership check.
func TestContainsString(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
		{[]string{"completed", "succeeded"}, "completed", true},
		{[]string{"failed", "error"}, "timeout", false},
	}

	for _, tc := range tests {
		t.Run(tc.item, func(t *testing.T) {
			result := ContainsString(tc.slice, tc.item)
			if result != tc.expected {
				t.Errorf("ContainsString(%v, %q) = %v, want %v", tc.slice, tc.item, result, tc.expected)
			}
		})
	}
}

// TestExtractField verifies nested field extraction.
func TestExtractField(t *testing.T) {
	data := map[string]interface{}{
		"status": "completed",
		"count":  42.0,
		"nested": map[string]interface{}{
			"value":  "inner",
			"number": 100.0,
			"deep": map[string]interface{}{
				"item": "deepest",
			},
		},
	}

	tests := []struct {
		path     string
		expected interface{}
	}{
		{"status", "completed"},
		{"count", 42.0},
		{"nested.value", "inner"},
		{"nested.number", 100.0},
		{"nested.deep.item", "deepest"},
		{"nonexistent", nil},
		{"nested.nonexistent", nil},
		{"nested.deep.nonexistent", nil},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := ExtractField(data, tc.path)
			if result != tc.expected {
				t.Errorf("ExtractField(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

// TestExtractStringField verifies string extraction.
func TestExtractStringField(t *testing.T) {
	data := map[string]interface{}{
		"status":  "completed",
		"number":  42.0,
		"boolean": true,
		"nested": map[string]interface{}{
			"message": "hello",
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"status", "completed"},
		{"number", ""},
		{"boolean", ""},
		{"nonexistent", ""},
		{"nested.message", "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := ExtractStringField(data, tc.path)
			if result != tc.expected {
				t.Errorf("ExtractStringField(%q) = %q, want %q", tc.path, result, tc.expected)
			}
		})
	}
}

// TestExtractIntField verifies int extraction from various numeric types.
func TestExtractIntField(t *testing.T) {
	data := map[string]interface{}{
		"float":   42.0,
		"int":     100,
		"int64":   int64(200),
		"string":  "not a number",
		"boolean": true,
		"nested": map[string]interface{}{
			"progress": 75.0,
		},
	}

	tests := []struct {
		path     string
		expected *int
	}{
		{"float", intPtr(42)},
		{"int", intPtr(100)},
		{"int64", intPtr(200)},
		{"string", nil},
		{"boolean", nil},
		{"nonexistent", nil},
		{"nested.progress", intPtr(75)},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := ExtractIntField(data, tc.path)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("ExtractIntField(%q) = %v, want nil", tc.path, *result)
				}
			} else {
				if result == nil {
					t.Errorf("ExtractIntField(%q) = nil, want %d", tc.path, *tc.expected)
				} else if *result != *tc.expected {
					t.Errorf("ExtractIntField(%q) = %d, want %d", tc.path, *result, *tc.expected)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

// TestProcessStatusResult_SuccessCompletion verifies success detection.
func TestProcessStatusResult_SuccessCompletion(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	sub := svc.SubscribeWithID("chat-1")
	defer svc.UnsubscribeByID(sub)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		ToolName:   "test_tool",
		Status:     "running",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &toolspb.AsyncBehavior{
			CompletionConditions: &toolspb.CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed", "succeeded"},
				FailureValues: []string{"failed", "error"},
			},
		},
	}
	svc.AddTestOperation(op)

	result := map[string]interface{}{
		"status": "completed",
	}

	conditions := op.AsyncBehavior.CompletionConditions
	isTerminal, status := svc.processStatusResult(op, result, conditions)

	if !isTerminal {
		t.Error("expected terminal to be true for success")
	}
	if status != "completed" {
		t.Errorf("expected status 'completed', got %q", status)
	}
	if op.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

// TestProcessStatusResult_FailureCompletion verifies failure detection.
func TestProcessStatusResult_FailureCompletion(t *testing.T) {
	svc := NewAsyncTrackerService(nil, nil, nil)

	op := &AsyncOperation{
		ToolCallID: "tc-1",
		ChatID:     "chat-1",
		Status:     "running",
		UpdatedAt:  time.Now(),
		AsyncBehavior: &toolspb.AsyncBehavior{
			CompletionConditions: &toolspb.CompletionConditions{
				StatusField:   "status",
				SuccessValues: []string{"completed"},
				FailureValues: []string{"failed", "error"},
				ErrorField:    "error_message",
			},
		},
	}
	svc.AddTestOperation(op)

	result := map[string]interface{}{
		"status":        "failed",
		"error_message": "something went wrong",
	}

	conditions := op.AsyncBehavior.CompletionConditions
	isTerminal, status := svc.processStatusResult(op, result, conditions)

	if !isTerminal {
		t.Error("expected terminal to be true for failure")
	}
	if status != "failed" {
		t.Errorf("expected status 'failed', got %q", status)
	}
	if op.Error != "something went wrong" {
		t.Errorf("expected error message, got %q", op.Error)
	}
}
