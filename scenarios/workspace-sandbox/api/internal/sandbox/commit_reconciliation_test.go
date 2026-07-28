package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

func TestReconcileCommittedChanges_RepairsPendingAndExternalRows(t *testing.T) {
	repo := mocks.NewFakeRepository()
	now := time.Now().UTC().Add(-time.Minute)
	externalAt := now.Add(-time.Minute)
	pending := &types.AppliedChange{ID: uuid.New(), SandboxID: uuid.New(), ProjectRoot: "/project", FilePath: "/project/new.go", AppliedAt: now}
	external := &types.AppliedChange{ID: uuid.New(), SandboxID: uuid.New(), ProjectRoot: "/project", FilePath: "/project/old.go", AppliedAt: now, CommittedAt: &externalAt, CommitHash: "EXTERNAL"}
	repo.AppliedChanges = []*types.AppliedChange{pending, external}
	git := mocks.NewFakeGitOps()
	git.ResolvedCommitHash = "0123456789abcdef0123456789abcdef01234567"
	svc := NewService(repo, mocks.NewFakeDriver(), ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}), process.NewOSExecStarter(), WithGitOps(git))

	report := svc.ReconcileCommittedChanges(context.Background())
	if report.Scanned != 2 || report.Repaired != 2 || report.Failed != 0 {
		t.Fatalf("report = %+v", report)
	}
	for _, change := range repo.AppliedChanges {
		if change.CommitHash != git.ResolvedCommitHash || change.CommittedAt == nil {
			t.Fatalf("change not reconciled: %+v", change)
		}
	}
	checks := 0
	for _, call := range git.Calls() {
		if call == "IsGitRepo:/project" {
			checks++
		}
	}
	if checks != 1 {
		t.Fatalf("IsGitRepo checks = %d, want 1 per project root per pass; calls=%v", checks, git.Calls())
	}
}

// TestReconcileCommittedChanges_ChecksSharedProjectRootOncePerPass protects the one-repository-check-per-project-root reconciliation invariant.
func TestReconcileCommittedChanges_ChecksSharedProjectRootOncePerPass(t *testing.T) {
	repo := mocks.NewFakeRepository()
	now := time.Now().UTC().Add(-time.Minute)
	first := &types.AppliedChange{ID: uuid.New(), SandboxID: uuid.New(), ProjectRoot: "/project", FilePath: "/project/first.go", AppliedAt: now}
	second := &types.AppliedChange{ID: uuid.New(), SandboxID: uuid.New(), ProjectRoot: "/project", FilePath: "/project/second.go", AppliedAt: now}
	repo.AppliedChanges = []*types.AppliedChange{first, second}

	git := mocks.NewFakeGitOps()
	git.ResolvedCommitHash = "0123456789abcdef0123456789abcdef01234567"
	svc := NewService(repo, mocks.NewFakeDriver(), ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}), process.NewOSExecStarter(), WithGitOps(git))

	report := svc.ReconcileCommittedChanges(context.Background())
	if report.Scanned != 2 || report.Repaired != 2 || report.Failed != 0 {
		t.Fatalf("report = %+v", report)
	}

	checks := 0
	for _, call := range git.Calls() {
		if call == "IsGitRepo:/project" {
			checks++
		}
	}
	if checks != 1 {
		t.Fatalf("IsGitRepo checks = %d, want 1 per project root per pass; calls=%v", checks, git.Calls())
	}
}

func TestReconcileCommittedChanges_LeavesUncommittedRowsPending(t *testing.T) {
	repo := mocks.NewFakeRepository()
	change := &types.AppliedChange{ID: uuid.New(), SandboxID: uuid.New(), ProjectRoot: "/project", FilePath: "/project/new.go", AppliedAt: time.Now().UTC()}
	repo.AppliedChanges = []*types.AppliedChange{change}
	git := mocks.NewFakeGitOps()
	svc := NewService(repo, mocks.NewFakeDriver(), ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}), process.NewOSExecStarter(), WithGitOps(git))

	report := svc.ReconcileCommittedChanges(context.Background())
	if report.Repaired != 0 || change.CommittedAt != nil || change.CommitHash != "" {
		t.Fatalf("uncommitted change was modified: report=%+v change=%+v", report, change)
	}
}

func TestReconcileCommittedChanges_RetiresOldUntrackedRowOnce(t *testing.T) {
	repo := mocks.NewFakeRepository()
	change := &types.AppliedChange{ID: uuid.New(), SandboxID: uuid.New(), ProjectRoot: "/project", FilePath: "/project/cache.bin", AppliedAt: time.Now().UTC().Add(-721 * time.Hour)}
	repo.AppliedChanges = []*types.AppliedChange{change}
	git := mocks.NewFakeGitOps()
	svc := NewService(repo, mocks.NewFakeDriver(), ServiceConfig{CommitResolutionBatchLimit: 1, CommitResolutionHorizon: 720 * time.Hour}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}), process.NewOSExecStarter(), WithGitOps(git))
	report := svc.ReconcileCommittedChanges(context.Background())
	if report.Retired != 1 || change.UnresolvableAt == nil || change.ResolutionAttempts != 1 {
		t.Fatalf("first pass = %+v change=%+v", report, change)
	}
	report = svc.ReconcileCommittedChanges(context.Background())
	if report.Scanned != 0 || change.ResolutionAttempts != 1 {
		t.Fatalf("retired row was retried: report=%+v change=%+v", report, change)
	}
}

func TestPurgeUnresolvableCommitChangesPreservesCommittedRows(t *testing.T) {
	repo := mocks.NewFakeRepository()
	old := time.Now().UTC().Add(-169 * time.Hour)
	committedAt := time.Now().UTC()
	retired := &types.AppliedChange{ID: uuid.New(), UnresolvableAt: &old}
	committed := &types.AppliedChange{ID: uuid.New(), UnresolvableAt: &old, CommitHash: "0123456789abcdef0123456789abcdef01234567", CommittedAt: &committedAt}
	repo.AppliedChanges = []*types.AppliedChange{retired, committed}
	svc := NewService(repo, mocks.NewFakeDriver(), ServiceConfig{UnresolvedProvenanceRetention: 168 * time.Hour}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}), process.NewOSExecStarter())
	purged, err := svc.PurgeUnresolvableCommitChanges(context.Background())
	if err != nil || purged != 1 || len(repo.AppliedChanges) != 1 || repo.AppliedChanges[0] != committed {
		t.Fatalf("purge=%d err=%v rows=%+v", purged, err, repo.AppliedChanges)
	}
}

func TestDefaultRunner_SchedulesCommitAttributionReconciliation(t *testing.T) {
	repo := mocks.NewFakeRepository()
	change := &types.AppliedChange{
		ID:          uuid.New(),
		SandboxID:   uuid.New(),
		ProjectRoot: "/project",
		FilePath:    "/project/changed.go",
		AppliedAt:   time.Now().UTC(),
	}
	repo.AppliedChanges = []*types.AppliedChange{change}
	git := mocks.NewFakeGitOps()
	git.ResolvedCommitHash = "0123456789abcdef0123456789abcdef01234567"
	svc := NewService(repo, mocks.NewFakeDriver(), ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}), process.NewOSExecStarter(), WithGitOps(git))

	runner := DefaultRunner(svc, 25*time.Millisecond, 0, 25*time.Millisecond, HealConfig{}, nil)
	runner.Start()
	t.Cleanup(runner.Stop)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if change.CommitHash == git.ResolvedCommitHash && change.CommittedAt != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("default runner did not reconcile pending commit attribution: %+v", change)
}
