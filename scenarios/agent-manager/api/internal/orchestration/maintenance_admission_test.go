package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

func TestMaintenanceFencesNewRunContinueResumeAndAttach(t *testing.T) {
	db, closeDB := testutil.SetupTestDB(t)
	t.Cleanup(closeDB)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithIdempotency(repos.Idempotency), WithMaintenanceGate(gate), WithIdentitySecret([]byte("maintenance-fixture-secret-0123456789")))
	// No test path can launch an executor, even on the pre-fix red run.
	o.dispatcher.Close()
	task := &domain.Task{ID: uuid.New(), Title: "maintenance", ScopePath: "."}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseExecuting, SessionID: "retained-session", RunMode: domain.RunModeInPlace}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	for name, call := range map[string]func() error{
		"create-force": func() error {
			_, err := o.CreateRun(t.Context(), CreateRunRequest{TaskID: task.ID, Prompt: "new", Force: true, IdempotencyKey: "new", Environment: map[string]string{workflowExecutionEnv: uuid.NewString()}})
			return err
		},
		"continue": func() error {
			_, err := o.ContinueRun(t.Context(), ContinueRunRequest{RunID: run.ID, Message: "next", IdempotencyKey: "next"})
			return err
		},
		"resume-review": func() error { _, err := o.ResumeRun(t.Context(), run.ID); return err },
		"attach": func() error {
			_, err := o.AttachRun(t.Context(), AttachRunRequest{HarnessKind: "codex", HarnessSession: "external"})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, maintenance.ErrClosed) {
				t.Fatalf("admission not fenced: %v", err)
			}
		})
	}
	for _, key := range []string{"new", "next"} {
		receipt, err := repos.Idempotency.Check(t.Context(), key)
		if err != nil || receipt != nil {
			t.Fatalf("refused operation reserved %s: %+v %v", key, receipt, err)
		}
	}
}

func TestMaintenanceKeepsAcceptedRunAndContinuationReplays(t *testing.T) {
	db, closeDB := testutil.SetupTestDB(t)
	t.Cleanup(closeDB)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithIdempotency(repos.Idempotency), WithMaintenanceGate(gate))
	t.Cleanup(o.dispatcher.Close)
	task := &domain.Task{ID: uuid.New(), Title: "replay", ScopePath: "."}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, IdempotencyKey: "accepted"}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.Idempotency.Reserve(t.Context(), "accepted", time.Hour); err != nil {
		t.Fatal(err)
	}
	req := ContinueRunRequest{RunID: run.ID, Message: "already accepted", IdempotencyKey: "continued"}
	if _, err := repos.Idempotency.Reserve(t.Context(), req.IdempotencyKey, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := o.completeContinuationReceipt(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	got, err := o.CreateRun(t.Context(), CreateRunRequest{TaskID: task.ID, IdempotencyKey: "accepted"})
	if err != nil || got.ID != run.ID {
		t.Fatalf("accepted run lost: %+v %v", got, err)
	}
	got, err = o.ContinueRun(t.Context(), req)
	if err != nil || got.ID != run.ID {
		t.Fatalf("accepted continuation lost: %+v %v", got, err)
	}
}

func TestMaintenanceFencesWorkflowStartButPreservesAdmittedWorkflow(t *testing.T) {
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	db, closeDB := testutil.SetupTestDB(t)
	t.Cleanup(closeDB)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	WithMaintenanceGate(gate)(o)
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatal(err)
	}
	req := StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "accepted"}
	accepted, err := o.StartWorkflowExecution(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	newReq := req
	newReq.IdempotencyKey = "new"
	if _, err := o.StartWorkflowExecution(t.Context(), newReq); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("new workflow not fenced: %v", err)
	}
	if _, err := o.RetryWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: accepted.ID, IdempotencyKey: "new-retry"}); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("new workflow retry not fenced: %v", err)
	}
	if prior, err := o.StartWorkflowExecution(t.Context(), req); err != nil || prior.ID != accepted.ID {
		t.Fatalf("admitted workflow replay lost: %+v %v", prior, err)
	}
	attempts, err := repos.WorkflowExecutions.ListAttempts(t.Context(), accepted.ID)
	if err != nil || len(attempts) == 0 {
		t.Fatalf("admitted attempts: %v", err)
	}
	ctx, err := o.admittedWorkflowContext(t.Context(), accepted.ID, attempts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	release, err := o.admitMaintenance(ctx)
	if err != nil {
		t.Fatalf("admitted descendant fenced: %v", err)
	}
	release()
	if _, err := o.admittedWorkflowContext(context.Background(), accepted.ID, uuid.New()); err == nil {
		t.Fatal("invented attempt bypassed fence")
	}
}

type blockedAdmissionTask struct {
	repository.TaskRepository
	entered chan struct{}
	release chan struct{}
}

func (r blockedAdmissionTask) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	close(r.entered)
	select {
	case <-r.release:
		return nil, errors.New("fixture admission rejected after release")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestMaintenanceConcurrentCreateCannotDisappearDuringDrain(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, _, _ := testutil.SetupTestReposWithDB(t, db)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	tasks := blockedAdmissionTask{TaskRepository: repos.Tasks, entered: make(chan struct{}), release: make(chan struct{})}
	o := New(repos.Profiles, tasks, repos.Runs, WithMaintenanceGate(gate))
	o.dispatcher.Close()
	done := make(chan error, 1)
	go func() {
		_, err := o.CreateRun(t.Context(), CreateRunRequest{TaskID: uuid.New(), Force: true})
		done <- err
	}()
	<-tasks.entered
	if _, err := gate.Enter(t.Context(), "owner", "concurrent rollout"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	standing, err := gate.Wait(ctx, func(context.Context) (int, error) { cancel(); return 0, nil }, time.Hour)
	if err == nil || standing.Drained {
		t.Fatalf("in-flight acceptance disappeared: %+v %v", standing, err)
	}
	status, err := gate.Status(t.Context())
	if err != nil || status.Admitting != 1 {
		t.Fatalf("accepted work cancelled by drain: %+v %v", status, err)
	}
	close(tasks.release)
	if err := <-done; err == nil || errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("pre-fence admission was retroactively refused: %v", err)
	}
	standing, err = gate.Wait(t.Context(), func(context.Context) (int, error) { return 0, nil }, time.Hour)
	if err != nil || !standing.Drained {
		t.Fatalf("released admission did not settle: %+v %v", standing, err)
	}
}

func TestMaintenanceAdmittedWorkflowContinuationCompletesWhileClosed(t *testing.T) {
	o, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	o.dispatcher.Close()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	WithMaintenanceGate(gate)(o)
	WithRunStateRoot(t.TempDir())(o)
	WithIdempotency(repos.Idempotency)(o)
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatal(err)
	}
	x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "accepted-parent"})
	if err != nil {
		t.Fatal(err)
	}
	attempts, err := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
	if err != nil || len(attempts) == 0 || attempts[0].RunID == nil {
		t.Fatalf("durable parent attempt: %+v %v", attempts, err)
	}
	task := &domain.Task{ID: uuid.New(), Title: "admitted continuation", ScopePath: "."}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	source := &domain.Run{ID: *attempts[0].RunID, TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, SessionID: "retained", RunMode: domain.RunModeInPlace, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, Timeout: time.Second}}
	if err := repos.Runs.Create(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{SupportsContinuation: true})
	called := make(chan runner.ContinueRequest, 1)
	mock.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		called <- req
		return &runner.ExecuteResult{Success: true, SessionID: req.SessionID}, nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	WithRunners(registry)(o)
	settled := make(chan struct{}, 1)
	nudger := NewWorkflowNudger(func(context.Context, uuid.UUID) error { settled <- struct{}{}; return nil }, 1, time.Second)
	o.workflowNudger = nudger
	nudger.Start()
	defer nudger.Stop()
	closed, err := gate.Enter(t.Context(), "owner", "rollout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.ContinueRun(t.Context(), ContinueRunRequest{RunID: source.ID, Message: "new explicit turn", IdempotencyKey: "new-explicit"}); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("explicit continuation escaped fence: %v", err)
	}
	child, err := (workflowChildLauncher{o: o}).Continue(t.Context(), workflowruntime.ChildRequest{ExecutionID: x.ID, AttemptID: attempts[0].ID, SourceRunID: &source.ID, Prompt: "admitted workflow turn", IdempotencyKey: "admitted-turn"})
	if err != nil || child.RunID != source.ID {
		t.Fatalf("admitted continuation refused: %+v %v", child, err)
	}
	select {
	case req := <-called:
		if req.RunID != source.ID || req.SessionID != "retained" {
			t.Fatal("existing session lost")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("admitted runner did not continue")
	}
	select {
	case <-settled:
	case <-time.After(5 * time.Second):
		t.Fatal("admitted continuation did not settle")
	}
	got, err := repos.Runs.Get(t.Context(), source.ID)
	if err != nil || got.Status != domain.RunStatusComplete || got.EndedAt == nil {
		t.Fatalf("admitted continuation did not complete: %+v %v", got, err)
	}
	standing, err := gate.Status(t.Context())
	if err != nil || !standing.Closed || standing.Revision != closed.Revision {
		t.Fatalf("continuation reopened fence: %+v %v", standing, err)
	}
}
