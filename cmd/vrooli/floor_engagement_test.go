package main

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/baselinefloor"
)

func writeFloorEngagement(t *testing.T, store *baselinefloor.Store, scenario, slug string, mode baselinefloor.Mode) {
	t.Helper()
	when := time.Now().UTC()
	if err := store.WriteManifest(baselinefloor.Manifest{Scenario: scenario, Slug: slug, Mode: mode, RestorePointPath: store.RestorePointPath(scenario, slug), CreatedAt: when, LastTouchedAt: when}); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestFloorEngagementResolverShadowAndLive(t *testing.T) {
	store := baselinefloor.NewStore(t.TempDir())
	writeFloorEngagement(t, store, "demo", "abc", baselinefloor.ModeShadow)
	resolver := newFloorEngagementResolverWithStore(store)
	info, engaged, err := resolver.Engagement("demo")
	if err != nil || !engaged || info.RestorePointDir != store.RestorePointPath("demo", "abc") {
		t.Fatalf("shadow engagement = %+v, %v, %t", info, err, engaged)
	}

	liveStore := baselinefloor.NewStore(t.TempDir())
	writeFloorEngagement(t, liveStore, "demo", "abc", baselinefloor.ModeLive)
	_, engaged, err = newFloorEngagementResolverWithStore(liveStore).Engagement("demo")
	if err != nil || engaged {
		t.Fatalf("live engagement = engaged %t, err %v; want unsplit", engaged, err)
	}
}

func TestFloorEngagementResolverRejectsDuplicates(t *testing.T) {
	store := baselinefloor.NewStore(t.TempDir())
	writeFloorEngagement(t, store, "demo", "abc", baselinefloor.ModeShadow)
	writeFloorEngagement(t, store, "demo", "def", baselinefloor.ModeShadow)
	if _, _, err := newFloorEngagementResolverWithStore(store).Engagement("demo"); err == nil {
		t.Fatal("expected duplicate shadow engagement error")
	}
}
