package settings

import (
	"testing"
)

func TestNormalizeSettingsDefaultsTheme(t *testing.T) {
	normalized := normalizeSettings(Settings{Theme: ""})
	if normalized.Theme != "dark" {
		t.Fatalf("expected default theme dark, got %q", normalized.Theme)
	}
}
