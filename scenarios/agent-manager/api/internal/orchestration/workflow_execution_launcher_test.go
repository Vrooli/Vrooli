package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

type cleanupOnlyWorkflowLauncher struct {
	*fakeRunLauncher
	dispatches int
	stops      []uuid.UUID
}

func (l *cleanupOnlyWorkflowLauncher) StartFresh(context.Context, workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	l.dispatches++
	return workflowruntime.ChildState{}, errors.New("cleanup must not dispatch")
}

func (l *cleanupOnlyWorkflowLauncher) Continue(context.Context, workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	l.dispatches++
	return workflowruntime.ChildState{}, errors.New("cleanup must not continue")
}

func (l *cleanupOnlyWorkflowLauncher) Stop(ctx context.Context, id uuid.UUID) error {
	l.stops = append(l.stops, id)
	return l.fakeRunLauncher.Stop(ctx, id)
}

func persistUnacknowledgedWorkflowRun(t *testing.T, repos *database.Repositories, attempt *domain.WorkflowNodeAttempt, runID uuid.UUID) {
	t.Helper()
	task := &domain.Task{ID: uuid.NewSHA1(attempt.ID, []byte("workflow-node-task")), Title: "original workflow dispatch", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: runID, TaskID: task.ID, ConversationID: "original-conversation", Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, CreatedAt: time.Now().UTC(), CustomEnv: map[string]string{
		workflowExecutionEnv: attempt.ExecutionID.String(), workflowAttemptEnv: attempt.ID.String(), workflowNodeEnv: attempt.NodeID,
	}}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
}

func removeWorkflowDispatchAcknowledgement(t *testing.T, repos *database.Repositories, executionID uuid.UUID) *domain.WorkflowNodeAttempt {
	t.Helper()
	x, err := repos.WorkflowExecutions.Get(t.Context(), executionID)
	if err != nil {
		t.Fatal(err)
	}
	attempts, err := repos.WorkflowExecutions.ListAttempts(t.Context(), executionID)
	if err != nil || len(attempts) != 1 {
		t.Fatalf("fixture attempts: %+v %v", attempts, err)
	}
	a := attempts[0]
	a.RunID, a.ChildExecutionID, a.ConversationID = nil, nil, ""
	a.Status, a.Version = domain.WorkflowAttemptDispatchPending, a.Version+1
	x.Version++
	x.BudgetUsage.Children = 0
	if ok, err := repos.WorkflowExecutions.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: a}); err != nil || !ok {
		t.Fatalf("remove dispatch acknowledgement: %t %v", ok, err)
	}
	return a
}

func TestWorkflowCancellationRecoversOriginalUnboundRunWithoutDispatch(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "already persisted", true: "late original dispatch"}[late], func(t *testing.T) {
			fake := newFakeRunLauncher()
			o, repos := newRelayOrchestrator(t, fake)
			if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
				t.Fatal(err)
			}
			x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "lost-run-ack"})
			if err != nil {
				t.Fatal(err)
			}
			runID := runIDForNode(t, repos.WorkflowExecutions, x.ID, "a")
			a := removeWorkflowDispatchAcknowledgement(t, repos, x.ID)
			if !late {
				persistUnacknowledgedWorkflowRun(t, repos, a, runID)
			}
			restarted, _ := reopenRelayOrchestrator(t, repos, fake)
			guard := &cleanupOnlyWorkflowLauncher{fakeRunLauncher: fake}
			restarted.workflowEngine.Children = guard
			_, _ = restarted.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "stop-original", Reason: "operator"})
			if late {
				pending, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
				if pending.Status != domain.WorkflowExecutionCancelling || pending.BudgetUsage.AccountingComplete || len(guard.stops) != 0 {
					t.Fatalf("absence released or fabricated dispatch: %+v stops=%v", pending, guard.stops)
				}
				persistUnacknowledgedWorkflowRun(t, repos, a, runID)
				if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			attempts, _ := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
			pending, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
			if attempts[0].RunID == nil || *attempts[0].RunID != runID || attempts[0].ConversationID != "original-conversation" || len(guard.stops) != 1 || guard.stops[0] != runID || guard.dispatches != 0 || len(fake.byKey) != 1 {
				t.Fatalf("original handle not recovered/stopped: %+v stops=%v dispatches=%d", attempts[0], guard.stops, guard.dispatches)
			}
			if pending.Status != domain.WorkflowExecutionCancelling || pending.BudgetUsage.AccountingComplete {
				t.Fatalf("missing terminal receipt was replaced with zero: %+v", pending)
			}
		})
	}
}

func TestWorkflowCancellationRecoversOriginalUnboundNestedWorkflow(t *testing.T) {
	fake := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, fake)
	child := relayDefinition()
	child.Key, child.Definition.Key, child.Digest = "owner/child", "owner/child", "sha256:child"
	parent := relayDefinition()
	parent.Definition.EntryNode = "review"
	parent.Definition.Nodes = []domain.WorkflowNode{{ID: "review", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "owner/child", Version: "1.0.0", MaxDepth: 2}}}
	parent.Definition.Edges = nil
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{child, parent}); err != nil {
		t.Fatal(err)
	}
	o.workflowEngine.Subworkflows = workflowSubworkflowLauncher{o: o}
	x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "lost-nested-ack"})
	if err != nil {
		t.Fatal(err)
	}
	attempts, _ := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
	childID := *attempts[0].ChildExecutionID
	runID := runIDForNode(t, repos.WorkflowExecutions, childID, "a")
	removeWorkflowDispatchAcknowledgement(t, repos, x.ID)
	restarted, _ := reopenRelayOrchestrator(t, repos, fake)
	guard := &cleanupOnlyWorkflowLauncher{fakeRunLauncher: fake}
	restarted.workflowEngine.Children = guard
	restarted.workflowEngine.Subworkflows = workflowSubworkflowLauncher{o: restarted}
	_, _ = restarted.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "stop-original", Reason: "operator"})
	attempts, _ = repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
	if attempts[0].ChildExecutionID == nil || *attempts[0].ChildExecutionID != childID || len(guard.stops) != 1 || guard.stops[0] != runID || guard.dispatches != 0 || len(fake.byKey) != 1 {
		t.Fatalf("nested original not recovered/stopped: %+v stops=%v dispatches=%d", attempts[0], guard.stops, guard.dispatches)
	}
}

func TestWorkflowCancellationRejectsUnboundRunWithDifferentProvenance(t *testing.T) {
	fake := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, fake)
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatal(err)
	}
	x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "wrong-run-binding"})
	if err != nil {
		t.Fatal(err)
	}
	runID := runIDForNode(t, repos.WorkflowExecutions, x.ID, "a")
	a := removeWorkflowDispatchAcknowledgement(t, repos, x.ID)
	persistUnacknowledgedWorkflowRun(t, repos, a, runID)
	run, _ := repos.Runs.Get(t.Context(), runID)
	run.CustomEnv[workflowExecutionEnv] = uuid.NewString()
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	guard := &cleanupOnlyWorkflowLauncher{fakeRunLauncher: fake}
	o.workflowEngine.Children = guard
	_, _ = o.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "stop-original", Reason: "operator"})
	attempts, _ := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
	pending, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	if attempts[0].RunID != nil || len(guard.stops) != 0 || guard.dispatches != 0 || pending.Status != domain.WorkflowExecutionCancelling {
		t.Fatalf("different owner's run was bound/stopped: %+v stops=%v dispatches=%d", attempts[0], guard.stops, guard.dispatches)
	}
}

func TestWorkflowCancellationStopsUnacknowledgedContinuationAndWaitsForOriginalReceipt(t *testing.T) {
	fake := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, fake)
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatal(err)
	}
	x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "lost-continuation-ack"})
	if err != nil {
		t.Fatal(err)
	}
	attempts, _ := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
	source := attempts[0]
	runID := *source.RunID
	persistUnacknowledgedWorkflowRun(t, repos, source, runID)
	source.Status, source.Version = domain.WorkflowAttemptCompleted, source.Version+1
	continuation := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: "b", Ordinal: 1, Strategy: domain.WorkflowAttemptContinue, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: "original-continuation", SourceAttemptID: &source.ID, InputSnapshot: json.RawMessage(`{}`), Version: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	x.Version++
	if ok, err := repos.WorkflowExecutions.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempts: []*domain.WorkflowNodeAttempt{source, continuation}}); err != nil || !ok {
		t.Fatalf("continuation fixture: %t %v", ok, err)
	}
	if _, err := repos.Idempotency.Reserve(t.Context(), continuation.IdempotencyKey, time.Hour); err != nil {
		t.Fatal(err)
	}
	restarted, _ := reopenRelayOrchestrator(t, repos, fake)
	WithIdempotency(repos.Idempotency)(restarted)
	guard := &cleanupOnlyWorkflowLauncher{fakeRunLauncher: fake}
	restarted.workflowEngine.Children = guard
	_, _ = restarted.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "stop-original", Reason: "operator"})
	pending, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	stored, _ := repos.WorkflowExecutions.GetAttemptByIdempotencyKey(t.Context(), continuation.IdempotencyKey)
	if len(guard.stops) != 1 || guard.stops[0] != runID || stored.RunID != nil || pending.Status != domain.WorkflowExecutionCancelling || pending.BudgetUsage.AccountingComplete {
		t.Fatalf("pending continuation was ignored or prematurely settled: %+v %+v stops=%v", stored, pending, guard.stops)
	}
	if err := repos.Idempotency.Complete(t.Context(), continuation.IdempotencyKey, runID, "Run", nil); err != nil {
		t.Fatal(err)
	}
	if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
		t.Fatal(err)
	}
	stored, _ = repos.WorkflowExecutions.GetAttemptByIdempotencyKey(t.Context(), continuation.IdempotencyKey)
	if stored.RunID == nil || *stored.RunID != runID || guard.dispatches != 0 || len(fake.byKey) != 1 {
		t.Fatalf("original continuation receipt not bound without redispatch: %+v dispatches=%d", stored, guard.dispatches)
	}
}

type workflowLauncherRoleResolver struct{}

func (workflowLauncherRoleResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runnerType, Role: role, Model: "workflow-test-model"}, nil
}

func newWorkflowChildOrchestrator(t *testing.T) (*Orchestrator, *database.Repositories) {
	t.Helper()
	_, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	catalogPath := filepath.Join(t.TempDir(), "roles.json")
	catalog := `{"schemaVersion":1,"metadata":{"catalogId":"workflow-test","updatedAt":"2026-07-23"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(catalogPath, []byte(catalog), 0o600); err != nil {
		t.Fatalf("write role catalog: %v", err)
	}
	state, err := rolepolicy.NewState(catalogPath, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("load role catalog: %v", err)
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)); err != nil {
		t.Fatalf("register workflow runner: %v", err)
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithRunners(registry), WithRolePolicyState(state, workflowLauncherRoleResolver{}),
		WithConfig(OrchestratorConfig{DefaultTimeout: time.Minute, DefaultProjectRoot: t.TempDir(), MaxConcurrentRuns: 4, RequireSandboxByDefault: false}),
		WithRunStateRoot(t.TempDir()),
	)
	t.Cleanup(func() {
		o.dispatcher.Close()
		deadline := time.Now().Add(10 * time.Second)
		for o.dispatcher.Stats().ActiveCount > 0 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if o.dispatcher.Stats().ActiveCount > 0 {
			t.Error("test-owned dispatcher did not drain")
		}
	})
	return o, repos
}

func TestChildStateFromRunPreservesTerminalAccountingAndReviewSemantics(t *testing.T) {
	id := uuid.New()
	result := &domain.RunResult{FinalOutput: "done"}
	summary := &domain.RunSummary{TurnsUsed: 3, TokensUsed: 42, CostEstimate: 0.12}
	for _, test := range []struct {
		name     string
		status   domain.RunStatus
		terminal bool
		failed   bool
	}{
		{name: "running", status: domain.RunStatusRunning},
		{name: "complete", status: domain.RunStatusComplete, terminal: true},
		{name: "needs review is terminal to workflow", status: domain.RunStatusNeedsReview, terminal: true},
		{name: "failed", status: domain.RunStatusFailed, terminal: true, failed: true},
		{name: "cancelled", status: domain.RunStatusCancelled, terminal: true, failed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := childStateFromRun(&domain.Run{ID: id, ConversationID: "conversation-1", Status: test.status, Result: result, Summary: summary})
			if state.RunID != id || state.ConversationID != "conversation-1" || state.Result != result || state.Terminal != test.terminal || state.Failed != test.failed {
				t.Fatalf("child state = %+v", state)
			}
			if state.Turns != 3 || state.Tokens != 42 || state.CostUSD != 0.12 {
				t.Fatalf("accounting lost: %+v", state)
			}
		})
	}
}

func TestWorkflowChildLauncherCreatesDeterministicTaskBeforeRejectingUnknownProfile(t *testing.T) {
	ctx := context.Background()
	o, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	launcher := workflowChildLauncher{o: o}
	req := workflowruntime.ChildRequest{
		ExecutionID: uuid.New(), AttemptID: uuid.New(), NodeID: "implement", ProfileKey: "missing/profile", IdempotencyKey: "child-1",
	}
	_, err := launcher.StartFresh(ctx, req)
	if err == nil {
		t.Fatal("unknown profile unexpectedly started a child run")
	}
	var notFound *domain.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %T %v, want NotFoundError", err, err)
	}
	taskID := uuid.NewSHA1(req.AttemptID, []byte("workflow-node-task"))
	task, getErr := repos.Tasks.Get(ctx, taskID)
	if getErr != nil || task == nil {
		t.Fatalf("deterministic workflow task not persisted: task=%+v err=%v", task, getErr)
	}
	if task.ScopePath != "." || task.Status != domain.TaskStatusQueued {
		t.Fatalf("workflow task = %+v", task)
	}
	// A retry must reuse the same durable task rather than adding duplicate work.
	_, _ = launcher.StartFresh(ctx, req)
	stored, getErr := repos.Tasks.Get(ctx, taskID)
	if getErr != nil || stored == nil || stored.ID != taskID {
		t.Fatalf("retry did not retain deterministic task: task=%+v err=%v", stored, getErr)
	}
}

func TestWorkflowChildLauncherStartsRoleBasedRunWithWorkflowProvenance(t *testing.T) {
	ctx := context.Background()
	o, repos := newWorkflowChildOrchestrator(t)
	req := workflowruntime.ChildRequest{
		ExecutionID: uuid.New(), AttemptID: uuid.New(), NodeID: "implement", RoleRef: "code.default", Prompt: "implement safely", IdempotencyKey: "child-success", ScopePath: "scenarios/agent-manager", MaxTurns: 7, Timeout: time.Minute,
	}
	state, err := (workflowChildLauncher{o: o}).StartFresh(ctx, req)
	if err != nil {
		t.Fatalf("start workflow child: %v", err)
	}
	if state.RunID == uuid.Nil || state.ConversationID == "" || state.Terminal {
		t.Fatalf("child state = %+v", state)
	}
	run, err := repos.Runs.Get(ctx, state.RunID)
	if err != nil || run == nil || run.ResolvedConfig == nil {
		t.Fatalf("persisted workflow run=%+v err=%v", run, err)
	}
	if run.ResolvedConfig.RoleRef != "code.default" || run.ResolvedConfig.MaxTurns != 7 || run.CustomEnv[workflowExecutionEnv] != req.ExecutionID.String() || run.CustomEnv[workflowNodeEnv] != "implement" || run.CustomEnv[workflowAttemptEnv] != req.AttemptID.String() {
		t.Fatalf("workflow provenance/config lost: %+v", run)
	}
	// StartFresh intentionally returns once durable dispatch succeeds.  Its
	// executor remains live, so make the test own that lifecycle before the
	// test repository is torn down.  Otherwise the next test can close the
	// shared SQLite handle while this child is still finalizing.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, getErr := repos.Runs.Get(ctx, state.RunID)
		if getErr != nil {
			t.Fatalf("get stopped workflow child: %v", getErr)
		}
		if stored != nil && stored.Status.IsTerminal() {
			replay, replayErr := (workflowChildLauncher{o: o}).StartFresh(ctx, req)
			if replayErr != nil || replay.RunID != state.RunID {
				t.Fatalf("durable attempt lost its child without a creation cache: %v %v", replay, replayErr)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	// The transcript runner is finite; a failure here means execution failed to
	// drain, rather than merely observing a race with repository teardown.
	t.Fatalf("workflow child %s did not become terminal", state.RunID)
}

func TestWorkflowChildLauncherInspectsStopsParkedAndRejectsMissingContinuationSource(t *testing.T) {
	ctx := context.Background()
	o, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	WithEvents(mocks.NewFakeEventStore())(o)
	launcher := workflowChildLauncher{o: o}
	task := &domain.Task{ID: uuid.New(), Title: "parked workflow child", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, ConversationID: "workflow-conversation", Status: domain.RunStatusParked, Phase: domain.RunPhaseExecuting, Summary: &domain.RunSummary{TurnsUsed: 2}}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	state, err := launcher.Inspect(ctx, run.ID)
	if err != nil || state.RunID != run.ID || state.Terminal || state.Turns != 2 {
		t.Fatalf("inspect state=%+v err=%v", state, err)
	}
	if err := launcher.Stop(ctx, run.ID); err != nil {
		t.Fatalf("stop parked child: %v", err)
	}
	stopped, err := launcher.Inspect(ctx, run.ID)
	if err != nil || !stopped.Terminal || !stopped.Failed {
		t.Fatalf("stopped state=%+v err=%v", stopped, err)
	}
	if _, err := launcher.Continue(ctx, workflowruntime.ChildRequest{}); err == nil {
		t.Fatal("continuation without explicit source unexpectedly succeeded")
	}
	missing := uuid.New()
	if _, err := launcher.Continue(ctx, workflowruntime.ChildRequest{SourceRunID: &missing, Prompt: "continue", IdempotencyKey: "missing-source"}); err == nil {
		t.Fatal("continuation from missing source unexpectedly succeeded")
	}
}

func TestWorkflowSubworkflowLauncherResolvesVersionsDrivesAndCancels(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	revision := relayDefinition()
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate revision: %v", err)
	}
	sub := workflowSubworkflowLauncher{o: o}
	state, err := sub.Start(ctx, workflowruntime.SubworkflowRequest{
		Owner: "owner", WorkflowKey: "owner/relay", Version: revision.SemanticVersion, Input: []byte(`{}`), IdempotencyKey: "child-workflow",
	})
	if err != nil {
		t.Fatalf("start versioned subworkflow: %v", err)
	}
	if state.ExecutionID == uuid.Nil || state.Terminal || state.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("started state = %+v", state)
	}
	inspected, err := sub.Inspect(ctx, state.ExecutionID)
	if err != nil || inspected.ExecutionID != state.ExecutionID || inspected.Terminal {
		t.Fatalf("inspect state=%+v err=%v", inspected, err)
	}
	// A stop acknowledgement retains cancellation until the same original
	// child supplies its terminal accounting receipt.
	childID := runIDForNode(t, repos.WorkflowExecutions, state.ExecutionID, "a")
	if err := sub.Cancel(ctx, state.ExecutionID, "parent cancelled"); err == nil {
		t.Fatal("stop acknowledgement substituted for terminal accounting")
	}
	launcher.mu.Lock()
	childState := launcher.states[childID]
	childState.TokensKnown, childState.ChargeMeasured = true, true
	childState.Tokens, childState.Turns = 23, 2
	launcher.states[childID] = childState
	launcher.mu.Unlock()
	if err := sub.Cancel(ctx, state.ExecutionID, "parent cancelled"); err != nil {
		t.Fatalf("cancel subworkflow: %v", err)
	}
	cancelled, err := sub.Inspect(ctx, state.ExecutionID)
	if err != nil || !cancelled.Terminal || cancelled.Status != domain.WorkflowExecutionCancelled {
		t.Fatalf("cancelled state=%+v err=%v", cancelled, err)
	}
	_, err = sub.Start(ctx, workflowruntime.SubworkflowRequest{Owner: "owner", WorkflowKey: "owner/relay", Version: "9.9.9", Input: []byte(`{}`), IdempotencyKey: "missing-version"})
	if err == nil {
		t.Fatal("missing workflow version unexpectedly started")
	}
}

func TestSubworkflowStateNilAndTerminalProjection(t *testing.T) {
	if state := subworkflowState(nil); state.ExecutionID != uuid.Nil {
		t.Fatalf("nil execution state = %+v", state)
	}
	reason := &domain.WorkflowTerminalReason{Code: "budget_exhausted", Message: "budget exhausted"}
	execution := &domain.WorkflowExecution{ID: uuid.New(), Status: domain.WorkflowExecutionFailed, TerminalReason: reason, Output: []byte(`{"partial":true}`), BudgetUsage: domain.WorkflowBudgetUsage{Turns: 4}}
	state := subworkflowState(execution)
	if !state.Terminal || !state.Failed || state.Status != domain.WorkflowExecutionFailed || state.TerminalReason != reason || state.BudgetUsage.Turns != 4 {
		t.Fatalf("projection = %+v", state)
	}
}

func TestWorkflowVerdictTraversalAndOutcomeStatusFailClosed(t *testing.T) {
	value := json.RawMessage(`{"verdict":"pass","nested":{"a/b":["ignore","approved"]}}`)
	for _, test := range []struct {
		pointer string
		want    string
		ok      bool
	}{
		{pointer: "/verdict", want: "pass", ok: true},
		{pointer: "/nested/a~1b/1", want: "approved", ok: true},
		{pointer: "/nested/a~1b/9"},
		{pointer: "/nested/missing"},
		{pointer: "verdict"},
	} {
		got, ok := workflowVerdict(value, test.pointer)
		if got != test.want || ok != test.ok {
			t.Fatalf("pointer %q = %q,%t; want %q,%t", test.pointer, got, ok, test.want, test.ok)
		}
	}
	if _, ok := workflowVerdict(json.RawMessage(`not-json`), "/verdict"); ok {
		t.Fatal("invalid JSON produced a verdict")
	}
	if outcomeStatus(nil, "pass") != "incomplete" || outcomeStatus(&domain.StructuredResult{Status: domain.StructuredResultSuccess}, "") != "incomplete" || outcomeStatus(&domain.StructuredResult{Status: domain.StructuredResultSuccess}, "pass") != "complete" {
		t.Fatal("outcome status did not preserve evaluator completeness semantics")
	}
	if !containsWorkflowNode([]string{"one", "two"}, "two") || containsWorkflowNode([]string{"one"}, "two") {
		t.Fatal("treatment-node membership is incorrect")
	}
}

func TestChildStateFromRunRetainsRequiredFinalization(t *testing.T) {
	for _, status := range []domain.RunFinalizationStatus{"", domain.RunFinalizationStatusNone, domain.RunFinalizationStatusPending, domain.RunFinalizationStatusRunning, domain.RunFinalizationStatusFailed, domain.RunFinalizationStatusSkipped, domain.RunFinalizationStatusSucceeded} {
		t.Run(string(status), func(t *testing.T) {
			run := &domain.Run{ID: uuid.New(), RunMode: domain.RunModeSandboxed, Status: domain.RunStatusComplete, SandboxConfig: &domain.SandboxConfig{}, FinalizationStatus: status, FinalizationError: "patch failed", Result: &domain.RunResult{FinalOutput: "done"}, Summary: &domain.RunSummary{TokensUsed: 123}}
			state := childStateFromRun(run)
			wantPending := status != domain.RunFinalizationStatusSucceeded
			if (state.FinalizationPending != "") != wantPending || !state.Terminal || state.Tokens != 123 || state.Result != run.Result {
				t.Fatalf("state=%+v pending=%t", state, wantPending)
			}
			run.SandboxConfig.ManualReview = true
			if childStateFromRun(run).FinalizationPending != "" {
				t.Fatal("manual review incorrectly gated")
			}
		})
	}
}

func TestChildFinalizationGateHonorsPersistedApplyPolicy(t *testing.T) {
	disabled := false
	for _, test := range []struct {
		name    string
		run     *domain.Run
		pending bool
	}{
		{"missing sandbox config", &domain.Run{RunMode: domain.RunModeSandboxed, Status: domain.RunStatusComplete}, true},
		{"resolved sandbox config", &domain.Run{RunMode: domain.RunModeSandboxed, Status: domain.RunStatusComplete, ResolvedConfig: &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{}}}, true},
		{"in place", &domain.Run{RunMode: domain.RunModeInPlace, Status: domain.RunStatusComplete, FinalizationStatus: domain.RunFinalizationStatusSkipped}, false},
		{"explicit no apply", &domain.Run{RunMode: domain.RunModeSandboxed, Status: domain.RunStatusComplete, SandboxConfig: &domain.SandboxConfig{AutoApply: &disabled}, FinalizationStatus: domain.RunFinalizationStatusSkipped}, false},
		{"failed no apply", &domain.Run{RunMode: domain.RunModeSandboxed, Status: domain.RunStatusFailed, SandboxConfig: &domain.SandboxConfig{ApplyOnFailure: &disabled}, FinalizationStatus: domain.RunFinalizationStatusSkipped}, false},
		{"cancelled required apply", &domain.Run{RunMode: domain.RunModeSandboxed, Status: domain.RunStatusCancelled, SandboxConfig: &domain.SandboxConfig{}, FinalizationStatus: domain.RunFinalizationStatusFailed}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := childStateFromRun(test.run)
			if (state.FinalizationPending != "") != test.pending {
				t.Fatalf("gate changed saved policy: %+v", state)
			}
		})
	}
}
