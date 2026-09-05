package orchestration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// fakeRunLauncher is the engine's ChildLauncher seam for these tests: it hands
// back a RunID on dispatch and lets the test mark a run terminal with a
// structured result, standing in for the real run executor without any run
// infrastructure. It is concurrency-safe because the nudger's worker Inspects
// while the test completes.
type fakeRunLauncher struct {
	mu      sync.Mutex
	states  map[uuid.UUID]workflowruntime.ChildState
	byKey   map[string]uuid.UUID
	prompts map[uuid.UUID]string
}

func newFakeRunLauncher() *fakeRunLauncher {
	return &fakeRunLauncher{states: map[uuid.UUID]workflowruntime.ChildState{}, byKey: map[string]uuid.UUID{}, prompts: map[uuid.UUID]string{}}
}

func (f *fakeRunLauncher) StartFresh(_ context.Context, r workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byKey[r.IdempotencyKey]
	if !ok {
		id = uuid.New()
		f.byKey[r.IdempotencyKey] = id
		f.states[id] = workflowruntime.ChildState{RunID: id, ConversationID: "conv-" + id.String()}
	}
	f.prompts[id] = r.Prompt
	return f.states[id], nil
}

func (f *fakeRunLauncher) promptFor(id uuid.UUID) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.prompts[id]
}

func (f *fakeRunLauncher) Continue(ctx context.Context, r workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	return f.StartFresh(ctx, r)
}

func (f *fakeRunLauncher) Inspect(_ context.Context, id uuid.UUID) (workflowruntime.ChildState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.states[id], nil
}

func (f *fakeRunLauncher) Stop(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	state := f.states[id]
	state.Terminal, state.Failed = true, true
	f.states[id] = state
	return nil
}

func (f *fakeRunLauncher) complete(id uuid.UUID, value map[string]any) {
	raw, _ := json.Marshal(value)
	f.mu.Lock()
	defer f.mu.Unlock()
	state := f.states[id]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: string(raw), Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: raw}}
	f.states[id] = state
}

func relayDefinition() *domain.WorkflowRevision {
	def := domain.WorkflowDefinition{
		SchemaVersion: domain.WorkflowSchemaVersionV1,
		Owner:         "owner",
		Key:           "owner/relay",
		Version:       "1.0.0",
		InputSchema:   json.RawMessage(`{}`),
		OutputSchema:  json.RawMessage(`{}`),
		EntryNode:     "a",
		Nodes: []domain.WorkflowNode{
			{ID: "a", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "a"}},
			{ID: "b", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "b"}},
			{ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
		},
		Edges: []domain.WorkflowEdge{{From: "a", To: "b"}, {From: "b", To: "end"}},
		Budgets: domain.WorkflowBudgets{
			WallTimeSeconds: 600, MaxTurns: 10, MaxTokens: 10000, MaxChargeMicroUSD: 10,
			MaxNodeAttempts: 10, MaxChildren: 10, MaxConcurrency: 2, MaxRecursion: 2, MaxRetries: 2, MaxWaitSeconds: 60,
		},
	}
	now := time.Now().UTC()
	return &domain.WorkflowRevision{ID: uuid.New(), Owner: "owner", Key: "owner/relay", SemanticVersion: "1.0.0", Digest: "sha256:relay", Definition: def, Active: true, SourcePath: ".vrooli/agent-manager/relay.json", SourceHash: "sha256:relay", SourceUpdatedAt: now, CreatedAt: now}
}

func newRelayOrchestrator(t *testing.T, launcher *fakeRunLauncher) (*Orchestrator, *database.Repositories) {
	t.Helper()
	t.Setenv("AM_SQLITE_PATH", filepath.Join(t.TempDir(), "am-nudge.db"))
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	db, err := database.NewConnection(log)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repos := database.NewRepositories(db, log)
	o := attachRelayEngine(New(repos.Profiles, repos.Tasks, repos.Runs, WithWorkflowExecutionRepository(repos.WorkflowExecutions), WithWorkflowRepository(repos.Workflows)), repos, launcher)
	return o, repos
}

// attachRelayEngine swaps the orchestrator's engine for one whose ChildLauncher
// is the fake, sharing the same durable store so ExecutionIDForRun and recovery
// see real persisted attempts.
func attachRelayEngine(o *Orchestrator, repos *database.Repositories, launcher *fakeRunLauncher) *Orchestrator {
	expr, _ := workflowruntime.NewExpressionEvaluator()
	o.workflowEngine = &workflowruntime.Engine{Store: repos.WorkflowExecutions, Catalog: repos.Workflows, Children: launcher, Expressions: expr}
	return o
}

func runIDForNode(t *testing.T, repo repository.WorkflowExecutionRepository, executionID uuid.UUID, nodeID string) uuid.UUID {
	t.Helper()
	attempts, err := repo.ListAttempts(context.Background(), executionID)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	for _, attempt := range attempts {
		if attempt.NodeID == nodeID && attempt.RunID != nil {
			return *attempt.RunID
		}
	}
	return uuid.Nil
}

func waitForNodeRun(t *testing.T, repo repository.WorkflowExecutionRepository, executionID uuid.UUID, nodeID string) uuid.UUID {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		if id := runIDForNode(t, repo, executionID, nodeID); id != uuid.Nil {
			return id
		}
		select {
		case <-deadline:
			t.Fatalf("node %s never dispatched a run", nodeID)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestCompletionNudgeAdvancesRunNodeWithoutConsumerAdvance is the core Phase 4
// proof: after a workflow run reaches terminal, the completion nudge alone
// drives the execution to its next node and then to a terminal state — the test
// never calls AdvanceWorkflowExecution. It also exercises the wait RPC as the
// adoption pattern for observing terminal completion.
func TestCompletionNudgeAdvancesRunNodeWithoutConsumerAdvance(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	o.SetWorkflowNudger(NewWorkflowNudger(o.NudgeDrive, 2, 5*time.Second))
	o.workflowNudger.Start()
	defer o.workflowNudger.Stop()

	execution, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "exec-1"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	// Start dispatched node a and parked at the version fixpoint (no further
	// progress until the run finishes).
	runA := waitForNodeRun(t, repos.WorkflowExecutions, execution.ID, "a")

	// Complete run a and fire ONLY the completion nudge — no consumer advance.
	launcher.complete(runA, map[string]any{"step": "a"})
	o.nudgeWorkflowForRun(runA)

	// The nudge must drive the execution to dispatch node b.
	runB := waitForNodeRun(t, repos.WorkflowExecutions, execution.ID, "b")
	if runB == runA {
		t.Fatal("node b reused node a's run id")
	}

	// Complete run b; the nudge drives to terminal, and the blocking wait
	// observes it — the whole flow with zero AdvanceWorkflowExecution calls.
	launcher.complete(runB, map[string]any{"step": "b"})
	o.nudgeWorkflowForRun(runB)

	res, err := o.WaitWorkflowExecution(ctx, execution.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if res.TimedOut || res.Execution.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("execution did not reach succeeded via nudge: %+v", res)
	}
}

// TestCompletionNudgeToleratesConcurrentExplicitAdvance drives the same
// execution from the nudge and from an explicit AdvanceWorkflowExecution
// concurrently after a run terminates. The engine's optimistic-version CAS must
// make one lose and reread; progression to the next node must still happen
// exactly once, cleanly under -race.
func TestCompletionNudgeToleratesConcurrentExplicitAdvance(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	o.SetWorkflowNudger(NewWorkflowNudger(o.NudgeDrive, 4, 5*time.Second))
	o.workflowNudger.Start()
	defer o.workflowNudger.Stop()

	execution, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "exec-race"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	runA := waitForNodeRun(t, repos.WorkflowExecutions, execution.ID, "a")
	launcher.complete(runA, map[string]any{"step": "a"})

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); o.nudgeWorkflowForRun(runA) }()
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = o.AdvanceWorkflowExecution(ctx, execution.ID) }()
	}
	wg.Wait()

	runB := waitForNodeRun(t, repos.WorkflowExecutions, execution.ID, "b")
	if runB == runA {
		t.Fatal("node b reused node a's run id")
	}
	// Exactly one attempt exists for node b despite the racing drivers.
	attempts, err := repos.WorkflowExecutions.ListAttempts(ctx, execution.ID)
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	bCount := 0
	for _, attempt := range attempts {
		if attempt.NodeID == "b" {
			bCount++
		}
	}
	if bCount != 1 {
		t.Fatalf("node b dispatched %d times under concurrent drive, want 1", bCount)
	}
}

// TestCompletionNudgeProgressesAcrossSimulatedRestart proves the reconciler
// recovery backstop covers a crash between run-terminal and the nudge: a fresh
// orchestrator over the same durable store re-drives the non-terminal execution
// on RecoverWorkflowExecutions with no nudge and no consumer advance.
func TestCompletionNudgeProgressesAcrossSimulatedRestart(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	execution, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "exec-restart"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	runA := waitForNodeRun(t, repos.WorkflowExecutions, execution.ID, "a")

	// The run finishes while agent-manager is "down": complete it but never
	// nudge this orchestrator.
	launcher.complete(runA, map[string]any{"step": "a"})

	// Restart: a fresh orchestrator over the SAME durable store and the SAME
	// launcher (Inspect still reports run a terminal). Its recovery sweep — not
	// a nudge, not a consumer advance — must progress the execution.
	restarted, _ := reopenRelayOrchestrator(t, repos, launcher)
	if err := restarted.RecoverWorkflowExecutions(ctx); err != nil {
		t.Fatalf("recover: %v", err)
	}

	runB := waitForNodeRun(t, repos.WorkflowExecutions, execution.ID, "b")
	if runB == runA {
		t.Fatal("node b reused node a's run id after restart")
	}
}

// reopenRelayOrchestrator builds a second orchestrator over an existing repo
// set, standing in for an agent-manager restart against the same database.
func reopenRelayOrchestrator(t *testing.T, repos *database.Repositories, launcher *fakeRunLauncher) (*Orchestrator, *database.Repositories) {
	t.Helper()
	o := attachRelayEngine(New(nil, nil, repos.Runs, WithWorkflowExecutionRepository(repos.WorkflowExecutions), WithWorkflowRepository(repos.Workflows)), repos, launcher)
	return o, repos
}
