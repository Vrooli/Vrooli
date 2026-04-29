// Regression gates for the filesystem orphan reconciler added in
// 2026-04-28 after the agent-manager mount-leak incident.
//
// These tests pin the contract that prevents the incident from recurring:
//
//  1. The reconciler ONLY acts on dirs whose UUID has no live record in
//     the repository (or whose record is Status=Deleted). Active /
//     Stopped / Approved / Rejected sandboxes are NEVER touched —
//     those belong to the existing repo-driven lifecycle.
//
//  2. Cleanup is delegated to the driver, which knows how to release
//     mounts (fuse-overlayfs, kernel overlayfs) and rm -rf the dir.
//     The reconciler is driver-agnostic.
//
//  3. The audit trail records every orphan acted on so operators can
//     verify the schedule is firing without trawling logs.
//
//  4. The pass is bounded and idempotent — failures on one pass are
//     retried on the next, and a re-run on a quiet system is a no-op.

package sandbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"
)

// -----------------------------------------------------------------------------
// Repo stub for the reconciler
// -----------------------------------------------------------------------------

// fakeOrphanRepo is a minimal Repository that only implements Get and
// LogAuditEvent. It returns the configured status for known IDs and
// NewNotFoundError for unknown ones — exactly what the reconciler relies on.
type fakeOrphanRepo struct {
	mu          sync.Mutex
	known       map[uuid.UUID]types.Status
	getErr      error
	auditEvents []types.AuditEvent
	auditErr    error
}

func newFakeOrphanRepo() *fakeOrphanRepo {
	return &fakeOrphanRepo{known: map[uuid.UUID]types.Status{}}
}

func (r *fakeOrphanRepo) setStatus(id uuid.UUID, s types.Status) {
	r.mu.Lock()
	r.known[id] = s
	r.mu.Unlock()
}

func (r *fakeOrphanRepo) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getErr != nil {
		return nil, r.getErr
	}
	if status, ok := r.known[id]; ok {
		return &types.Sandbox{ID: id, Status: status}, nil
	}
	return nil, types.NewNotFoundError(id.String())
}

func (r *fakeOrphanRepo) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.auditErr != nil {
		return r.auditErr
	}
	r.auditEvents = append(r.auditEvents, *event)
	return nil
}

func (r *fakeOrphanRepo) auditEventCount(eventType string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.auditEvents {
		if e.EventType == eventType {
			n++
		}
	}
	return n
}

// Unused Repository methods — stubbed to satisfy the interface.
func (r *fakeOrphanRepo) Create(context.Context, *types.Sandbox) error { return nil }
func (r *fakeOrphanRepo) Update(context.Context, *types.Sandbox) error { return nil }
func (r *fakeOrphanRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (r *fakeOrphanRepo) List(context.Context, *types.ListFilter) (*types.ListResult, error) {
	return &types.ListResult{}, nil
}

func (r *fakeOrphanRepo) CheckScopeOverlap(context.Context, string, string, *uuid.UUID) ([]types.PathConflict, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) GetActiveSandboxes(context.Context, string) ([]*types.Sandbox, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) GetAuditLog(context.Context, *uuid.UUID, int, int) ([]*types.AuditEvent, int, error) {
	return nil, 0, nil
}
func (r *fakeOrphanRepo) GetStats(context.Context) (*types.SandboxStats, error) { return nil, nil }
func (r *fakeOrphanRepo) FindByIdempotencyKey(context.Context, string) (*types.Sandbox, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) UpdateWithVersionCheck(context.Context, *types.Sandbox, int64) error {
	return nil
}
func (r *fakeOrphanRepo) BeginTx(context.Context) (repository.TxRepository, error) { return nil, nil }
func (r *fakeOrphanRepo) GetGCCandidates(context.Context, *types.GCPolicy, int) ([]*types.Sandbox, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) RecordAppliedChanges(context.Context, []*types.AppliedChange) error {
	return nil
}

func (r *fakeOrphanRepo) GetPendingChanges(context.Context, string, int, int) (*types.PendingChangesResult, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) GetPendingChangeFiles(context.Context, string, []uuid.UUID) ([]*types.AppliedChange, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) GetFileProvenance(context.Context, string, string, int) ([]*types.AppliedChange, error) {
	return nil, nil
}

func (r *fakeOrphanRepo) MarkChangesCommitted(context.Context, []uuid.UUID, string, string) error {
	return nil
}

func (r *fakeOrphanRepo) MarkChangesCommittedByPath(context.Context, string, []string, string, string) (int, int, error) {
	return 0, 0, nil
}

func (r *fakeOrphanRepo) GetPendingChangesByRun(context.Context, string) ([]types.ProvenanceRunGroup, error) {
	return nil, nil
}

// -----------------------------------------------------------------------------
// Driver stub for the reconciler
// -----------------------------------------------------------------------------

type fakeOrphanDriver struct {
	mu sync.Mutex

	dirs       []uuid.UUID
	listErr    error
	cleaned    []uuid.UUID
	cleanupErr map[uuid.UUID]error
}

func (d *fakeOrphanDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]uuid.UUID, len(d.dirs))
	copy(out, d.dirs)
	return out, d.listErr
}

func (d *fakeOrphanDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err, ok := d.cleanupErr[id]; ok {
		return err
	}
	d.cleaned = append(d.cleaned, id)
	// remove from dirs to model successful cleanup so re-runs are no-ops.
	out := d.dirs[:0]
	for _, x := range d.dirs {
		if x != id {
			out = append(out, x)
		}
	}
	d.dirs = out
	return nil
}

// Unused Driver methods. We can't easily skip them because Driver is
// large; stubbing keeps the test type-safe without forcing real driver
// setup (filesystem, exec, etc.) for tests that only need the orphan
// surface.
func (d *fakeOrphanDriver) Type() driver.DriverType                   { return "fake-orphan" }
func (d *fakeOrphanDriver) Version() string                           { return "test" }
func (d *fakeOrphanDriver) IsAvailable(context.Context) (bool, error) { return true, nil }
func (d *fakeOrphanDriver) Mount(context.Context, *types.Sandbox) (*driver.MountPaths, error) {
	return nil, nil
}
func (d *fakeOrphanDriver) Unmount(context.Context, *types.Sandbox) error { return nil }

func (d *fakeOrphanDriver) GetChangedFiles(context.Context, *types.Sandbox) ([]*types.FileChange, error) {
	return nil, nil
}
func (d *fakeOrphanDriver) Cleanup(context.Context, *types.Sandbox) error { return nil }
func (d *fakeOrphanDriver) VerifyMountIntegrity(context.Context, *types.Sandbox) error {
	return nil
}

func (d *fakeOrphanDriver) RemoveFromUpper(context.Context, *types.Sandbox, string) error {
	return nil
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func newReconcilerService(repo *fakeOrphanRepo, drv *fakeOrphanDriver) *Service {
	return &Service{repo: repo, driver: drv}
}

// TestReconcileFilesystemOrphans_NoDirs — quiet system: no dirs, no
// orphans, no audit events, fast pass.
func TestReconcileFilesystemOrphans_NoDirs(t *testing.T) {
	repo := newFakeOrphanRepo()
	drv := &fakeOrphanDriver{}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())
	if report.FilesystemDirs != 0 || report.OrphansCleaned != 0 || report.OrphansFailed != 0 {
		t.Errorf("expected empty report on quiet system, got %+v", report)
	}
	if got := repo.auditEventCount("sandbox.orphan-cleaned"); got != 0 {
		t.Errorf("expected 0 audit events on quiet system, got %d", got)
	}
}

// TestReconcileFilesystemOrphans_CleansUnknownDir — the canonical
// 2026-04-28 case: a dir on disk with no repo record. Must be cleaned.
func TestReconcileFilesystemOrphans_CleansUnknownDir(t *testing.T) {
	orphan := uuid.New()
	repo := newFakeOrphanRepo()
	drv := &fakeOrphanDriver{dirs: []uuid.UUID{orphan}}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", report.OrphansCleaned)
	}
	if len(drv.cleaned) != 1 || drv.cleaned[0] != orphan {
		t.Errorf("expected CleanupOrphan(%s), got %v", orphan, drv.cleaned)
	}
	if got := repo.auditEventCount("sandbox.orphan-cleaned"); got != 1 {
		t.Errorf("expected 1 'sandbox.orphan-cleaned' audit event, got %d", got)
	}
}

// TestReconcileFilesystemOrphans_LeavesActiveSandboxes — the safety
// invariant. A dir with an Active repo record MUST NOT be touched.
func TestReconcileFilesystemOrphans_LeavesActiveSandboxes(t *testing.T) {
	live := uuid.New()
	repo := newFakeOrphanRepo()
	repo.setStatus(live, types.StatusActive)
	drv := &fakeOrphanDriver{dirs: []uuid.UUID{live}}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 0 {
		t.Fatalf("REGRESSION: live Active sandbox was cleaned (cleaned=%d)", report.OrphansCleaned)
	}
	if len(drv.cleaned) != 0 {
		t.Fatalf("REGRESSION: CleanupOrphan called on live sandbox %v", drv.cleaned)
	}
}

// TestReconcileFilesystemOrphans_CleansDeletedRepoRecord — a repo record
// with Status=Deleted means the API said "this is gone" but the FS still
// has the dir; safe to remove. Closes the half-finished-Delete window.
func TestReconcileFilesystemOrphans_CleansDeletedRepoRecord(t *testing.T) {
	zombie := uuid.New()
	repo := newFakeOrphanRepo()
	repo.setStatus(zombie, types.StatusDeleted)
	drv := &fakeOrphanDriver{dirs: []uuid.UUID{zombie}}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 1 {
		t.Errorf("expected dir with Status=Deleted to be cleaned, got cleaned=%d", report.OrphansCleaned)
	}
}

// TestReconcileFilesystemOrphans_RetriesFailure — a CleanupOrphan that
// errors must be reported as failed and surfaced in FailedIDs, but must
// NOT abort the rest of the pass. Idempotency is the contract: next
// pass tries again.
func TestReconcileFilesystemOrphans_RetriesFailure(t *testing.T) {
	good := uuid.New()
	bad := uuid.New()
	repo := newFakeOrphanRepo()
	drv := &fakeOrphanDriver{
		dirs:       []uuid.UUID{good, bad},
		cleanupErr: map[uuid.UUID]error{bad: errors.New("mount busy")},
	}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", report.OrphansCleaned)
	}
	if report.OrphansFailed != 1 {
		t.Errorf("expected 1 failed, got %d", report.OrphansFailed)
	}
	if len(report.FailedIDs) != 1 || report.FailedIDs[0].ID != bad {
		t.Errorf("expected FailedIDs to surface %s, got %+v", bad, report.FailedIDs)
	}
	if got := repo.auditEventCount("sandbox.orphan-cleanup-failed"); got != 1 {
		t.Errorf("expected 1 cleanup-failed audit, got %d", got)
	}
}

// TestReconcileFilesystemOrphans_RepoErrorIsFailSafe — if the repository
// returns an error we don't recognize (DB down, transient), we MUST NOT
// delete the dir on the basis of "we couldn't confirm it exists." Skip
// this pass, retry on the next.
func TestReconcileFilesystemOrphans_RepoErrorIsFailSafe(t *testing.T) {
	mystery := uuid.New()
	repo := newFakeOrphanRepo()
	repo.getErr = errors.New("connection refused")
	drv := &fakeOrphanDriver{dirs: []uuid.UUID{mystery}}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 0 || len(drv.cleaned) != 0 {
		t.Fatalf("REGRESSION: dir was cleaned despite repo error (cleaned=%d, drv.cleaned=%v)",
			report.OrphansCleaned, drv.cleaned)
	}
}

// TestReconcileFilesystemOrphans_DriverListErrorReturnsEmpty — if the
// driver itself can't enumerate (permissions, missing BaseDir), the
// reconciler must return safely with an empty report rather than panic.
func TestReconcileFilesystemOrphans_DriverListErrorReturnsEmpty(t *testing.T) {
	repo := newFakeOrphanRepo()
	drv := &fakeOrphanDriver{listErr: errors.New("permission denied")}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.FilesystemDirs != 0 || report.OrphansCleaned != 0 {
		t.Errorf("expected empty report on driver list error, got %+v", report)
	}
}

// TestReconcileFilesystemOrphans_ReRunIsNoOp — running the reconciler
// twice must not double-count or double-clean. The second pass sees an
// empty filesystem because the first pass cleaned everything.
func TestReconcileFilesystemOrphans_ReRunIsNoOp(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	repo := newFakeOrphanRepo()
	drv := &fakeOrphanDriver{dirs: []uuid.UUID{a, b}}
	svc := newReconcilerService(repo, drv)

	first := svc.ReconcileFilesystemOrphans(context.Background())
	if first.OrphansCleaned != 2 {
		t.Fatalf("first pass: expected 2 cleaned, got %d", first.OrphansCleaned)
	}

	second := svc.ReconcileFilesystemOrphans(context.Background())
	if second.OrphansCleaned != 0 || second.FilesystemDirs != 0 {
		t.Errorf("second pass should be a no-op, got %+v", second)
	}
}

// TestReconcileFilesystemOrphans_DurationMeasured — the report includes
// a wall-clock duration so operators can spot pathologically slow
// passes. Loose bound (<5s for an empty pass) — this is a smoke test,
// not a perf gate.
func TestReconcileFilesystemOrphans_DurationMeasured(t *testing.T) {
	repo := newFakeOrphanRepo()
	drv := &fakeOrphanDriver{}
	svc := newReconcilerService(repo, drv)

	report := svc.ReconcileFilesystemOrphans(context.Background())
	if report.Duration < 0 || report.Duration > 5*time.Second {
		t.Errorf("expected reasonable Duration, got %v", report.Duration)
	}
}
