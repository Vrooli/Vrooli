package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

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
	return New(repos.Profiles, repos.Tasks, repos.Runs,
		WithRunners(registry), WithRolePolicyState(state, workflowLauncherRoleResolver{}),
		WithConfig(OrchestratorConfig{DefaultTimeout: time.Minute, DefaultProjectRoot: t.TempDir(), MaxConcurrentRuns: 4, RequireSandboxByDefault: false}),
		WithRunStateRoot(t.TempDir()),
	), repos
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
	// Cancellation performs durable child cleanup through the production run
	// service. The relay engine intentionally uses an in-memory launcher, so
	// materialize its dispatched child as the real run that cleanup must stop.
	childID := runIDForNode(t, repos.WorkflowExecutions, state.ExecutionID, "a")
	task := &domain.Task{ID: uuid.New(), Title: "workflow child", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create child task: %v", err)
	}
	if err := repos.Runs.Create(ctx, &domain.Run{ID: childID, TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting}); err != nil {
		t.Fatalf("create child run: %v", err)
	}
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
