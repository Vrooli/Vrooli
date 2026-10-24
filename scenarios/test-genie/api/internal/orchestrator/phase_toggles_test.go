package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPhaseToggleStoreSaveAndLoadUsesStorageConfigRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_CONFIG_ROOT", root)

	store := newPhaseToggleStore()
	if store == nil {
		t.Fatal("newPhaseToggleStore() returned nil")
	}

	want := PhaseToggleConfig{
		Phases: map[string]PhaseToggle{
			"unit": {
				Disabled: true,
				Reason:   "maintenance window",
				Owner:    "platform",
			},
			"quality": {
				Disabled: false,
			},
		},
	}

	saved, err := store.Save(want)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !saved.Phases["unit"].Disabled {
		t.Fatalf("saved unit toggle should be disabled")
	}
	if saved.Phases["unit"].AddedAt.IsZero() {
		t.Fatalf("saved unit toggle should record AddedAt")
	}
	if !saved.Phases["quality"].AddedAt.IsZero() {
		t.Fatalf("enabled quality toggle should not record AddedAt")
	}

	path := filepath.Join(root, "vrooli", phaseToggleScenarioID, phaseToggleFilename)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected toggle file at %s: %v", path, err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded.Phases["unit"].Disabled {
		t.Fatalf("loaded unit toggle should be disabled")
	}
	if loaded.Phases["unit"].Reason != "maintenance window" {
		t.Fatalf("loaded unit reason = %q", loaded.Phases["unit"].Reason)
	}
	if loaded.Phases["unit"].Owner != "platform" {
		t.Fatalf("loaded unit owner = %q", loaded.Phases["unit"].Owner)
	}
	if loaded.Phases["unit"].AddedAt.IsZero() {
		t.Fatalf("loaded unit toggle should preserve AddedAt")
	}
}

func TestNormalizePhaseToggleConfigClearsAddedAtForEnabledPhases(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	cfg := PhaseToggleConfig{
		Phases: map[string]PhaseToggle{
			" Unit ": {
				Disabled: false,
				AddedAt:  now,
				Reason:   "  stale  ",
				Owner:    "  ops  ",
			},
		},
	}

	got := normalizePhaseToggleConfig(cfg, now)
	toggle, ok := got.Phases["unit"]
	if !ok {
		t.Fatalf("expected normalized unit phase")
	}
	if !toggle.AddedAt.IsZero() {
		t.Fatalf("enabled phase should clear AddedAt")
	}
	if toggle.Reason != "stale" {
		t.Fatalf("Reason = %q, want stale", toggle.Reason)
	}
	if toggle.Owner != "ops" {
		t.Fatalf("Owner = %q, want ops", toggle.Owner)
	}
}
