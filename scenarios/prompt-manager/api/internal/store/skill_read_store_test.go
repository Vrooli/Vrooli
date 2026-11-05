package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSkillReadStoreBoundsAndPrunesByRetention(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	store := &SkillReadStore{
		path:       filepath.Join(t.TempDir(), "skill-reads.jsonl"),
		now:        func() time.Time { return now },
		maxEntries: 2,
		retention:  30 * 24 * time.Hour,
	}
	if err := store.Append(SkillRead{ID: "old", At: now.Add(-40 * 24 * time.Hour).Format(time.RFC3339), SkillID: "old"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadSince(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("retention left stale entry before cap eviction: %#v", got)
	}
	for _, id := range []string{"one", "two", "three"} {
		if err := store.Append(SkillRead{ID: id, SkillID: id}); err != nil {
			t.Fatal(err)
		}
	}

	got, err = store.ReadSince(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "two" || got[1].ID != "three" {
		t.Fatalf("bounded reads = %#v, want latest two entries", got)
	}
}

func TestSkillReadStoreWindowReturnsAnEmptyCollectionWhenNoRowsMatch(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	store := &SkillReadStore{
		path:      filepath.Join(t.TempDir(), "skill-reads.jsonl"),
		now:       func() time.Time { return now },
		retention: 365 * 24 * time.Hour,
	}
	if err := store.Append(SkillRead{ID: "old", At: now.Add(-24 * time.Hour).Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadSince(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("window reads = %#v, want non-nil empty collection", got)
	}
}

func TestBoundedJSONLAppendPreservesMalformedRowsUntilRetentionRewrite(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	type row struct {
		ID string `json:"id"`
		At string `json:"at"`
	}
	path := filepath.Join(t.TempDir(), "events.jsonl")
	appendRow := func(entry row) error {
		return appendBoundedJSONL(path, entry, now, 10, 30*24*time.Hour, func(entry row) string { return entry.At })
	}
	if err := os.WriteFile(path, []byte("{malformed}\n{\"id\":\"invalid-time\",\"at\":\"not-a-time\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendRow(row{ID: "fresh", At: now.Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "{malformed}") {
		t.Fatalf("append without expired rows discarded malformed line: %q", data)
	}
	rows, err := readJSONLSince(path, 24*time.Hour, now, func(entry row) string { return entry.At })
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != "invalid-time" || rows[1].ID != "fresh" {
		t.Fatalf("window read = %#v, want invalid timestamp retained and malformed line skipped", rows)
	}

	if err := appendRow(row{ID: "expired", At: now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "{malformed}") || strings.Contains(string(data), "expired") {
		t.Fatalf("retention rewrite retained malformed or expired row: %q", data)
	}
	rows, err = readJSONLSince(path, 0, time.Time{}, func(entry row) string { return entry.At })
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != "invalid-time" || rows[1].ID != "fresh" {
		t.Fatalf("post-prune rows = %#v, want invalid timestamp and fresh row", rows)
	}
}
