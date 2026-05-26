package runs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// seedRun creates an index record plus a run directory with a file of the given
// size so retention size accounting has something to measure.
func seedRun(t *testing.T, idx *Index, scenarioDir, id string, started time.Time, sizeBytes int, pinned bool) {
	t.Helper()
	rec := RunRecord{RunID: id, Scenario: "demo", StartedAt: started, Status: StatusPassed}
	if pinned {
		rec.Pins = []PinRecord{{PinnedBy: "gct:baseline:x", PinnedAt: started}}
	}
	if err := idx.Append(rec); err != nil {
		t.Fatalf("seed append %s: %v", id, err)
	}
	dir := sharedartifacts.RunDir(scenarioDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed dir %s: %v", id, err)
	}
	if sizeBytes > 0 {
		if err := os.WriteFile(filepath.Join(dir, "blob.bin"), make([]byte, sizeBytes), 0o644); err != nil {
			t.Fatalf("seed blob %s: %v", id, err)
		}
	}
}

func TestGCKeepsMostRecentAndDropsOldest(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC()

	// 5 unpinned runs, oldest..newest.
	for i := 0; i < 5; i++ {
		seedRun(t, idx, dir, "run-"+string(rune('0'+i)), base.Add(time.Duration(i)*time.Minute), 0, false)
	}

	policy := RetentionPolicy{KeepMostRecent: 2, AlwaysKeepPinned: true}
	deleted, err := GC(context.Background(), dir, policy)
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	if len(deleted) != 3 {
		t.Fatalf("expected 3 oldest dropped, got %v", deleted)
	}

	remaining, _ := idx.List()
	if len(remaining) != 2 {
		t.Fatalf("expected 2 runs kept, got %d", len(remaining))
	}
	// The two newest (run-4, run-3) should remain.
	if remaining[0].RunID != "run-4" || remaining[1].RunID != "run-3" {
		t.Fatalf("expected newest two kept, got %v", remaining)
	}
	// Dropped run dirs are gone.
	if _, err := os.Stat(sharedartifacts.RunDir(dir, "run-0")); !os.IsNotExist(err) {
		t.Errorf("expected run-0 dir removed")
	}
}

func TestGCNeverDropsPinned(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC().Add(-365 * 24 * time.Hour) // very old

	// Oldest run is pinned; should survive even aggressive GC.
	seedRun(t, idx, dir, "pinned-old", base, 0, true)
	seedRun(t, idx, dir, "fresh-1", time.Now().UTC().Add(-2*time.Minute), 0, false)
	seedRun(t, idx, dir, "fresh-2", time.Now().UTC().Add(-1*time.Minute), 0, false)

	policy := RetentionPolicy{KeepMostRecent: 1, KeepMaxAgeDays: 1, AlwaysKeepPinned: true}
	deleted, err := GC(context.Background(), dir, policy)
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	for _, d := range deleted {
		if d == "pinned-old" {
			t.Fatalf("pinned run was deleted: %v", deleted)
		}
	}
	if _, err := idx.Find("pinned-old"); err != nil {
		t.Fatalf("expected pinned run to survive, got %v", err)
	}
}

func TestGCEnforcesSizeCap(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC()

	// 3 unpinned runs of 1 MB each, within count limit but over a 2 MB cap.
	const mb = 1024 * 1024
	seedRun(t, idx, dir, "size-0", base.Add(0), mb, false)
	seedRun(t, idx, dir, "size-1", base.Add(time.Minute), mb, false)
	seedRun(t, idx, dir, "size-2", base.Add(2*time.Minute), mb, false)

	policy := RetentionPolicy{KeepMostRecent: 10, KeepMaxSizeMB: 2, AlwaysKeepPinned: true}
	deleted, err := GC(context.Background(), dir, policy)
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	// Oldest dropped first to get under the 2 MB cap.
	if len(deleted) != 1 || deleted[0] != "size-0" {
		t.Fatalf("expected oldest (size-0) dropped for size cap, got %v", deleted)
	}
}

func TestDeleteRunRefusesPinnedWithoutForce(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	seedRun(t, idx, dir, "pinned", time.Now().UTC(), 0, true)

	if err := DeleteRun(dir, "pinned", false); err != ErrRunPinned {
		t.Fatalf("expected ErrRunPinned, got %v", err)
	}
	if err := DeleteRun(dir, "pinned", true); err != nil {
		t.Fatalf("force delete should succeed, got %v", err)
	}
	if _, err := idx.Find("pinned"); err != ErrRunNotFound {
		t.Fatalf("expected run removed after force delete, got %v", err)
	}
}
