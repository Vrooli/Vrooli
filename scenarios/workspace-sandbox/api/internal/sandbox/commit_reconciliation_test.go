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
