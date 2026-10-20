package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestCancelTaskReleasesWakeGateBeforeRunnerStop(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	task := &domain.Task{ID: uuid.New(), Title: "cancel lock scope", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
	}
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 2})
	t.Cleanup(dispatcher.Close)
	admission, err := dispatcher.ReserveAdmission(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dispatcher.ReleaseAdmission(run.ID, admission) })
	if !admission.BeginExternalStart() {
		t.Fatal("failed to mark the admitted run as potentially started")
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, run, nil, task.ProjectRoot); err != nil {
		t.Fatal(err)
	}
	// This regression covers the lock scope around terminating an active run.
	// Pending runs are cancelled cooperatively without calling the runner.
	run.Status = domain.RunStatusStarting
	run.Phase = domain.RunPhaseInitializing
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}

	stopEntered := make(chan struct{})
	allowStop := make(chan struct{})
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.StopFunc = func(context.Context, uuid.UUID) error {
		close(stopEntered)
		<-allowStop
		return nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithRunners(registry), WithSpawnDispatcher(dispatcher))

	cancelDone := make(chan error, 1)
	go func() { cancelDone <- o.CancelTask(ctx, task.ID) }()
	select {
	case <-stopEntered:
	case <-time.After(time.Second):
		close(allowStop)
		t.Fatal("CancelTask did not enter runner stop")
	}

	continueDone := make(chan error, 1)
	go func() {
		_, err := o.ContinueRun(ctx, ContinueRunRequest{RunID: run.ID, Message: "must not reach the cancelled run"})
		continueDone <- err
	}()
	select {
	case err := <-continueDone:
		if err == nil {
			close(allowStop)
			t.Fatal("ContinueRun unexpectedly accepted a cancelled task")
		}
	case <-time.After(250 * time.Millisecond):
		close(allowStop)
		t.Fatal("ContinueRun remained blocked behind slow runner stop; wakeMu is still held across external termination")
	}

	close(allowStop)
	select {
	case err := <-cancelDone:
		if err != nil {
			t.Fatalf("CancelTask: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("CancelTask did not finish after runner stop was released")
	}
}
