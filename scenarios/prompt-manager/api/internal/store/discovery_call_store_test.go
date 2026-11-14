package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestCallStore(t *testing.T, now time.Time) *DiscoveryCallStore {
	t.Helper()
	return &DiscoveryCallStore{
		path:       filepath.Join(t.TempDir(), "discovery-calls.jsonl"),
		now:        func() time.Time { return now },
		maxEntries: 3,
		retention:  30 * 24 * time.Hour,
	}
}

func TestDiscoveryCallStoreAppendStampsAndReads(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	store := newTestCallStore(t, now)

	call := DiscoveryCall{
		Queries:           []string{"telemetry", "metrics"},
		Type:              "skill",
		Complexity:        "moderate",
		Threshold:         0.5,
		BudgetChars:       75000,
		TotalContentChars: 12000,
		BudgetStatus:      "under",
		ReturnedCount:     2,
		Results: []DiscoveryCallResult{
			{ID: "a", Score: 0.55, Chars: 6000, Source: "search", Type: "skill"},
			{ID: "b", Score: 0.52, Chars: 6000, Source: "topic", Type: "skill"},
		},
	}
	if err := store.Append(call); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadSince(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 call, got %d", len(got))
	}
	if got[0].ID == "" || got[0].At == "" {
		t.Fatalf("expected ID and At to be stamped, got %#v", got[0])
	}
	if len(got[0].Results) != 2 || got[0].Results[0].ID != "a" {
		t.Fatalf("expected results to round-trip, got %#v", got[0].Results)
	}
}

func TestDiscoveryCallStoreBoundRespected(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	store := newTestCallStore(t, now) // maxEntries = 3
	for i := 0; i < 6; i++ {
		if err := store.Append(DiscoveryCall{Queries: []string{"q"}, Type: "skill"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.ReadSince(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected bound of 3 entries, got %d", len(got))
	}
}

func TestDiscoveryCallStorePrunesByRetention(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	store := newTestCallStore(t, now)

	old := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	if err := store.Append(DiscoveryCall{At: old, Queries: []string{"stale"}, Type: "skill"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.ReadSince(0); len(got) != 0 {
		t.Fatalf("expected stale entry to be pruned, got %d", len(got))
	}

	if err := store.Append(DiscoveryCall{Queries: []string{"fresh"}, Type: "skill"}); err != nil {
		t.Fatal(err)
	}
	got, _ := store.ReadSince(0)
	if len(got) != 1 || len(got[0].Queries) != 1 || got[0].Queries[0] != "fresh" {
		t.Fatalf("expected only the fresh entry, got %#v", got)
	}
}

func TestDiscoveryCallStoreReadSinceWindow(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	store := newTestCallStore(t, now)
	store.retention = 365 * 24 * time.Hour // keep everything so the window is what filters

	if err := store.Append(DiscoveryCall{At: now.Add(-10 * 24 * time.Hour).Format(time.RFC3339), Queries: []string{"older"}, Type: "skill"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(DiscoveryCall{At: now.Add(-1 * time.Hour).Format(time.RFC3339), Queries: []string{"recent"}, Type: "skill"}); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadSince(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Queries[0] != "recent" {
		t.Fatalf("expected only the recent entry within 24h, got %#v", got)
	}
}
