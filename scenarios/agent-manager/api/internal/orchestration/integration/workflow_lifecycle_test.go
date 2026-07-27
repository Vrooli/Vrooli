// Package integration exercises durable workflow lifecycles against real
// repositories and a deterministic child-run seam. It deliberately has no
// runner process or wall-clock waiting.
package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	adapterrunner "agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

type lifecycleChildLauncher struct {
	mu      sync.Mutex
	states  map[uuid.UUID]workflowruntime.ChildState
	prompts map[uuid.UUID]string
}

// replayChildLauncher adapts the scriptable transcript fake to the workflow
// ChildLauncher seam. Execute blocks at its declared gate; once it returns the
// next Inspect observes the child terminal, matching a runner process that has
// finished while workflow dispatch records its identity.
type replayChildLauncher struct {
	*lifecycleChildLauncher
	runner    *mocks.TranscriptReplayRunner
	byRequest map[string]uuid.UUID
}

func newReplayChildLauncher(runner *mocks.TranscriptReplayRunner) *replayChildLauncher {
	return &replayChildLauncher{lifecycleChildLauncher: newLifecycleChildLauncher(), runner: runner, byRequest: make(map[string]uuid.UUID)}
}

func (l *replayChildLauncher) StartFresh(ctx context.Context, request workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	l.mu.Lock()
	if existing, ok := l.byRequest[request.IdempotencyKey]; ok {
		state := l.states[existing]
		l.mu.Unlock()
		return workflowruntime.ChildState{RunID: state.RunID, ConversationID: state.ConversationID}, nil
	}
	l.mu.Unlock()
	if _, err := l.runner.Execute(ctx, adapterrunner.ExecuteRequest{}); err != nil {
		return workflowruntime.ChildState{}, err
	}
	state, err := l.lifecycleChildLauncher.StartFresh(ctx, request)
	if err != nil {
		return state, err
	}
	// The child is deliberately returned as non-terminal from dispatch. Its
	// terminal structured result becomes visible on the following Inspect.
	l.lifecycleChildLauncher.complete(state.RunID)
	l.mu.Lock()
	l.byRequest[request.IdempotencyKey] = state.RunID
	l.mu.Unlock()
	return state, nil
}

func newLifecycleChildLauncher() *lifecycleChildLauncher {
	return &lifecycleChildLauncher{states: make(map[uuid.UUID]workflowruntime.ChildState), prompts: make(map[uuid.UUID]string)}
}

func (l *lifecycleChildLauncher) StartFresh(_ context.Context, request workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := workflowruntime.ChildState{RunID: uuid.New(), ConversationID: "deterministic-workflow-child"}
	l.states[state.RunID] = state
	l.prompts[state.RunID] = request.Prompt
	return state, nil
}

func (l *lifecycleChildLauncher) Continue(ctx context.Context, request workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	return l.StartFresh(ctx, request)
}

func (l *lifecycleChildLauncher) Inspect(_ context.Context, id uuid.UUID) (workflowruntime.ChildState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.states[id], nil
}

func (l *lifecycleChildLauncher) Stop(_ context.Context, id uuid.UUID) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.states[id]
	state.Terminal, state.Failed = true, true
	l.states[id] = state
	return nil
}

func (l *lifecycleChildLauncher) complete(id uuid.UUID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.states[id]
	value := json.RawMessage(`{"decision":"approved","summary":"review complete"}`)
	state.Terminal = true
	state.Result = &domain.RunResult{
		FinalOutput: string(value),
		Structured:  &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: value},
	}
	l.states[id] = state
}

func (l *lifecycleChildLauncher) onlyRun(t *testing.T) uuid.UUID {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.states) != 1 {
		t.Fatalf("child runs = %d, want 1", len(l.states))
	}
	for id := range l.states {
		return id
	}
	return uuid.Nil
}

func lifecycleRevision() *domain.WorkflowRevision {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &domain.WorkflowRevision{
		ID: uuid.New(), Owner: "integration", Key: "integration/lifecycle", SemanticVersion: "1.0.0", Digest: "sha256:workflow-lifecycle", Active: true, SourcePath: "test", SourceHash: "test", SourceUpdatedAt: now, CreatedAt: now,
		Definition: domain.WorkflowDefinition{
			SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "integration", Key: "integration/lifecycle", Version: "1.0.0", EntryNode: "review",
			InputSchema:  json.RawMessage(`{"type":"object","required":["subject"],"properties":{"subject":{"type":"string"}}}`),
			OutputSchema: json.RawMessage(`{"type":"object","required":["summary"],"properties":{"summary":{"type":"string"}}}`),
			Nodes: []domain.WorkflowNode{
				{ID: "review", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "review {{.subject}}", Bindings: []domain.WorkflowInputBinding{{Name: "subject", Source: domain.WorkflowBindingInput, Selector: "$.subject", Limit: 1, MaxBytes: 128, RenderAs: "text", MissingPolicy: "error"}}}},
				{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approve", PayloadSchema: json.RawMessage(`{"type":"object"}`), TimeoutSeconds: 60}},
				{ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded", Bindings: []domain.WorkflowInputBinding{{Name: "summary", Source: domain.WorkflowBindingHandoff, Selector: "node=review;$.text", Limit: 1, MaxBytes: 256, RenderAs: "text", MissingPolicy: "error"}}}},
			},
			Edges:   []domain.WorkflowEdge{{From: "review", To: "approval"}, {From: "approval", To: "end"}},
			Budgets: domain.WorkflowBudgets{WallTimeSeconds: 600, MaxTurns: 4, MaxTokens: 1000, MaxCostUSD: 1, MaxNodeAttempts: 4, MaxChildren: 4, MaxConcurrency: 1, MaxRecursion: 1, MaxRetries: 1, MaxWaitSeconds: 60},
		},
	}
}

// TestWorkflowLifecycle_BufferedSignalCompletesWithStructuredChildResult
// covers the durable no-LLM lifecycle: a child run dispatches, an approval
// signal is persisted before the wait arms, the child result is journaled, and
// the wait consumes the buffered signal to produce a typed terminal output.
func TestWorkflowLifecycle_BufferedSignalCompletesWithStructuredChildResult(t *testing.T) {
	ctx := context.Background()
	harness := newOrchestratorHarness(t)
	t.Cleanup(harness.Cleanup)
	launcher := newLifecycleChildLauncher()
	revision := lifecycleRevision()
	if err := harness.Repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate workflow: %v", err)
	}
	expressions, err := workflowruntime.NewExpressionEvaluator()
	if err != nil {
		t.Fatalf("new expression evaluator: %v", err)
	}
	engine := &workflowruntime.Engine{Store: harness.Repos.WorkflowExecutions, Catalog: harness.Repos.Workflows, Children: launcher, Expressions: expressions, Now: harness.Clock.Now}

	execution, err := engine.Start(ctx, revision, json.RawMessage(`{"subject":"release"}`), "workflow-lifecycle")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); err != nil { // create attempt
		t.Fatalf("create attempt: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); err != nil { // dispatch child
		t.Fatalf("dispatch child: %v", err)
	}
	runID := launcher.onlyRun(t)

	// The signal arrives while review is still running. Signal resolves the
	// uniquely declared future wait and must persist instead of being lost.
	if _, duplicate, err := engine.Signal(ctx, execution.ID, "approve", json.RawMessage(`{"by":"operator"}`), "approval-before-arm", 0); err != nil || duplicate {
		t.Fatalf("buffer approval: duplicate=%v err=%v", duplicate, err)
	}
	launcher.complete(runID)
	if _, err := engine.Advance(ctx, execution.ID); err != nil { // observe terminal child and enter wait
		t.Fatalf("complete child: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); err != nil { // arm and consume buffered signal
		t.Fatalf("consume buffered signal: %v", err)
	}
	completed, err := engine.Advance(ctx, execution.ID) // end node
	if err != nil {
		t.Fatalf("complete workflow: %v", err)
	}
	if completed.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("workflow status = %s, want succeeded", completed.Status)
	}
	var output map[string]string
	if err := json.Unmarshal(completed.Output, &output); err != nil {
		t.Fatalf("decode terminal output: %v", err)
	}
	if output["summary"] != `{"decision":"approved","summary":"review complete"}` {
		t.Fatalf("structured child handoff = %q", output["summary"])
	}
}

// TestWorkflowLifecycle_ApprovalGateDoesNotConsumeWallTime drives the
// scriptable multi-turn fake's gate as the approval boundary. The signal is
// stored while dispatch is paused, releases the fake without wall-clock sleep,
// and the workflow reaches terminal without advancing its injected clock.
func TestWorkflowLifecycle_ApprovalGateDoesNotConsumeWallTime(t *testing.T) {
	ctx := context.Background()
	harness := newOrchestratorHarness(t)
	t.Cleanup(harness.Cleanup)
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	replay := mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)
	replay.SetReplayTurns(mocks.ReplayTurn{Result: &adapterrunner.ExecuteResult{Success: true, ExitCode: 0}, Gate: gate, Started: started})
	launcher := newReplayChildLauncher(replay)
	revision := lifecycleRevision()
	if err := harness.Repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate workflow: %v", err)
	}
	expressions, err := workflowruntime.NewExpressionEvaluator()
	if err != nil {
		t.Fatalf("new expression evaluator: %v", err)
	}
	engine := &workflowruntime.Engine{Store: harness.Repos.WorkflowExecutions, Catalog: harness.Repos.Workflows, Children: launcher, Expressions: expressions, Now: harness.Clock.Now}
	execution, err := engine.Start(ctx, revision, json.RawMessage(`{"subject":"release"}`), "workflow-approval-gate")
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	dispatched := make(chan error, 1)
	go func() {
		_, err := engine.Advance(ctx, execution.ID)
		dispatched <- err
	}()
	<-started
	initialClock := harness.Clock.Now()
	if _, duplicate, err := engine.Signal(ctx, execution.ID, "approve", json.RawMessage(`{"by":"operator"}`), "approval-releases-gate", 0); err != nil || duplicate {
		t.Fatalf("store approval: duplicate=%v err=%v", duplicate, err)
	}
	close(gate)
	if err := <-dispatched; err != nil && err != workflowruntime.ErrConcurrentAdvance {
		t.Fatalf("dispatch gated child: %v", err)
	}
	// Signal and dispatch intentionally race over the same durable version. A
	// lost CAS is benign: the engine rereads the buffered approval and the
	// idempotent child launcher returns the already-created child identity.
	if _, err := engine.Advance(ctx, execution.ID); err != nil {
		t.Fatalf("retry dispatch after concurrent signal: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); err != nil {
		t.Fatalf("observe child: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); err != nil {
		t.Fatalf("consume approval: %v", err)
	}
	completed, err := engine.Advance(ctx, execution.ID)
	if err != nil {
		t.Fatalf("finish workflow: %v", err)
	}
	if completed.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("workflow status = %s, want succeeded", completed.Status)
	}
	if !harness.Clock.Now().Equal(initialClock) {
		t.Fatalf("approval wait advanced wall clock: before=%s after=%s", initialClock, harness.Clock.Now())
	}
}
