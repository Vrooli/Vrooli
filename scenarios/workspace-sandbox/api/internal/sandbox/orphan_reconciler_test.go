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

package sandbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

// orphanRepo wires a FakeRepository so the reconciler sees Active
// sandboxes (live) for IDs we explicitly seed and the production
// (nil, nil) "missing" convention for everything else.
//
// The previous fakeOrphanRepo returned types.NewNotFoundError for
// missing IDs — that's a non-production-faithful behavior; the real
// SandboxRepository.Get returns (nil, nil) for missing rows. The
// reconciler's isOrphan accepts both, but tests now run against the
// production-faithful path.
func setActive(t *testing.T, repo *mocks.FakeRepository, id uuid.UUID) {
	t.Helper()
	repo.SetSandbox(&types.Sandbox{ID: id, Status: types.StatusActive})
}

func setDeleted(t *testing.T, repo *mocks.FakeRepository, id uuid.UUID) {
	t.Helper()
	repo.SetSandbox(&types.Sandbox{ID: id, Status: types.StatusDeleted})
}

// TestReconcileFilesystemOrphans_NoDirs — quiet system: no dirs, no
// orphans, no audit events, fast pass.
func TestReconcileFilesystemOrphans_NoDirs(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

	report := svc.ReconcileFilesystemOrphans(context.Background())
	if report.FilesystemDirs != 0 || report.OrphansCleaned != 0 || report.OrphansFailed != 0 {
		t.Errorf("expected empty report on quiet system, got %+v", report)
	}
	if got := repo.AuditEventCount("sandbox.orphan-cleaned"); got != 0 {
		t.Errorf("expected 0 audit events on quiet system, got %d", got)
	}
}

// TestReconcileFilesystemOrphans_CleansUnknownDir — the canonical
// 2026-04-28 case: a dir on disk with no repo record. Must be cleaned.
func TestReconcileFilesystemOrphans_CleansUnknownDir(t *testing.T) {
	orphan := uuid.New()
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ListDirsResult = []uuid.UUID{orphan}
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", report.OrphansCleaned)
	}
	if len(drv.OrphanCleanups) != 1 || drv.OrphanCleanups[0] != orphan {
		t.Errorf("expected CleanupOrphan(%s), got %v", orphan, drv.OrphanCleanups)
	}
	if got := repo.AuditEventCount("sandbox.orphan-cleaned"); got != 1 {
		t.Errorf("expected 1 'sandbox.orphan-cleaned' audit event, got %d", got)
	}
}

// TestReconcileFilesystemOrphans_LeavesActiveSandboxes — the safety
// invariant. A dir with an Active repo record MUST NOT be touched.
func TestReconcileFilesystemOrphans_LeavesActiveSandboxes(t *testing.T) {
	live := uuid.New()
	repo := mocks.NewFakeRepository()
	setActive(t, repo, live)
	drv := mocks.NewFakeDriver()
	drv.ListDirsResult = []uuid.UUID{live}
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 0 {
		t.Fatalf("REGRESSION: live Active sandbox was cleaned (cleaned=%d)", report.OrphansCleaned)
	}
	if len(drv.OrphanCleanups) != 0 {
		t.Fatalf("REGRESSION: CleanupOrphan called on live sandbox %v", drv.OrphanCleanups)
	}
}

// TestReconcileFilesystemOrphans_CleansDeletedRepoRecord — a repo record
// with Status=Deleted means the API said "this is gone" but the FS still
// has the dir; safe to remove. Closes the half-finished-Delete window.
func TestReconcileFilesystemOrphans_CleansDeletedRepoRecord(t *testing.T) {
	zombie := uuid.New()
	repo := mocks.NewFakeRepository()
	setDeleted(t, repo, zombie)
	drv := mocks.NewFakeDriver()
	drv.ListDirsResult = []uuid.UUID{zombie}
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

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
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ListDirsResult = []uuid.UUID{good, bad}
	// FakeDriver supports per-ID cleanup failure.
	drv.CleanupOrphanErr = errors.New("mount busy")
	// Configure the driver to fail only for `bad`. We do this by
	// installing a custom driver whose CleanupOrphan inspects the ID.
	failingDrv := &perIDFailingDriver{FakeDriver: drv, failID: bad}
	svc := sandbox.NewService(repo, failingDrv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

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
	if got := repo.AuditEventCount("sandbox.orphan-cleanup-failed"); got != 1 {
		t.Errorf("expected 1 cleanup-failed audit, got %d", got)
	}
}

// perIDFailingDriver wraps FakeDriver to fail CleanupOrphan only for a
// specific ID. Per-ID failure injection isn't worth a generic feature
// on FakeDriver since this is the one test that needs it.
type perIDFailingDriver struct {
	*mocks.FakeDriver
	failID uuid.UUID
}

func (d *perIDFailingDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	if id == d.failID {
		return errors.New("mount busy")
	}
	// Reset the parent's err so the success path takes effect for non-failID.
	saved := d.FakeDriver.CleanupOrphanErr
	d.FakeDriver.CleanupOrphanErr = nil
	defer func() { d.FakeDriver.CleanupOrphanErr = saved }()
	return d.FakeDriver.CleanupOrphan(ctx, id)
}

// TestReconcileFilesystemOrphans_RepoErrorIsFailSafe — if the repository
// returns an error we don't recognize (DB down, transient), we MUST NOT
// delete the dir on the basis of "we couldn't confirm it exists." Skip
// this pass, retry on the next.
func TestReconcileFilesystemOrphans_RepoErrorIsFailSafe(t *testing.T) {
	mystery := uuid.New()
	repo := mocks.NewFakeRepository()
	repo.GetErr = errors.New("connection refused")
	drv := mocks.NewFakeDriver()
	drv.ListDirsResult = []uuid.UUID{mystery}
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

	report := svc.ReconcileFilesystemOrphans(context.Background())

	if report.OrphansCleaned != 0 || len(drv.OrphanCleanups) != 0 {
		t.Fatalf("REGRESSION: dir was cleaned despite repo error (cleaned=%d, drv.cleaned=%v)",
			report.OrphansCleaned, drv.OrphanCleanups)
	}
}

// TestReconcileFilesystemOrphans_DriverListErrorReturnsEmpty — if the
// driver itself can't enumerate (permissions, missing BaseDir), the
// reconciler must return safely with an empty report rather than panic.
func TestReconcileFilesystemOrphans_DriverListErrorReturnsEmpty(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ListSandboxDirsErr = errors.New("permission denied")
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

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
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ListDirsResult = []uuid.UUID{a, b}
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

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
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := sandbox.NewService(repo, drv, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

	report := svc.ReconcileFilesystemOrphans(context.Background())
	if report.Duration < 0 || report.Duration > 5*time.Second {
		t.Errorf("expected reasonable Duration, got %v", report.Duration)
	}
}
