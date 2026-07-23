package phases

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

type retryingRunRepository struct {
	repository.RunRepository
	calls int
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
