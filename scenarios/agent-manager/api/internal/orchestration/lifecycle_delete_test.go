package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestDeleteTaskAndRunEnforceTerminalLifecyclePolicies(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	o := New(repos.Profiles, repos.Tasks, repos.Runs)
	task := &domain.Task{ID: uuid.New(), Title: "delete", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := o.DeleteTask(ctx, task.ID); err == nil {
		t.Fatal("queued task deleted")
	}
	task.Status = domain.TaskStatusCancelled
	if err := repos.Tasks.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := o.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := repos.Tasks.Get(ctx, task.ID); err != nil || got != nil {
		t.Fatalf("deleted task=%+v err=%v", got, err)
	}

	terminalTask := &domain.Task{ID: uuid.New(), Title: "run-delete", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, terminalTask); err != nil {
		t.Fatal(err)
	}
	pending := &domain.Run{ID: uuid.New(), TaskID: terminalTask.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued}
	if err := repos.Runs.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}
	if err := o.DeleteRun(ctx, pending.ID); err == nil {
		t.Fatal("pending run deleted")
	}
	pending.Status = domain.RunStatusCancelled
	pending.Phase = domain.RunPhaseCompleted
	if err := repos.Runs.Update(ctx, pending); err != nil {
		t.Fatal(err)
	}
	if err := o.DeleteRun(ctx, pending.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := repos.Runs.Get(ctx, pending.ID); err != nil || got != nil {
		t.Fatalf("deleted run=%+v err=%v", got, err)
	}
}
