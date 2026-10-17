package phases

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

type retryingRunRepository struct {
	repository.RunRepository
	calls int
}

func TestOldFailureWriterCannotProjectTerminalAfterContinuation(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "fenced failure", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	old := *run
	run.Status = domain.RunStatusFailed
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	run.Status = domain.RunStatusRunning
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	broadcast := mocks.NewFakeBroadcaster()
	FailWithError(t.Context(), FailWithErrorInput{Deps: Deps{Runs: repos.Runs, Broadcaster: broadcast}, Run: &old, Err: errors.New("late old executor exit")})
	if len(broadcast.StatusBroadcasts()) != 0 {
		t.Fatal("rejected old writer still projected a terminal status")
	}
	current, _ := repos.Runs.Get(t.Context(), run.ID)
	if current.Status != domain.RunStatusRunning {
		t.Fatal("old failure replaced current continuation")
	}
}

func (r *retryingRunRepository) Update(ctx context.Context, run *domain.Run) error {
	r.calls++
	if r.calls == 1 {
		return errors.New("transient update failure")
	}
	return r.RunRepository.Update(ctx, run)
}

func TestFailWithErrorRetriesTerminalRunPersistence(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "failure retry", Description: "test", ScopePath: ".", ProjectRoot: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig()}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	stub := &retryingRunRepository{RunRepository: repos.Runs}
	FailWithError(context.Background(), FailWithErrorInput{Deps: Deps{Runs: stub}, Run: run, Err: errors.New("runner failed")})
	if stub.calls != 2 {
		t.Fatalf("update attempts = %d, want 2", stub.calls)
	}
	persisted, err := repos.Runs.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != domain.RunStatusFailed || persisted.ErrorMsg != "runner failed" {
		t.Fatalf("persisted run = %+v", persisted)
	}
}
