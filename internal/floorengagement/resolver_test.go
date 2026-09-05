package floorengagement

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/baselinefloor"
)

func writeEngagement(t *testing.T, store *baselinefloor.Store, scenario, slug string, mode baselinefloor.Mode) {
	t.Helper()
	m := baselinefloor.Manifest{
		Scenario:         scenario,
		Slug:             slug,
		Mode:             mode,
		RestorePointPath: store.RestorePointPath(scenario, slug),
		CreatedAt:        time.Now().UTC(),
		LastTouchedAt:    time.Now().UTC(),
	}
	if err := store.WriteManifest(m); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestResolverShadowEngagementSplits(t *testing.T) {
	store := baselinefloor.NewStore(t.TempDir())
	writeEngagement(t, store, "demo", "abc", baselinefloor.ModeShadow)
	r := NewWithStore(store)

	info, engaged, err := r.Engagement("demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !engaged {
		t.Fatal("expected shadow engagement to report engaged")
	}
	if want := store.RestorePointPath("demo", "abc"); info.RestorePointDir != want {
		t.Errorf("RestorePointDir = %q, want %q", info.RestorePointDir, want)
	}
	if info.Slug != "abc" {
		t.Errorf("Slug = %q, want abc", info.Slug)
	}
}

func TestResolverLiveEngagementDoesNotSplit(t *testing.T) {
	store := baselinefloor.NewStore(t.TempDir())
	writeEngagement(t, store, "demo", "abc", baselinefloor.ModeLive)
	r := NewWithStore(store)

	_, engaged, err := r.Engagement("demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if engaged {
		t.Fatal("live-mode engagement must not report a source-dir split")
	}
}

func TestResolverNoEngagement(t *testing.T) {
	store := baselinefloor.NewStore(t.TempDir())
	writeEngagement(t, store, "other", "abc", baselinefloor.ModeShadow)
	r := NewWithStore(store)

	_, engaged, err := r.Engagement("demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if engaged {
		t.Fatal("scenario with no engagement must report not engaged")
	}
}

func TestResolverDuplicateShadowErrors(t *testing.T) {
	store := baselinefloor.NewStore(t.TempDir())
	writeEngagement(t, store, "demo", "abc", baselinefloor.ModeShadow)
	writeEngagement(t, store, "demo", "def", baselinefloor.ModeShadow)
	r := NewWithStore(store)

	if _, _, err := r.Engagement("demo"); err == nil {
		t.Fatal("expected error for two open shadow engagements on one scenario")
	}
}
