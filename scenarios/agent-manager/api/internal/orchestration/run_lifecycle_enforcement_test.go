package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// TestApplyRunStatusTransition_RejectsIllegalTransition verifies that the single
// status-mutation helper enforces the run state machine: an illegal transition
// (here pending → complete) is rejected and the persisted run is left untouched.
// This is the guard that stops future statuses from being set ad-hoc.
func TestApplyRunStatusTransition_RejectsIllegalTransition(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := New(repos.Profiles, repos.Tasks, repos.Runs)

	task, terr := svc.CreateTask(ctx, &domain.Task{Title: "enforce task", ScopePath: "src/"})
	if terr != nil {
		t.Fatalf("create task: %v", terr)
	}

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:            runID,
		TaskID:        task.ID,
		Tag:           runID.String(),
		RunMode:       domain.RunModeInPlace,
		Status:        domain.RunStatusPending,
		Phase:         domain.RunPhaseQueued,
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	_, err := svc.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       run,
		NewStatus: domain.RunStatusComplete, // pending → complete is not allowed
	})
	if err == nil {
		t.Fatal("expected illegal transition to be rejected, got nil error")
	}

	persisted, gerr := repos.Runs.Get(ctx, runID)
	if gerr != nil {
		t.Fatalf("get run: %v", gerr)
	}
	if persisted.Status != domain.RunStatusPending {
		t.Fatalf("run status = %s, want pending (illegal transition must not persist)", persisted.Status)
	}
}

// TestApplyRunStatusTransition_AllowsSameStatusNoop verifies that a same-status
// write (e.g. a heartbeat/progress refresh) is treated as a no-op update rather
// than a transition, and is therefore always permitted even though running →
// running is not an edge in the transition table.
func TestApplyRunStatusTransition_AllowsSameStatusNoop(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	svc := New(repos.Profiles, repos.Tasks, repos.Runs)

	task, terr := svc.CreateTask(ctx, &domain.Task{Title: "noop task", ScopePath: "src/"})
	if terr != nil {
		t.Fatalf("create task: %v", terr)
	}

	now := time.Now()
	runID := uuid.New()
	run := &domain.Run{
		ID:            runID,
		TaskID:        task.ID,
		Tag:           runID.String(),
		RunMode:       domain.RunModeInPlace,
		Status:        domain.RunStatusRunning,
		Phase:         domain.RunPhaseExecuting,
		StartedAt:     &now,
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	progress := 42
	if _, err := svc.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:             run,
		NewStatus:       domain.RunStatusRunning,
		ProgressPercent: &progress,
	}); err != nil {
		t.Fatalf("same-status update should be permitted, got: %v", err)
	}

	persisted, gerr := repos.Runs.Get(ctx, runID)
	if gerr != nil {
		t.Fatalf("get run: %v", gerr)
	}
	if persisted.Status != domain.RunStatusRunning {
		t.Fatalf("run status = %s, want running", persisted.Status)
	}
	if persisted.ProgressPercent != progress {
		t.Fatalf("progress = %d, want %d", persisted.ProgressPercent, progress)
	}
}
