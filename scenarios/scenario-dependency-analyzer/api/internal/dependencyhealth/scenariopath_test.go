package dependencyhealth

import (
	"context"
	"path/filepath"
	"testing"
)

// TestScenarioDir proves the evaluation stages resolve an explicit scenario path
// (threaded via withScenarioPath) when present — letting deep template
// validation point SDA at a temp scenario outside the repo scenarios/ tree — and
// fall back to <scenariosDir>/<scenario> otherwise.
func TestScenarioDir(t *testing.T) {
	h := &connectHandler{scenariosDir: func() string { return "/repo/scenarios" }}

	t.Run("falls back to scenarios dir by name", func(t *testing.T) {
		got := h.scenarioDir(context.Background(), "demo")
		if want := filepath.Join("/repo/scenarios", "demo"); got != want {
			t.Fatalf("scenarioDir = %q, want %q", got, want)
		}
	})

	t.Run("prefers explicit path from ctx", func(t *testing.T) {
		explicit := "/tmp/vrooli-template-deep-123/scenarios/demo"
		ctx := withScenarioPath(context.Background(), explicit)
		if got := h.scenarioDir(ctx, "demo"); got != explicit {
			t.Fatalf("scenarioDir = %q, want %q", got, explicit)
		}
	})

	t.Run("blank explicit path is ignored", func(t *testing.T) {
		ctx := withScenarioPath(context.Background(), "   ")
		if got := h.scenarioDir(ctx, "demo"); got != filepath.Join("/repo/scenarios", "demo") {
			t.Fatalf("blank path should fall back, got %q", got)
		}
	})
}
