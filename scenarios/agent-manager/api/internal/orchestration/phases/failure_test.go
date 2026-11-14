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

type failingTerminalRunRepository struct {
	repository.RunRepository
	err error
}

func (r *failingTerminalRunRepository) Update(context.Context, *domain.Run) error {
	return r.err
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
	events := mocks.NewFakeEventStore()
	FailWithError(context.Background(), FailWithErrorInput{Deps: Deps{Runs: stub, Events: events}, Run: run, Err: errors.New("runner failed")})
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
	statusEvents := events.TypedEvents(run.ID, domain.EventTypeStatus)
	if len(statusEvents) != 1 {
		t.Fatalf("status event count = %d, want exactly one after persistence succeeds", len(statusEvents))
	}
	status, ok := statusEvents[0].Data.(*domain.StatusEventData)
	if !ok || status.NewStatus != string(domain.RunStatusFailed) {
		t.Fatalf("status event = %#v, want failed transition", statusEvents[0].Data)
	}
}

func TestHandleFailurePersistsOnlyDefinitivePrelaunchMaintenanceReceipt(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "maintenance refusal", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig()}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	err := &domain.RunnerError{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: domain.RefuseBeforeEffects(domain.ErrProviderAdmissionMaintenance), ExecutionStartedKnown: true, ExecutionStarted: false}
	HandleResult(t.Context(), HandleResultInput{Deps: Deps{Runs: repos.Runs}, Run: run, ExecErr: err})
	persisted, getErr := repos.Runs.Get(t.Context(), run.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if persisted.StopReason != domain.RunStopReasonProviderMaintenance || persisted.TerminalClass != domain.RunTerminalClassInterruption || persisted.ErrorMsg != "Agent execution is paused for Workspace Sandbox maintenance; no agent work started." {
		t.Fatalf("persisted provider refusal receipt = status:%q class:%q reason:%q error:%q", persisted.Status, persisted.TerminalClass, persisted.StopReason, persisted.ErrorMsg)
	}
	for _, ambiguous := range []*domain.RunnerError{
		{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: domain.ErrProviderAdmissionMaintenance, ExecutionStartedKnown: false},
		{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: errors.New("HTTP 409: another conflict"), ExecutionStartedKnown: true},
		{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: domain.RefuseBeforeEffects(domain.ErrProviderAdmissionMaintenance), ExecutionStartedKnown: true, ExecutionStarted: true},
		{RunnerType: domain.RunnerTypeCodex, Operation: "execute", Cause: errors.New("provider admission is closed for lifecycle maintenance"), ExecutionStartedKnown: true},
	} {
		if ApplyProviderMaintenanceFailureReceipt(&domain.Run{}, ambiguous) {
			t.Fatalf("ambiguous dispatch error became a definitive no-start receipt: %+v", ambiguous)
		}
	}
}

func TestFailWithErrorDoesNotPublishTerminalStatusWhenPersistenceFails(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "failure persistence", Description: "test", ScopePath: ".", ProjectRoot: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig()}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	events := mocks.NewFakeEventStore()
	FailWithError(context.Background(), FailWithErrorInput{
		Deps: Deps{Runs: &failingTerminalRunRepository{RunRepository: repos.Runs, err: errors.New("durable write unavailable")}, Events: events},
		Run:  run,
		Err:  errors.New("runner failed"),
	})
	if statusEvents := events.TypedEvents(run.ID, domain.EventTypeStatus); len(statusEvents) != 0 {
		t.Fatalf("published terminal status despite failed persistence: %#v", statusEvents)
	}
}

func TestHandleContextErrorPublishesCancelledAfterPersistence(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "cancel event", Description: "test", ScopePath: ".", ProjectRoot: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig()}
	if err := repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	events := mocks.NewFakeEventStore()
	HandleContextError(context.Background(), HandleContextErrorInput{
		Deps: Deps{Runs: repos.Runs, Events: events}, Run: run, Err: context.Canceled,
	})

	statusEvents := events.TypedEvents(run.ID, domain.EventTypeStatus)
	if len(statusEvents) != 1 {
		t.Fatalf("status event count = %d, want exactly one cancellation transition", len(statusEvents))
	}
	status, ok := statusEvents[0].Data.(*domain.StatusEventData)
	if !ok || status.NewStatus != string(domain.RunStatusCancelled) {
		t.Fatalf("status event = %#v, want cancelled transition", statusEvents[0].Data)
	}
}
