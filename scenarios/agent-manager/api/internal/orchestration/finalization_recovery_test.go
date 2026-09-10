package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"
	"agent-manager/internal/orchestration/testutil/mocks"
	"github.com/google/uuid"
)

func TestRecoverRun_RetriesFinalizationWithoutRepeatingExecution(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "finalization recovery", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	sandboxID := uuid.New()
	ended := time.Date(2026, 9, 10, 7, 51, 20, 0, time.UTC)
	cfg := fixtures.NewSandboxConfig(nil)
	cfg.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTurnCompleted}
	cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, RunMode: domain.RunModeSandboxed, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, EndedAt: &ended, SandboxConfig: cfg, FinalizationStatus: domain.RunFinalizationStatusFailed, FinalizationError: "provenance schema missing", Summary: &domain.RunSummary{TokensUsed: 154600}}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	provider := mocks.NewFakeSandboxProvider()
	calls := 0
	provider.TurnCheckpointFunc = func(_ context.Context, req sandbox.TurnCheckpointRequest) (*sandbox.TurnCheckpointResult, error) {
		calls++
		if req.SandboxID != sandboxID || req.RunID != run.ID.String() {
			t.Fatalf("changed recovery origin: %+v", req)
		}
		if calls == 1 {
			return nil, errors.New("archive digest changed")
		}
		return &sandbox.TurnCheckpointResult{Success: true, Applied: 1, TotalSizeBytes: 30}, nil
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
	if _, err := o.RecoverRun(ctx, run.ID); err == nil {
		t.Fatal("failed evidence recovery returned success")
	}
	failed, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || failed.FinalizationStatus != domain.RunFinalizationStatusFailed {
		t.Fatalf("lost failed finalization: %+v %v", failed, err)
	}
	if provider.DeleteCallCount() != 0 {
		t.Fatal("failed recovery destroyed original sandbox")
	}
	result, err := o.RecoverRun(ctx, run.ID)
	if err != nil || !result.Recovered || result.Run.FinalizationStatus != domain.RunFinalizationStatusSucceeded || result.Run.ChangedFiles != 1 {
		t.Fatalf("retry result=%+v err=%v", result, err)
	}
	if !result.Run.EndedAt.Equal(ended) || result.Run.Status != domain.RunStatusComplete || result.Run.Summary.TokensUsed != 154600 {
		t.Fatalf("recovery rewrote execution evidence: %+v", result.Run)
	}
	if provider.DeleteCallCount() != 1 {
		t.Fatal("successful recovery did not release original sandbox")
	}
	again, err := o.RecoverRun(ctx, run.ID)
	if err != nil || !again.Idempotent || calls != 2 || provider.DeleteCallCount() != 1 {
		t.Fatalf("successful recovery repeated effects: calls=%d result=%+v err=%v", calls, again, err)
	}
}
