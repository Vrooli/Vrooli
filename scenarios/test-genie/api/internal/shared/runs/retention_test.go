package runs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

type recordingDetailStore struct {
	deleted []string
	err     error
}

func (s *recordingDetailStore) DeleteByRunID(_ context.Context, runID string) error {
	if s.err != nil {
		return s.err
	}
	s.deleted = append(s.deleted, runID)
	return nil
}

// seedRun creates an index record plus a run directory with a file of the given
// size so retention size accounting has something to measure.
func seedRun(t *testing.T, idx *Index, scenarioDir, id string, started time.Time, sizeBytes int, leased bool) {
	t.Helper()
	rec := RunRecord{RunID: id, Scenario: "demo", StartedAt: started, Status: StatusPassed}
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
	if leased {
		if _, err := NewPinLeaseStore(scenarioDir).Grant(id, "gct:baseline:x", "test", DefaultPinLeaseTTL, time.Now()); err != nil {
			t.Fatalf("seed lease %s: %v", id, err)
		}
	}
}

func seedRunLog(t *testing.T, scenarioDir, id string, sizeBytes int) {
	t.Helper()
	dir := sharedartifacts.RunLogsDir(scenarioDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed log dir %s: %v", id, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "phase.log"), make([]byte, sizeBytes), 0o644); err != nil {
		t.Fatalf("seed log %s: %v", id, err)
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

	policy := RetentionPolicy{KeepMostRecent: 2}
	report, err := NewRetentionService(dir, policy).Collect(context.Background())
	deleted := report.Deleted
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

func TestGCNeverDropsActiveLease(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC().Add(-365 * 24 * time.Hour) // very old

	// Oldest run has an active lease; it should survive aggressive GC.
	seedRun(t, idx, dir, "pinned-old", base, 0, true)
	seedRun(t, idx, dir, "fresh-1", time.Now().UTC().Add(-2*time.Minute), 0, false)
	seedRun(t, idx, dir, "fresh-2", time.Now().UTC().Add(-1*time.Minute), 0, false)

	policy := RetentionPolicy{KeepMostRecent: 1, KeepMaxAgeDays: 1}
	report, err := NewRetentionService(dir, policy).Collect(context.Background())
	deleted := report.Deleted
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	for _, d := range deleted {
		if d == "pinned-old" {
			t.Fatalf("leased run was deleted: %v", deleted)
		}
	}
	if _, err := idx.Find("pinned-old"); err != nil {
		t.Fatalf("expected leased run to survive, got %v", err)
	}
}

func TestGCKeepsActiveRunAndReportsBindingByteCeiling(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC().Add(-time.Hour)
	seedRun(t, idx, dir, "active", base.Add(-time.Minute), 2*1024*1024, false)
	if err := idx.Update("active", func(record *RunRecord) error {
		record.Status = StatusInProgress
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	seedRun(t, idx, dir, "terminal", base, 2*1024*1024, false)

	report, err := NewRetentionService(dir, RetentionPolicy{KeepMaxSizeMB: 3}).Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Deleted) != 1 || report.Deleted[0] != "terminal" {
		t.Fatalf("deleted = %#v, want only terminal run", report.Deleted)
	}
	if len(report.TriggeredBounds) != 1 || report.TriggeredBounds[0] != "max_bytes" {
		t.Fatalf("triggered bounds = %#v, want max_bytes", report.TriggeredBounds)
	}
	if _, err := idx.Find("active"); err != nil {
		t.Fatalf("active run was not protected: %v", err)
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

	policy := RetentionPolicy{KeepMostRecent: 10, KeepMaxSizeMB: 2}
	report, err := NewRetentionService(dir, policy).Collect(context.Background())
	deleted := report.Deleted
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	// Oldest dropped first to get under the 2 MB cap.
	if len(deleted) != 1 || deleted[0] != "size-0" {
		t.Fatalf("expected oldest (size-0) dropped for size cap, got %v", deleted)
	}
}

func TestGCCountsAndRemovesRunScopedLogs(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC()
	seedRun(t, idx, dir, "old", base, 0, false)
	seedRunLog(t, dir, "old", 1024)
	seedRun(t, idx, dir, "new", base.Add(time.Minute), 0, false)
	seedRunLog(t, dir, "new", 1024)

	report, err := NewRetentionService(dir, RetentionPolicy{KeepMostRecent: 1}).Collect(context.Background())
	deleted := report.Deleted
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != "old" {
		t.Fatalf("deleted = %v, want old", deleted)
	}
	if _, err := os.Stat(sharedartifacts.RunLogsDir(dir, "old")); !os.IsNotExist(err) {
		t.Fatalf("old log tree remains: %v", err)
	}
	if _, err := os.Stat(sharedartifacts.RunLogsDir(dir, "new")); err != nil {
		t.Fatalf("new log tree lost: %v", err)
	}
}

func TestRetentionDeletesExecutionHistoryWithFootprint(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	base := time.Now().UTC()
	seedRun(t, idx, dir, "old", base, 0, false)
	seedRun(t, idx, dir, "new", base.Add(time.Minute), 0, false)
	store := &recordingDetailStore{}
	report, err := NewRetentionService(dir, RetentionPolicy{KeepMostRecent: 1}).WithDetailStore(store).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(report.Deleted) != 1 || report.Deleted[0] != "old" {
		t.Fatalf("deleted = %v", report.Deleted)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "old" {
		t.Fatalf("details deleted = %v", store.deleted)
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

func TestGCDeletesExpiredLease(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	started := time.Now().UTC().Add(-48 * time.Hour)
	seedRun(t, idx, dir, "expired", started, 0, false)
	if _, err := NewPinLeaseStore(dir).Grant("expired", "gct:baseline:x", "expired", time.Hour, started); err != nil {
		t.Fatal(err)
	}
	report, err := NewRetentionService(dir, RetentionPolicy{KeepMostRecent: 1, KeepMaxAgeDays: 1}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(report.Deleted) != 1 || report.Deleted[0] != "expired" {
		t.Fatalf("expired lease prevented deletion: %v", report)
	}
}

func TestRetentionServiceReconcilesInterruptedDeletion(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	seedRun(t, idx, dir, "interrupted", time.Now().UTC(), 1, false)
	seedRunLog(t, dir, "interrupted", 1)
	service := NewRetentionService(dir, DefaultRetentionPolicy())
	if err := os.MkdirAll(service.tombstoneDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(retentionTombstone{RunID: "interrupted", CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(service.tombstoneDir(), "interrupted.json"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if _, err := idx.Find("interrupted"); err != ErrRunNotFound {
		t.Fatalf("index retained interrupted run: %v", err)
	}
	if _, err := os.Stat(sharedartifacts.RunDir(dir, "interrupted")); !os.IsNotExist(err) {
		t.Fatalf("artifact tree remains: %v", err)
	}
	if _, err := os.Stat(sharedartifacts.RunLogsDir(dir, "interrupted")); !os.IsNotExist(err) {
		t.Fatalf("log tree remains: %v", err)
	}
}
