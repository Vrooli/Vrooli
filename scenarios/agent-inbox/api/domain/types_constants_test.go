package domain

import (
	"testing"
)

// =============================================================================
// TruncatePreview Tests
// =============================================================================

func TestTruncatePreview(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "short string",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "exactly at limit",
			input:    string(make([]byte, PreviewMaxLength)),
			expected: string(make([]byte, PreviewMaxLength)),
		},
		{
			name:     "one over limit",
			input:    string(make([]byte, PreviewMaxLength+1)),
			expected: string(make([]byte, PreviewMaxLength)) + "...",
		},
		{
			name:     "much longer",
			input:    string(make([]byte, 500)),
			expected: string(make([]byte, PreviewMaxLength)) + "...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TruncatePreview(tc.input); got != tc.expected {
				t.Errorf("TruncatePreview() length = %d, want %d", len(got), len(tc.expected))
			}
		})
	}
}

func TestTruncatePreview_WithRealText(t *testing.T) {
	// Test with actual readable text
	shortText := "This is a short message."
	if got := TruncatePreview(shortText); got != shortText {
		t.Errorf("short text should not be truncated: %q", got)
	}

	// Create text slightly over the limit
	longText := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam."
	result := TruncatePreview(longText)

	if len(result) != PreviewMaxLength+3 { // +3 for "..."
		t.Errorf("truncated length = %d, want %d", len(result), PreviewMaxLength+3)
	}
	if result[len(result)-3:] != "..." {
		t.Errorf("should end with '...', got %q", result[len(result)-3:])
	}
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestStatusConstants(t *testing.T) {
	// Verify status constants have expected values
	statuses := map[string]string{
		"StatusPending":         StatusPending,
		"StatusPendingApproval": StatusPendingApproval,
		"StatusApproved":        StatusApproved,
		"StatusRejected":        StatusRejected,
		"StatusRunning":         StatusRunning,
		"StatusCompleted":       StatusCompleted,
		"StatusFailed":          StatusFailed,
		"StatusCancelled":       StatusCancelled,
	}

	expectedValues := map[string]string{
		"StatusPending":         "pending",
		"StatusPendingApproval": "pending_approval",
		"StatusApproved":        "approved",
		"StatusRejected":        "rejected",
		"StatusRunning":         "running",
		"StatusCompleted":       "completed",
		"StatusFailed":          "failed",
		"StatusCancelled":       "cancelled",
	}

	for name, got := range statuses {
		want := expectedValues[name]
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestPreviewMaxLength(t *testing.T) {
	if PreviewMaxLength != 100 {
		t.Errorf("PreviewMaxLength = %d, want 100", PreviewMaxLength)
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("RoleUser = %q", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q", RoleAssistant)
	}
	if RoleSystem != "system" {
		t.Errorf("RoleSystem = %q", RoleSystem)
	}
	if RoleTool != "tool" {
		t.Errorf("RoleTool = %q", RoleTool)
	}
}

func TestViewModeConstants(t *testing.T) {
	if ViewModeBubble != "bubble" {
		t.Errorf("ViewModeBubble = %q", ViewModeBubble)
	}
	if ViewModeCompact != "compact" {
		t.Errorf("ViewModeCompact = %q", ViewModeCompact)
	}
}
