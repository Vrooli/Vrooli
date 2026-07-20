package backlog

import (
	"testing"
	"time"
)

func TestIsStaleUsesUpdatedAgeAndAcceptancePaths(t *testing.T) {
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	fresh := BacklogItem{Updated: now.Add(-13 * 24 * time.Hour).Format(time.RFC3339)}
	if IsStale(fresh, t.TempDir(), now) {
		t.Fatal("fresh item marked stale")
	}
	old := fresh
	old.Updated = now.Add(-14 * 24 * time.Hour).Format(time.RFC3339)
	if !IsStale(old, t.TempDir(), now) {
		t.Fatal("old item was not marked stale")
	}
	missing := fresh
	missing.AcceptanceAllow = []string{"missing/path/**"}
	if !IsStale(missing, t.TempDir(), now) {
		t.Fatal("missing acceptance path was not marked stale")
	}
}
