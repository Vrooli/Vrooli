package audits

import (
	"strings"
	"testing"
	"time"
)

func baseInventory() InventorySummary {
	return InventorySummary{
		Files:          10,
		Directories:    3,
		Symlinks:       1,
		RegularBytes:   1024,
		PathListSHA256: "pathhash",
		TreeContentSHA: "contenthash",
	}
}

func TestCompare_ExactMatch(t *testing.T) {
	live := baseInventory()
	snap := baseInventory()
	c := compareInventories(live, snap, time.Time{})
	if !c.Matches {
		t.Errorf("expected matches=true, got mismatches: %v", c.Mismatches)
	}
	if len(c.Mismatches) != 0 {
		t.Errorf("expected no mismatches, got %v", c.Mismatches)
	}
}

func TestCompare_CountAndHashMismatch(t *testing.T) {
	live := baseInventory()
	snap := baseInventory()
	snap.Files = 9
	snap.PathListSHA256 = "different"
	snap.TreeContentSHA = "different"

	c := compareInventories(live, snap, time.Time{})
	if c.Matches {
		t.Fatalf("expected matches=false")
	}
	joined := strings.Join(c.Mismatches, "; ")
	for _, want := range []string{"file count", "path-list hash", "tree content hash"} {
		if !strings.Contains(joined, want) {
			t.Errorf("mismatches missing %q: %v", want, c.Mismatches)
		}
	}
}

func TestCompare_ContentHashSkippedWhenEitherEmpty(t *testing.T) {
	live := baseInventory()
	live.TreeContentSHA = "" // content hash disabled on live
	snap := baseInventory()
	snap.TreeContentSHA = "whatever"

	c := compareInventories(live, snap, time.Time{})
	for _, m := range c.Mismatches {
		if strings.Contains(m, "tree content hash") {
			t.Errorf("content hash should not be compared when one side is empty: %v", c.Mismatches)
		}
	}
}

func TestCompare_DriftFlagWhenLiveNewer(t *testing.T) {
	snapshotTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	live := baseInventory()
	live.Files = 11 // a mismatch
	live.MaxModTime = snapshotTime.Add(48 * time.Hour)
	snap := baseInventory()

	c := compareInventories(live, snap, snapshotTime)
	if c.Matches {
		t.Fatalf("expected mismatch")
	}
	if !c.LiveNewerThanSnapshot {
		t.Errorf("expected live_newer_than_snapshot=true (live changed after snapshot)")
	}
}

func TestCompare_NoDriftWhenLiveOlderThanSnapshot(t *testing.T) {
	snapshotTime := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	live := baseInventory()
	live.MaxModTime = snapshotTime.Add(-48 * time.Hour)
	snap := baseInventory()

	c := compareInventories(live, snap, snapshotTime)
	if c.LiveNewerThanSnapshot {
		t.Errorf("expected no drift when live's newest mtime predates the snapshot")
	}
}

func TestCompareSQLite_IntegrityAndSchema(t *testing.T) {
	live := baseInventory()
	snap := baseInventory()
	live.SQLite = []SqliteInventory{{Path: "events.db", IntegrityStatus: "ok", SchemaSHA256: "s1"}}
	snap.SQLite = []SqliteInventory{{Path: "events.db", IntegrityStatus: "failed", SchemaSHA256: "s2"}}

	c := compareInventories(live, snap, time.Time{})
	joined := strings.Join(c.Mismatches, "; ")
	if !strings.Contains(joined, "snapshot integrity check failed") {
		t.Errorf("expected snapshot integrity failure flagged: %v", c.Mismatches)
	}
	if !strings.Contains(joined, "schema hash differs") {
		t.Errorf("expected schema hash mismatch flagged: %v", c.Mismatches)
	}
}

func TestCompareSQLite_PresentOnOneSideOnly(t *testing.T) {
	live := baseInventory()
	snap := baseInventory()
	snap.SQLite = []SqliteInventory{{Path: "only-in-snapshot.db", IntegrityStatus: "ok", SchemaSHA256: "s"}}

	c := compareInventories(live, snap, time.Time{})
	joined := strings.Join(c.Mismatches, "; ")
	if !strings.Contains(joined, "present in snapshot, absent in live") {
		t.Errorf("expected one-sided sqlite flagged: %v", c.Mismatches)
	}
}
