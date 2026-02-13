package settings

import (
	"testing"
)

func TestNormalizeSettingsDefaultsThemeAndFocus(t *testing.T) {
	normalized := normalizeSettings(Settings{Theme: "", CustomFocus: "  focus  "})
	if normalized.Theme != "dark" {
		t.Fatalf("expected default theme dark, got %q", normalized.Theme)
	}
	if normalized.CustomFocus != "focus" {
		t.Fatalf("expected trimmed customFocus, got %q", normalized.CustomFocus)
	}
}
