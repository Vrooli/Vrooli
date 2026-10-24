package store

import (
	"path/filepath"
	"testing"
	"time"
)

func mustAppend(t *testing.T, store *DiscoveryMissStore, miss DiscoveryMiss) {
	t.Helper()
	if err := store.Append(miss); err != nil {
		t.Fatal(err)
	}
}

func newTestMissStore(t *testing.T, now time.Time) *DiscoveryMissStore {
	t.Helper()
	return &DiscoveryMissStore{
		path:       filepath.Join(t.TempDir(), "discovery-misses.jsonl"),
		now:        func() time.Time { return now },
		maxEntries: 3,
		retention:  30 * 24 * time.Hour,
	}
}

func TestDiscoveryMissStoreAppendStampsAndReads(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	store := newTestMissStore(t, now)

	if err := store.Append(DiscoveryMiss{Query: "scrape competitor pricing", Type: "all"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadSince(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 miss, got %d", len(got))
	}
	if got[0].ID == "" || got[0].At == "" {
		t.Fatalf("expected ID and At to be stamped, got %#v", got[0])
	}
}

func TestDiscoveryMissStoreBoundRespected(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	store := newTestMissStore(t, now) // maxEntries = 3
	for i := 0; i < 6; i++ {
		if err := store.Append(DiscoveryMiss{Query: "q", Type: "skill"}); err != nil {
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

func TestDiscoveryMissStorePrunesByRetention(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	store := newTestMissStore(t, now)

	// An entry older than the 30-day retention window is dropped on append.
	old := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	if err := store.Append(DiscoveryMiss{At: old, Query: "stale", Type: "skill"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.ReadSince(0); len(got) != 0 {
		t.Fatalf("expected stale entry to be pruned, got %d", len(got))
	}

	// A fresh entry survives.
	if err := store.Append(DiscoveryMiss{Query: "fresh", Type: "skill"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.ReadSince(0); len(got) != 1 || got[0].Query != "fresh" {
		t.Fatalf("expected only the fresh entry, got %#v", got)
	}
}

func TestDiscoveryMissStoreReadSinceWindow(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	store := newTestMissStore(t, now)
	store.retention = 365 * 24 * time.Hour // keep everything so the window is what filters

	mustAppend(t, store, DiscoveryMiss{At: now.Add(-10 * 24 * time.Hour).Format(time.RFC3339), Query: "older", Type: "skill"})
	mustAppend(t, store, DiscoveryMiss{At: now.Add(-1 * time.Hour).Format(time.RFC3339), Query: "recent", Type: "skill"})

	got, err := store.ReadSince(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Query != "recent" {
		t.Fatalf("expected only the recent entry within 24h, got %#v", got)
	}
}
