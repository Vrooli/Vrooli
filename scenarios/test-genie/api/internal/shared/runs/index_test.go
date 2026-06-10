package runs

import (
	"fmt"
	"sync"
	"testing"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"

	"github.com/vrooli/freshness-go/runindex"
)

// TestIndexPathMatchesFreshnessGo pins this write side's index location to the
// shared read contract's: a drift would make freshness-go consumers read an
// index test-genie never writes.
func TestIndexPathMatchesFreshnessGo(t *testing.T) {
	dir := t.TempDir()
	if got, want := runindex.IndexPath(dir), sharedartifacts.RunsIndexPath(dir); got != want {
		t.Fatalf("runindex.IndexPath = %q, test-genie writes %q", got, want)
	}
}

func TestIndexAppendAndFindRoundTrip(t *testing.T) {
	idx := NewIndex(t.TempDir())

	rec := RunRecord{
		RunID:       "20251208-151044-aaaa1111",
		Scenario:    "demo",
		StartedAt:   time.Now().UTC().Truncate(time.Second),
		Status:      StatusInProgress,
		Diagnostics: DiagnosticsConfig{Console: true, Video: true},
	}
	if err := idx.Append(rec); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, err := idx.Find(rec.RunID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Scenario != "demo" || got.Status != StatusInProgress {
		t.Fatalf("unexpected record: %#v", got)
	}
	if !got.Diagnostics.Console || !got.Diagnostics.Video {
		t.Fatalf("diagnostics not round-tripped: %#v", got.Diagnostics)
	}
}

func TestIndexAppendUpsertsSameRunID(t *testing.T) {
	idx := NewIndex(t.TempDir())
	id := "20251208-151044-bbbb2222"

	if err := idx.Append(RunRecord{RunID: id, Status: StatusInProgress}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := idx.Append(RunRecord{RunID: id, Status: StatusPassed}); err != nil {
		t.Fatalf("re-append: %v", err)
	}

	all, err := idx.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected single record after upsert, got %d", len(all))
	}
	if all[0].Status != StatusPassed {
		t.Fatalf("expected upserted status passed, got %q", all[0].Status)
	}
}

func TestIndexListNewestFirst(t *testing.T) {
	idx := NewIndex(t.TempDir())
	base := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 3; i++ {
		if err := idx.Append(RunRecord{
			RunID:     fmt.Sprintf("run-%d", i),
			StartedAt: base.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	all, err := idx.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 || all[0].RunID != "run-2" || all[2].RunID != "run-0" {
		t.Fatalf("expected newest-first ordering, got %v", all)
	}
}

func TestIndexUpdateNotFound(t *testing.T) {
	idx := NewIndex(t.TempDir())
	err := idx.Update("missing", func(*RunRecord) error { return nil })
	if err != ErrRunNotFound {
		t.Fatalf("expected ErrRunNotFound, got %v", err)
	}
}

func TestIndexPinUnpin(t *testing.T) {
	idx := NewIndex(t.TempDir())
	id := "20251208-151044-cccc3333"
	if err := idx.Append(RunRecord{RunID: id, Status: StatusPassed}); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := idx.Pin(id, PinRecord{PinnedBy: "gct:baseline:plan-7c3", PinnedAt: time.Now().UTC(), Reason: "baseline"}); err != nil {
		t.Fatalf("pin: %v", err)
	}
	// Re-pinning by the same owner is idempotent.
	if err := idx.Pin(id, PinRecord{PinnedBy: "gct:baseline:plan-7c3", PinnedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("re-pin: %v", err)
	}
	got, _ := idx.Find(id)
	if !got.IsPinned() || len(got.Pins) != 1 {
		t.Fatalf("expected single pin, got %#v", got.Pins)
	}

	if err := idx.Unpin(id, "gct:baseline:plan-7c3"); err != nil {
		t.Fatalf("unpin: %v", err)
	}
	got, _ = idx.Find(id)
	if got.IsPinned() {
		t.Fatalf("expected no pins after unpin, got %#v", got.Pins)
	}
}

// TestIndexConcurrentAppend validates flock-protected concurrent writes from
// many goroutines do not corrupt the index (Plan A §1.8).
func TestIndexConcurrentAppend(t *testing.T) {
	idx := NewIndex(t.TempDir())
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			err := idx.Append(RunRecord{
				RunID:     fmt.Sprintf("run-%02d", i),
				Scenario:  "demo",
				StartedAt: time.Now().UTC(),
				Status:    StatusPassed,
			})
			if err != nil {
				t.Errorf("concurrent append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	all, err := idx.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != n {
		t.Fatalf("expected %d records after concurrent append, got %d", n, len(all))
	}
	seen := make(map[string]bool, n)
	for _, r := range all {
		if seen[r.RunID] {
			t.Fatalf("duplicate run id %s (corruption)", r.RunID)
		}
		seen[r.RunID] = true
	}
}
