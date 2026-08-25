package retention

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vrooli/browser-automation-studio/database"
)

// fakeFS is an in-memory FileSystem for deterministic tests.
type fakeFS struct {
	sizes   map[string]int64 // dir -> bytes (presence means exists)
	removed []string
	failOn  map[string]bool // dirs whose RemoveAll should fail
}

func newFakeFS() *fakeFS {
	return &fakeFS{sizes: map[string]int64{}, failOn: map[string]bool{}}
}

func (f *fakeFS) DirSize(dir string) (int64, bool, error) {
	size, ok := f.sizes[dir]
	if !ok {
		return 0, false, nil
	}
	return size, true, nil
}

func (f *fakeFS) RemoveAll(dir string) error {
	if f.failOn[dir] {
		return errors.New("boom")
	}
	delete(f.sizes, dir)
	f.removed = append(f.removed, dir)
	return nil
}

// fakeStore is an in-memory ExecutionStore.
type fakeStore struct {
	execs   []*database.ExecutionIndex
	deleted []uuid.UUID
	failDel map[uuid.UUID]bool
}

func (s *fakeStore) ListExecutions(_ context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, _, _ int) ([]*database.ExecutionIndex, error) {
	var out []*database.ExecutionIndex
	for _, e := range s.execs {
		if workflowID != nil && e.WorkflowID != *workflowID {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *fakeStore) ListExecutionsByStatus(_ context.Context, status string, _, _ int) ([]*database.ExecutionIndex, error) {
	var out []*database.ExecutionIndex
	for _, e := range s.execs {
		if e.Status == status {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *fakeStore) DeleteExecution(_ context.Context, id uuid.UUID) error {
	if s.failDel[id] {
		return errors.New("db boom")
	}
	s.deleted = append(s.deleted, id)
	for i, e := range s.execs {
		if e.ID == id {
			s.execs = append(s.execs[:i], s.execs[i+1:]...)
			break
		}
	}
	return nil
}

const testRoot = "/recordings"

func mkExec(t *testing.T, status string, ageDays int, wf uuid.UUID) *database.ExecutionIndex {
	t.Helper()
	id := uuid.New()
	started := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).Add(-time.Duration(ageDays) * 24 * time.Hour)
	completed := started.Add(time.Minute)
	return &database.ExecutionIndex{
		ID:          id,
		WorkflowID:  wf,
		Status:      status,
		StartedAt:   started,
		CompletedAt: &completed,
		ResultPath:  filepath.Join(testRoot, id.String(), "result.json"),
	}
}

func newService(store *fakeStore, fs *fakeFS) *Service {
	svc := NewService(store, fs, testRoot, nil)
	return svc.WithClock(func() time.Time { return time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC) })
}

func TestSweep_DryRunNoSideEffects(t *testing.T) {
	wf := uuid.New()
	old := mkExec(t, database.ExecutionStatusCompleted, 10, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{old}}
	fs := newFakeFS()
	fs.sizes[filepath.Join(testRoot, old.ID.String())] = 5000

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: false})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if !rep.DryRun {
		t.Fatalf("expected DryRun=true")
	}
	if rep.RemovedCount != 1 {
		t.Fatalf("expected 1 removal candidate, got %d", rep.RemovedCount)
	}
	if rep.EstimatedBytes != 5000 {
		t.Fatalf("expected 5000 estimated bytes, got %d", rep.EstimatedBytes)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("dry-run must not delete DB rows, deleted %d", len(store.deleted))
	}
	if len(fs.removed) != 0 {
		t.Fatalf("dry-run must not remove directories, removed %d", len(fs.removed))
	}
}

func TestSweep_ApplyDeletesRowAndDir(t *testing.T) {
	wf := uuid.New()
	old := mkExec(t, database.ExecutionStatusFailed, 10, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{old}}
	fs := newFakeFS()
	dir := filepath.Join(testRoot, old.ID.String())
	fs.sizes[dir] = 4096

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.DryRun {
		t.Fatalf("expected DryRun=false")
	}
	if rep.RemovedCount != 1 || rep.ErrorCount != 0 {
		t.Fatalf("expected 1 removed 0 errors, got removed=%d errors=%d", rep.RemovedCount, rep.ErrorCount)
	}
	if len(store.deleted) != 1 || store.deleted[0] != old.ID {
		t.Fatalf("expected DB row deleted")
	}
	if len(fs.removed) != 1 || fs.removed[0] != dir {
		t.Fatalf("expected artifact dir removed")
	}
	if rep.RemovedByStatus[database.ExecutionStatusFailed] != 1 {
		t.Fatalf("expected removed_by_status[failed]=1, got %v", rep.RemovedByStatus)
	}
}

func TestSweep_SkipsRunningAndPending(t *testing.T) {
	wf := uuid.New()
	running := mkExec(t, database.ExecutionStatusRunning, 10, wf)
	pending := mkExec(t, database.ExecutionStatusPending, 10, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{running, pending}}
	fs := newFakeFS()
	fs.sizes[filepath.Join(testRoot, running.ID.String())] = 100
	fs.sizes[filepath.Join(testRoot, pending.ID.String())] = 100

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// Non-terminal executions are never even gathered as candidates.
	if rep.RemovedCount != 0 {
		t.Fatalf("expected nothing removed, got %d", rep.RemovedCount)
	}
	if len(store.deleted) != 0 || len(fs.removed) != 0 {
		t.Fatalf("running/pending must never be deleted")
	}
}

func TestSweep_KeepLatestProtectsNewest(t *testing.T) {
	wf := uuid.New()
	// Three completed execs at decreasing age; keep_latest=1 protects the newest.
	e0 := mkExec(t, database.ExecutionStatusCompleted, 1, wf) // newest
	e1 := mkExec(t, database.ExecutionStatusCompleted, 5, wf)
	e2 := mkExec(t, database.ExecutionStatusCompleted, 9, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{e1, e0, e2}}
	fs := newFakeFS()
	for _, e := range store.execs {
		fs.sizes[filepath.Join(testRoot, e.ID.String())] = 10
	}

	svc := newService(store, fs)
	// max_age_days=0 disables age filter; keep_latest=1 keeps the newest.
	rep, err := svc.Sweep(context.Background(), Options{KeepLatest: 1, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.RemovedCount != 2 {
		t.Fatalf("expected 2 removed, got %d", rep.RemovedCount)
	}
	for _, id := range store.deleted {
		if id == e0.ID {
			t.Fatalf("newest execution should have been protected by keep_latest")
		}
	}
	// e0 must appear as skipped with the keep_latest reason.
	foundProtected := false
	for _, item := range rep.Skipped {
		if item.ExecutionID == e0.ID && item.Reason == ReasonKeepLatest {
			foundProtected = true
		}
	}
	if !foundProtected {
		t.Fatalf("expected newest exec skipped with keep_latest reason")
	}
}

func TestSweep_AgeFilterSkipsTooNew(t *testing.T) {
	wf := uuid.New()
	tooNew := mkExec(t, database.ExecutionStatusCompleted, 0, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{tooNew}}
	fs := newFakeFS()
	fs.sizes[filepath.Join(testRoot, tooNew.ID.String())] = 10

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.RemovedCount != 0 || rep.SkippedCount != 1 {
		t.Fatalf("expected 0 removed 1 skipped, got removed=%d skipped=%d", rep.RemovedCount, rep.SkippedCount)
	}
	if rep.Skipped[0].Reason != ReasonTooNew {
		t.Fatalf("expected too-new reason, got %q", rep.Skipped[0].Reason)
	}
}

func TestSweep_AgeFilterUsesTransportSeconds(t *testing.T) {
	wf := uuid.New()
	tooNew := mkExec(t, database.ExecutionStatusCompleted, 0, wf)
	old := mkExec(t, database.ExecutionStatusCompleted, 3, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{tooNew, old}}
	fs := newFakeFS()
	fs.sizes[filepath.Join(testRoot, tooNew.ID.String())] = 10
	fs.sizes[filepath.Join(testRoot, old.ID.String())] = 20

	rep, err := newService(store, fs).Sweep(context.Background(), Options{MaxAgeSeconds: 24 * 60 * 60, Apply: false})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.RemovedCount != 1 || rep.Removed[0].ExecutionID != old.ID {
		t.Fatalf("transport age filter removed %#v, want only old execution", rep.Removed)
	}
	if rep.SkippedCount != 1 || rep.Skipped[0].ExecutionID != tooNew.ID || rep.Skipped[0].Reason != ReasonTooNew {
		t.Fatalf("transport age filter skipped %#v, want too-new execution", rep.Skipped)
	}
}

func TestSweep_MissingDirIsIdempotent(t *testing.T) {
	wf := uuid.New()
	old := mkExec(t, database.ExecutionStatusCompleted, 10, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{old}}
	fs := newFakeFS() // no dir registered -> missing

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.RemovedCount != 1 || rep.ErrorCount != 0 {
		t.Fatalf("missing dir should still remove DB row, got removed=%d errors=%d", rep.RemovedCount, rep.ErrorCount)
	}
	if len(store.deleted) != 1 {
		t.Fatalf("expected DB row deleted for missing-dir execution")
	}
	if rep.Removed[0].Reason != ReasonMissingDir {
		t.Fatalf("expected missing-dir reason, got %q", rep.Removed[0].Reason)
	}
}

func TestSweep_RejectsPathOutsideRoot(t *testing.T) {
	wf := uuid.New()
	bad := mkExec(t, database.ExecutionStatusCompleted, 10, wf)
	bad.ResultPath = "/etc/passwd" // resolves to /etc, outside recordings root
	store := &fakeStore{execs: []*database.ExecutionIndex{bad}}
	fs := newFakeFS()

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.RemovedCount != 0 || rep.SkippedCount != 1 {
		t.Fatalf("expected unsafe path skipped, got removed=%d skipped=%d", rep.RemovedCount, rep.SkippedCount)
	}
	if rep.Skipped[0].Reason != ReasonUnsafePath {
		t.Fatalf("expected unsafe-path reason, got %q", rep.Skipped[0].Reason)
	}
	if len(fs.removed) != 0 || len(store.deleted) != 0 {
		t.Fatalf("unsafe path must not be deleted")
	}
}

func TestSweep_FileDeleteFailurePreventsRowDeletion(t *testing.T) {
	wf := uuid.New()
	old := mkExec(t, database.ExecutionStatusCompleted, 10, wf)
	store := &fakeStore{execs: []*database.ExecutionIndex{old}}
	fs := newFakeFS()
	dir := filepath.Join(testRoot, old.ID.String())
	fs.sizes[dir] = 10
	fs.failOn[dir] = true

	svc := newService(store, fs)
	rep, err := svc.Sweep(context.Background(), Options{MaxAgeDays: 3, Apply: true})
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if rep.ErrorCount != 1 || rep.RemovedCount != 0 {
		t.Fatalf("expected 1 error 0 removed, got errors=%d removed=%d", rep.ErrorCount, rep.RemovedCount)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("DB row must not be deleted when artifact deletion fails")
	}
}

func TestSweep_RecordingsRootRequired(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, newFakeFS(), "", nil)
	if _, err := svc.Sweep(context.Background(), Options{}); !errors.Is(err, ErrRecordingsRootNotConfigured) {
		t.Fatalf("expected ErrRecordingsRootNotConfigured, got %v", err)
	}
}

func TestSweep_NonTerminalStatusFilterRejected(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, newFakeFS())
	if _, err := svc.Sweep(context.Background(), Options{Status: database.ExecutionStatusRunning}); err == nil {
		t.Fatalf("expected error for non-terminal status filter")
	}
}
