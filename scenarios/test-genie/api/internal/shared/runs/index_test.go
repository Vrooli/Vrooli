package runs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// TestIndexReadsCreateNothing pins that querying an index which was never
// written leaves the target directory untouched. withLock created coverage/ and
// the lock file before discovering there was nothing to read, so a plain read of
// scenarios/<name> materialized that directory — which is how phantom scenario
// directories accumulated for every non-scenario name that reached the index.
func TestIndexReadsCreateNothing(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "never-written")
	idx := NewIndex(target)

	records, err := idx.List()
	if err != nil {
		t.Fatalf("List on a missing index: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("List = %d records, want 0", len(records))
	}
	if _, err := idx.Find("any-run"); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("Find on a missing index = %v, want ErrRunNotFound", err)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("reading a missing index created %s; reads must not create the target directory", target)
	}
}

// TestPinLeaseReadsCreateNothing pins the same rule for the pin-lease store,
// which shares the create-then-lock shape.
func TestPinLeaseReadsCreateNothing(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "never-pinned")

	active, err := NewPinLeaseStore(target).ActiveForRun("any-run", time.Now())
	if err != nil {
		t.Fatalf("ActiveForRun on a missing store: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("ActiveForRun = %d leases, want 0", len(active))
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("reading a missing lease store created %s", target)
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

func TestIndexFinalizePublishesSnapshotBeforeTerminalIndex(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0]
	dir := t.TempDir()
	idx := NewIndex(dir)
	started := time.Now().UTC().Truncate(time.Second)
	runID := "20260710-161854-2c0462f9"
	if err := idx.Append(RunRecord{
		RunID:     runID,
		Scenario:  "demo",
		StartedAt: started,
		Status:    StatusInProgress,
		Pins:      []PinRecord{{PinnedBy: "gct:baseline:demo", PinnedAt: started}},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	result := map[string]any{
		"runId":    runID,
		"scenario": "demo",
		"phases": []map[string]any{
			{"name": "unit", "status": "failed", "durationSeconds": 12},
		},
	}
	completed := started.Add(12 * time.Second)
	if err := idx.Finalize(runID, result, func(r *RunRecord) error {
		r.Status = StatusFailed
		r.CompletedAt = completed
		r.Phases = []PhaseRecord{{Name: "unit", Status: "failed", DurationSeconds: 12}}
		return nil
	}); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	rec, err := idx.Find(runID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != StatusFailed || len(rec.Phases) != 1 || len(rec.Pins) != 1 {
		t.Fatalf("terminal index lost evidence or pins: %#v", rec)
	}
	snapshot, err := idx.ReadTerminalSnapshot(runID)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if snapshot.SchemaVersion != TerminalSnapshotSchemaVersion || snapshot.Run.Status != StatusFailed {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	if got := string(snapshot.Result); !strings.Contains(got, `"durationSeconds": 12`) {
		t.Fatalf("snapshot result missing full phase evidence: %s", got)
	}
}

func TestIndexFinalizeFailureLeavesIndexNonTerminal(t *testing.T) {
	idx := NewIndex(t.TempDir())
	runID := "run-finalize-failure"
	if err := idx.Append(RunRecord{RunID: runID, Scenario: "demo", Status: StatusInProgress}); err != nil {
		t.Fatalf("append: %v", err)
	}

	err := idx.Finalize(runID, make(chan int), func(r *RunRecord) error {
		r.Status = StatusPassed
		return nil
	})
	if err == nil {
		t.Fatal("expected snapshot serialization failure")
	}
	rec, findErr := idx.Find(runID)
	if findErr != nil {
		t.Fatalf("find: %v", findErr)
	}
	if rec.Status != StatusInProgress {
		t.Fatalf("partial finalization became terminal: %q", rec.Status)
	}
	if _, statErr := os.Stat(sharedartifacts.RunSnapshotPath(idx.scenarioDir, runID)); !os.IsNotExist(statErr) {
		t.Fatalf("failed finalization left a snapshot artifact: %v", statErr)
	}
}

func TestReadTerminalSnapshotDistinguishesLegacyCorruptAndFuture(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0]
	dir := t.TempDir()
	idx := NewIndex(dir)
	runID := "legacy"
	if _, err := idx.ReadTerminalSnapshot(runID); !errors.Is(err, ErrSnapshotNotFound) {
		t.Fatalf("missing snapshot error = %v, want ErrSnapshotNotFound", err)
	}

	path := sharedartifacts.RunSnapshotPath(dir, runID)
	if err := os.MkdirAll(sharedartifacts.RunDir(dir, runID), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	if _, err := idx.ReadTerminalSnapshot(runID); !errors.Is(err, ErrInvalidTerminalSnapshot) {
		t.Fatalf("corrupt snapshot error = %v, want ErrInvalidTerminalSnapshot", err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":99,"run":{"run_id":"legacy","status":"passed"},"result":{}}`), 0o644); err != nil {
		t.Fatalf("write future: %v", err)
	}
	if _, err := idx.ReadTerminalSnapshot(runID); !errors.Is(err, ErrUnsupportedSnapshotVersion) {
		t.Fatalf("future snapshot error = %v, want ErrUnsupportedSnapshotVersion", err)
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
