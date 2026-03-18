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
