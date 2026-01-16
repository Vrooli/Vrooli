package domain

import (
	"testing"
)

// =============================================================================
// Attachment Tests
// =============================================================================

func TestAttachment_IsImage(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"jpeg", "image/jpeg", true},
		{"png", "image/png", true},
		{"gif", "image/gif", true},
		{"webp", "image/webp", true},
		{"pdf", "application/pdf", false},
		{"json", "application/json", false},
		{"text", "text/plain", false},
		{"empty", "", false},
		{"svg", "image/svg+xml", false}, // SVG not in allowed list
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := Attachment{ContentType: tc.contentType}
			if got := a.IsImage(); got != tc.expected {
				t.Errorf("IsImage() = %v, want %v for %q", got, tc.expected, tc.contentType)
			}
		})
	}
}

func TestAttachment_IsPDF(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"pdf", "application/pdf", true},
		{"jpeg", "image/jpeg", false},
		{"json", "application/json", false},
		{"empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := Attachment{ContentType: tc.contentType}
			if got := a.IsPDF(); got != tc.expected {
				t.Errorf("IsPDF() = %v, want %v for %q", got, tc.expected, tc.contentType)
			}
		})
	}
}

// =============================================================================
// ViewMode Tests
// =============================================================================

func TestValidViewModes(t *testing.T) {
	modes := ValidViewModes()

	if len(modes) != 2 {
		t.Fatalf("expected 2 view modes, got %d", len(modes))
	}

	// Check both modes are present
	hasBubble := false
	hasCompact := false
	for _, m := range modes {
		if m == ViewModeBubble {
			hasBubble = true
		}
		if m == ViewModeCompact {
			hasCompact = true
		}
	}

	if !hasBubble {
		t.Error("ValidViewModes() missing 'bubble'")
	}
	if !hasCompact {
		t.Error("ValidViewModes() missing 'compact'")
	}
}

func TestIsValidViewMode(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{ViewModeBubble, true},
		{ViewModeCompact, true},
		{"bubble", true},
		{"compact", true},
		{"", false},
		{"invalid", false},
		{"BUBBLE", false}, // Case sensitive
		{"Compact", false},
	}

	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			if got := IsValidViewMode(tc.mode); got != tc.expected {
				t.Errorf("IsValidViewMode(%q) = %v, want %v", tc.mode, got, tc.expected)
			}
		})
	}
}

// =============================================================================
// Role Tests
// =============================================================================

func TestValidRoles(t *testing.T) {
	roles := ValidRoles()

	if len(roles) != 4 {
		t.Fatalf("expected 4 roles, got %d", len(roles))
	}

	expected := map[string]bool{
		RoleUser:      false,
		RoleAssistant: false,
		RoleSystem:    false,
		RoleTool:      false,
	}

	for _, r := range roles {
		if _, ok := expected[r]; !ok {
			t.Errorf("unexpected role: %q", r)
		}
		expected[r] = true
	}

	for role, found := range expected {
		if !found {
			t.Errorf("missing role: %q", role)
		}
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{RoleUser, true},
		{RoleAssistant, true},
		{RoleSystem, true},
		{RoleTool, true},
		{"user", true},
		{"assistant", true},
		{"system", true},
		{"tool", true},
		{"", false},
		{"invalid", false},
		{"USER", false}, // Case sensitive
		{"Admin", false},
	}

	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			if got := IsValidRole(tc.role); got != tc.expected {
				t.Errorf("IsValidRole(%q) = %v, want %v", tc.role, got, tc.expected)
			}
		})
	}
}

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
