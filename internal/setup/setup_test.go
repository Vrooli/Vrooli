package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-10

func TestParseOptionsAcceptsSetupFlags(t *testing.T) {
	opts, err := parseOptions("setup", []string{"--environment", "minimal", "--resources", "none", "--yes", "yes", "--sudo-mode", "skip", "--dry-run"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opts.Environment != "minimal" || opts.Resources != "none" || opts.Yes != "yes" || opts.SudoMode != "skip" || !opts.DryRun {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestApplyEnvironmentSetsDefaultsAndRestoresState(t *testing.T) {
	t.Setenv("TARGET", "")
	t.Setenv("LOCATION", "")
	root := t.TempDir()
	restore, err := applyEnvironment(root, filepath.Join(root, ".vrooli", "service.json"), options{
		Environment: "production",
		Resources:   "none",
		Yes:         "yes",
		SudoMode:    "skip",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("applyEnvironment: %v", err)
	}

	if got := os.Getenv("APP_ROOT"); got != root {
		t.Fatalf("APP_ROOT = %q", got)
	}
	if got := os.Getenv("TARGET"); got != defaultTarget {
		t.Fatalf("TARGET = %q", got)
	}
	if got := os.Getenv("LOCATION"); got != defaultLocation {
		t.Fatalf("LOCATION = %q", got)
	}
	if got := os.Getenv("ENVIRONMENT"); got != "production" {
		t.Fatalf("ENVIRONMENT = %q", got)
	}
	if got := os.Getenv("RESOURCES"); got != "none" {
		t.Fatalf("RESOURCES = %q", got)
	}
	if got := os.Getenv("YES"); got != "yes" {
		t.Fatalf("YES = %q", got)
	}
	if got := os.Getenv("SUDO_MODE"); got != "skip" {
		t.Fatalf("SUDO_MODE = %q", got)
	}
	if got := os.Getenv("DRY_RUN"); got != "true" {
		t.Fatalf("DRY_RUN = %q", got)
	}

	restore()

	if got := os.Getenv("APP_ROOT"); got != "" {
		t.Fatalf("APP_ROOT after restore = %q", got)
	}
	if got := os.Getenv("TARGET"); got != "" {
		t.Fatalf("TARGET after restore = %q", got)
	}
}

func TestMarkCompleteWritesSetupAndResourceMarkers(t *testing.T) {
	root := t.TempDir()
	manifest := scenario.ServiceManifest{
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "base-setup"},
					{Name: "add-data"},
				},
			},
		},
	}

	if err := markComplete(root, manifest); err != nil {
		t.Fatalf("markComplete: %v", err)
	}

	setupMarker := filepath.Join(root, "data", ".setup-complete")
	data, err := os.ReadFile(setupMarker)
	if err != nil {
		t.Fatalf("read setup marker: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal setup marker: %v", err)
	}
	if payload["setup_version"] != "2.0.0" {
		t.Fatalf("setup_version = %v", payload["setup_version"])
	}
	if _, err := os.Stat(filepath.Join(root, "data", ".resources-populated")); err != nil {
		t.Fatalf("expected resources marker: %v", err)
	}
}
