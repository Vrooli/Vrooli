package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

type taskCreateCapture struct {
	repository.TaskRepository
	created *domain.Task
}

func (r *taskCreateCapture) Create(_ context.Context, task *domain.Task) error {
	r.created = task
	return nil
}

func TestCreateTaskClearsUntrustedCurrentRunProjection(t *testing.T) {
	staleRunID := uuid.New()
	repo := &taskCreateCapture{}
	o := &Orchestrator{tasks: repo, clock: time.Now}
	created, err := o.CreateTask(context.Background(), &domain.Task{
		ID:           uuid.New(),
		Title:        "unowned task",
		Status:       domain.TaskStatusCompleted,
		CurrentRunID: &staleRunID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != domain.TaskStatusQueued || created.CurrentRunID != nil || repo.created != created {
		t.Fatalf("caller-provided lifecycle projection survived creation: task=%+v persisted=%+v", created, repo.created)
	}
}

func TestCancelTaskDurablyFencesQueuedAdmissionBeforeDispatch(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "cancel queued attempt", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	d := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 2})
	defer d.Close()
	blockerEntered := make(chan struct{})
	unblock := make(chan struct{})
	if err := d.Enqueue(&spawn.Job{RunID: uuid.New(), Fn: func(started spawn.StartedFn) { started(); close(blockerEntered); <-unblock }}); err != nil {
		t.Fatal(err)
	}
	<-blockerEntered

	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued}
	admission, err := d.ReserveAdmission(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.CreateTaskAttempt(ctx, task, run, nil, task.ProjectRoot); err != nil {
		t.Fatal(err)
	}
	executed := make(chan struct{}, 1)
	if err := d.EnqueueAdmitted(&spawn.Job{RunID: run.ID, Fn: func(spawn.StartedFn) { executed <- struct{}{} }}, admission); err != nil {
		t.Fatal(err)
	}

	o := &Orchestrator{tasks: repos.Tasks, runs: repos.Runs, dispatcher: d, clock: time.Now}
	if err := o.CancelTask(ctx, task.ID); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	close(unblock)
	deadline := time.Now().Add(time.Second)
	for d.Stats().ActiveCount != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if d.Stats().ActiveCount != 0 {
		t.Fatal("dispatcher did not drain cancelled job")
	}
	select {
	case <-executed:
		t.Fatal("cancelled queued Run was dispatched")
	default:
	}
	gotTask, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil || gotTask.Status != domain.TaskStatusCancelled {
		t.Fatalf("task state=%+v err=%v", gotTask, err)
	}
	gotRun, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || gotRun.Status != domain.RunStatusCancelled {
		t.Fatalf("run state=%+v err=%v", gotRun, err)
	}
}
