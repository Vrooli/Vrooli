package settings

import (
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
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

func TestHasDeprecatedRecommendationFields(t *testing.T) {
	mode := "yolo"
	req := &apipb.UpdateSettingsRequest{
		RecommendationMode: &mode,
	}
	if !hasDeprecatedRecommendationFields(req) {
		t.Fatalf("expected deprecated recommendation fields to be detected")
	}
}
