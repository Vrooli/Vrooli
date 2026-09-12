package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

type recordingWorkflowRecovery struct {
	err   error
	calls int
}

func (r *recordingWorkflowRecovery) RecoverWorkflowExecutions(context.Context) error {
	r.calls++
	return r.err
}

type recordingWorkflowLiveness struct {
	err   error
	calls int
}

func (r *recordingWorkflowLiveness) ReconcileUnarmedWorkflowWaits(context.Context, time.Duration, time.Duration) error {
	r.calls++
	return r.err
}

func TestReconcileReapsStrandedPendingRunAndRecordsWorkflowHookFailures(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "reconcile pending", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	pending := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "agent-manager-test-pending", Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued}
	if err := repos.Runs.Create(ctx, pending); err != nil {
		t.Fatalf("create pending run: %v", err)
	}
	workflowRecovery := &recordingWorkflowRecovery{}
	workflowLiveness := &recordingWorkflowLiveness{err: errors.New("liveness backend unavailable")}
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerConfig(ReconcilerConfig{PendingThreshold: time.Nanosecond}),
		WithReconcilerWorkflowRecovery(workflowRecovery),
		WithReconcilerWorkflowWaitingLiveness(workflowLiveness),
	)

	stats := reconciler.RunOnce(ctx)
	if stats.RunsChecked != 1 || stats.StaleRuns != 1 || stats.WorkflowRecoveryRuns != 1 {
		t.Fatalf("reconcile stats = %+v", stats)
	}
	if workflowRecovery.calls != 1 || workflowLiveness.calls != 1 || len(stats.Errors) != 1 {
		t.Fatalf("workflow hook behavior = recovery=%d liveness=%d errors=%+v", workflowRecovery.calls, workflowLiveness.calls, stats.Errors)
	}
	persisted, err := repos.Runs.Get(ctx, pending.ID)
	if err != nil || persisted.Status != domain.RunStatusFailed || persisted.EndedAt == nil || persisted.ErrorMsg == "" {
		t.Fatalf("stranded pending run = %+v, err=%v", persisted, err)
	}
}

func TestReconcilerLifecycleRejectsDoubleStartAndPublishesLastStats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	reconciler := NewReconciler(repos.Runs, nil, WithReconcilerConfig(ReconcilerConfig{Interval: time.Hour}))
	if err := reconciler.Start(ctx); err != nil {
		t.Fatalf("start reconciler: %v", err)
	}
	if err := reconciler.Start(ctx); err == nil {
		t.Fatal("second start was accepted")
	}
	if err := reconciler.Stop(); err != nil {
		t.Fatalf("stop reconciler: %v", err)
	}
	now := time.Now().UTC()
	reconciler.updateStats(ReconcileStats{Timestamp: now, RunsChecked: 3})
	if got := reconciler.LastStats(); !got.Timestamp.Equal(now) || got.RunsChecked != 3 {
		t.Fatalf("last stats = %+v", got)
	}
}
