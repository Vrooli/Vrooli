package workflowruntime

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

func TestEngineLoopCreatesDistinctFreshRunsAndTerminates(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "slice", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.fast", PromptTemplate: "Work {{.topic}}", Bindings: []domain.WorkflowInputBinding{inputBinding("topic", "$.topic")}}}, {ID: "more", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Expression: "iteration < 2"}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "slice"
	definition.Edges = []domain.WorkflowEdge{{From: "slice", To: "more"}, {From: "more", To: "slice", Condition: "iteration < 2", MaxTraversals: 2}, {From: "more", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{"topic":"A"}`), "loop-1")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		mustAdvance(t, engine, execution.ID)
		mustAdvance(t, engine, execution.ID)
		runID := children.requests[len(children.requests)-1].runID
		children.complete(runID, "handoff")
		mustAdvance(t, engine, execution.ID)
		mustAdvance(t, engine, execution.ID)
	}
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("status=%s reason=%+v", final.Status, final.TerminalReason)
	}
	if len(children.requests) != 2 || children.requests[0].runID == children.requests[1].runID {
		t.Fatalf("fresh loop did not create distinct Runs: %+v", children.requests)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 2 || attempts[0].Ordinal != 1 || attempts[1].Ordinal != 2 {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestEngineContinueUsesNamedPriorRun(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "initial", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "initial"}}, {ID: "followup", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "initial", PromptTemplate: "follow up"}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "initial"
	definition.Edges = []domain.WorkflowEdge{{From: "initial", To: "followup"}, {From: "followup", To: "done"}}
	engine, _, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "continue-1")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	first := children.requests[0].runID
	children.complete(first, "first")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[1].source == nil || *children.requests[1].source != first {
		t.Fatalf("continuation did not preserve named source: %+v", children.requests)
	}
}

func TestRecoveryReusesPersistedDispatchIntentExactlyOnce(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work"}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "recover-1")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID) // persisted intent, no side effect
	expressions, _ := NewExpressionEvaluator()
	restarted := &Engine{Store: store, Catalog: fakeCatalog{revision(definition)}, Children: children, Expressions: expressions}
	mustAdvance(t, restarted, execution.ID)
	mustAdvance(t, restarted, execution.ID)
	if len(children.requests) != 1 {
		t.Fatalf("dispatch count=%d, want 1", len(children.requests))
	}
}

func TestExecutionBudgetExhaustionPersistsCompletedAttempt(t *testing.T) {
	definition := baseDefinition()
	definition.Budgets.MaxTokens = 1
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work"}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "budget-1")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	runID := children.requests[0].runID
	state := children.states[runID]
	state.Terminal = true
	state.Tokens = 2
	state.Result = &domain.RunResult{FinalOutput: "done"}
	children.states[runID] = state
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionBudgetExhausted || final.TerminalReason.BudgetName != "tokens" {
		t.Fatalf("execution=%+v", final)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 1 || attempts[0].Status != domain.WorkflowAttemptCompleted {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestWaitSignalIsTypedDurableAndIdempotent(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", PayloadSchema: json.RawMessage(`{"type":"object","required":["actor"],"properties":{"actor":{"type":"string"}},"additionalProperties":false}`), TimeoutSeconds: 30}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "wait-1")
	waiting := mustAdvance(t, engine, execution.ID)
	if waiting.Status != domain.WorkflowExecutionWaiting || waiting.BudgetUsage.Turns != 0 {
		t.Fatalf("waiting execution=%+v", waiting)
	}
	if _, _, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"wrong":true}`), "signal-bad", waiting.Version); err == nil {
		t.Fatal("wrong signal payload accepted")
	}
	signalled, duplicate, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"actor":"operator"}`), "signal-1", waiting.Version)
	if err != nil || duplicate || signalled.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("signal execution=%+v duplicate=%t err=%v", signalled, duplicate, err)
	}
	if _, duplicate, err = engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"actor":"operator"}`), "signal-1", 0); err != nil || !duplicate {
		t.Fatalf("duplicate signal duplicate=%t err=%v", duplicate, err)
	}
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("final=%+v", final)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	if len(journal) < 3 || journal[1].Kind != domain.WorkflowJournalWait || journal[2].Kind != domain.WorkflowJournalSignal {
		t.Fatalf("journal=%+v", journal)
	}
}

func TestCancelWinsAgainstLateSignalAndIsIdempotent(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 30}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, _, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "cancel-1")
	waiting := mustAdvance(t, engine, execution.ID)
	cancelled, duplicate, err := engine.Cancel(context.Background(), execution.ID, "cancel-op", "operator request", waiting.Version)
	if err != nil || duplicate || cancelled.Status != domain.WorkflowExecutionCancelled {
		t.Fatalf("cancelled=%+v duplicate=%t err=%v", cancelled, duplicate, err)
	}
	if _, duplicate, err = engine.Cancel(context.Background(), execution.ID, "cancel-op", "operator request", 0); err != nil || !duplicate {
		t.Fatalf("duplicate cancel duplicate=%t err=%v", duplicate, err)
	}
	if _, _, err = engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "late-signal", 0); err == nil {
		t.Fatal("late signal changed cancelled execution")
	}
}

func TestChildWorkflowPersistsIdentityAndAggregatesBudget(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "example/child", MaxDepth: 2}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "child"
	definition.Edges = []domain.WorkflowEdge{{From: "child", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	children := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = children
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parent-1")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.starts) != 1 {
		t.Fatalf("child starts=%d", len(children.starts))
	}
	childID := children.byKey[children.starts[0].IdempotencyKey]
	children.states[childID] = SubworkflowState{ExecutionID: childID, Terminal: true, Output: json.RawMessage(`{"ok":true}`), BudgetUsage: domain.WorkflowBudgetUsage{Turns: 2, Tokens: 30, NodeAttempts: 1}}
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded || final.BudgetUsage.Turns != 2 || final.BudgetUsage.Tokens != 30 {
		t.Fatalf("final=%+v", final)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 1 || attempts[0].ChildExecutionID == nil || *attempts[0].ChildExecutionID != childID {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestParallelBranchDispatchesDistinctProfilesAndJoins(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Expression: "true", Parallel: true}},
		{ID: "research", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "researcher", PromptTemplate: "research"}},
		{ID: "review", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "reviewer", PromptTemplate: "review"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "research"}, {From: "fanout", To: "review"}, {From: "research", To: "joined"}, {From: "review", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-1")
	mustAdvance(t, engine, execution.ID) // atomically persist membership and intents
	mustAdvance(t, engine, execution.ID) // dispatch member 1
	mustAdvance(t, engine, execution.ID) // dispatch member 2
	if len(children.requests) != 2 || children.requests[0].runID == children.requests[1].runID {
		t.Fatalf("parallel runs=%+v", children.requests)
	}
	profiles := map[string]bool{children.requests[0].profile: true, children.requests[1].profile: true}
	if !profiles["researcher"] || !profiles["reviewer"] {
		t.Fatalf("profiles=%v", profiles)
	}
	children.complete(children.requests[0].runID, "research handoff")
	children.complete(children.requests[1].runID, "review handoff")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("final=%+v", final)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 2 {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestConcurrentSignalAndCancelCommitExactlyOneOperation(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "gate", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "go", TimeoutSeconds: 30}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "gate"
	definition.Edges = []domain.WorkflowEdge{{From: "gate", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "race-1")
	waiting := mustAdvance(t, engine, execution.ID)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, _, _ = engine.Signal(context.Background(), execution.ID, "go", json.RawMessage(`{}`), "race-signal", waiting.Version)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _, _ = engine.Cancel(context.Background(), execution.ID, "race-cancel", "race", waiting.Version)
	}()
	close(start)
	wg.Wait()
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	operations := 0
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalSignal || entry.Kind == domain.WorkflowJournalCancel {
			operations++
		}
	}
	if operations != 1 {
		t.Fatalf("operations=%d journal=%+v", operations, journal)
	}
}

func TestSubworkflowRecoveryReusesPersistedChildIntent(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "example/child", MaxDepth: 2}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "child"
	definition.Edges = []domain.WorkflowEdge{{From: "child", To: "done"}}
	engine, store, runs := testEngine(t, definition)
	children := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = children
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "child-recovery")
	mustAdvance(t, engine, execution.ID)
	expressions, _ := NewExpressionEvaluator()
	restarted := &Engine{Store: store, Catalog: fakeCatalog{revision(definition)}, Children: runs, Subworkflows: children, Expressions: expressions}
	mustAdvance(t, restarted, execution.ID)
	mustAdvance(t, restarted, execution.ID)
	if len(children.starts) != 1 {
		t.Fatalf("child dispatches=%d", len(children.starts))
	}
}

func TestBindingsAreBoundedAndDoNotExposeTranscript(t *testing.T) { // [REQ:REQ-P2-001]
	ctx := BindingContext{Input: json.RawMessage(`{"topic":"x"}`)}
	binding := inputBinding("topic", "$.topic")
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx)
	if err != nil || values["topic"] != "x" {
		t.Fatalf("values=%v err=%v", values, err)
	}
	binding.Source = domain.WorkflowBindingSource("transcript")
	if _, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx); err == nil {
		t.Fatal("transcript source accepted")
	}
	binding = inputBinding("topic", "$.topic")
	binding.MaxBytes = 1
	if _, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx); err == nil {
		t.Fatal("oversized binding accepted")
	}
}

func TestExpressionEnvironmentRejectsNonBooleanAndUnknownNames(t *testing.T) {
	e, err := NewExpressionEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	ok, err := e.Evaluate(`input.allowed && iteration < 2`, ExpressionContext{Input: map[string]any{"allowed": true}, Iteration: 1, EdgeTraversals: map[string]int{}, Budget: map[string]any{}})
	if err != nil || !ok {
		t.Fatalf("ok=%t err=%v", ok, err)
	}
	if _, err := e.Evaluate(`now()`, ExpressionContext{}); err == nil {
		t.Fatal("undeclared time function accepted")
	}
	if _, err := e.Evaluate(`input`, ExpressionContext{}); err == nil {
		t.Fatal("non-boolean expression accepted")
	}
}

func inputBinding(name, path string) domain.WorkflowInputBinding {
	return domain.WorkflowInputBinding{Name: name, Source: domain.WorkflowBindingInput, Selector: path, Order: "asc", Limit: 1, MaxBytes: 4096, RenderAs: "json", MissingPolicy: "error"}
}

func baseDefinition() domain.WorkflowDefinition {
	schema := json.RawMessage(`{"type":"object","additionalProperties":true}`)
	return domain.WorkflowDefinition{SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "example", Key: "example/flow", Version: "1.0.0", InputSchema: schema, OutputSchema: schema, Budgets: domain.WorkflowBudgets{WallTimeSeconds: 600, MaxTurns: 10, MaxTokens: 10000, MaxCostUSD: 10, MaxNodeAttempts: 10, MaxChildren: 10, MaxConcurrency: 2, MaxRecursion: 2, MaxRetries: 2, MaxWaitSeconds: 60}}
}

func revision(d domain.WorkflowDefinition) *domain.WorkflowRevision {
	return &domain.WorkflowRevision{ID: uuid.New(), Owner: d.Owner, Key: d.Key, SemanticVersion: d.Version, Digest: "sha256:test", Definition: d, Active: true, CreatedAt: time.Now()}
}

func testEngine(t *testing.T, d domain.WorkflowDefinition) (*Engine, *memoryStore, *fakeChildren) {
	t.Helper()
	expressions, err := NewExpressionEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	store := newMemoryStore()
	children := &fakeChildren{states: map[uuid.UUID]ChildState{}, byKey: map[string]uuid.UUID{}}
	return &Engine{Store: store, Catalog: fakeCatalog{revision(d)}, Children: children, Expressions: expressions}, store, children
}

func mustAdvance(t *testing.T, e *Engine, id uuid.UUID) *domain.WorkflowExecution {
	t.Helper()
	x, err := e.Advance(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

type fakeCatalog struct{ revision *domain.WorkflowRevision }

func (f fakeCatalog) GetByDigest(context.Context, string) (*domain.WorkflowRevision, error) {
	return f.revision, nil
}

type (
	childCall struct {
		runID   uuid.UUID
		source  *uuid.UUID
		prompt  string
		profile string
	}
	fakeChildren struct {
		requests []childCall
		states   map[uuid.UUID]ChildState
		byKey    map[string]uuid.UUID
	}
)

type fakeSubworkflows struct {
	starts []SubworkflowRequest
	states map[uuid.UUID]SubworkflowState
	byKey  map[string]uuid.UUID
}

func (f *fakeSubworkflows) Start(_ context.Context, req SubworkflowRequest) (SubworkflowState, error) {
	id, ok := f.byKey[req.IdempotencyKey]
	if !ok {
		id = uuid.New()
		f.byKey[req.IdempotencyKey] = id
		f.starts = append(f.starts, req)
	}
	state := f.states[id]
	state.ExecutionID = id
	f.states[id] = state
	return state, nil
}

func (f *fakeSubworkflows) Inspect(_ context.Context, id uuid.UUID) (SubworkflowState, error) {
	return f.states[id], nil
}

func (f *fakeSubworkflows) Cancel(_ context.Context, id uuid.UUID, _ string) error {
	state := f.states[id]
	state.Terminal, state.Failed = true, true
	f.states[id] = state
	return nil
}

func (f *fakeChildren) launch(req ChildRequest) (ChildState, error) {
	id, ok := f.byKey[req.IdempotencyKey]
	if !ok {
		id = uuid.New()
		f.byKey[req.IdempotencyKey] = id
		f.requests = append(f.requests, childCall{runID: id, source: req.SourceRunID, prompt: req.Prompt, profile: req.ProfileKey})
	}
	state := f.states[id]
	state.RunID = id
	state.ConversationID = "conversation-" + id.String()
	f.states[id] = state
	return state, nil
}

func (f *fakeChildren) StartFresh(_ context.Context, r ChildRequest) (ChildState, error) {
	return f.launch(r)
}

func (f *fakeChildren) Continue(_ context.Context, r ChildRequest) (ChildState, error) {
	return f.launch(r)
}

func (f *fakeChildren) Inspect(_ context.Context, id uuid.UUID) (ChildState, error) {
	return f.states[id], nil
}

func (f *fakeChildren) complete(id uuid.UUID, text string) {
	state := f.states[id]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: text}
	f.states[id] = state
}

type memoryStore struct {
	mu         sync.Mutex
	executions map[uuid.UUID]*domain.WorkflowExecution
	keys       map[string]uuid.UUID
	attempts   map[uuid.UUID][]*domain.WorkflowNodeAttempt
	journal    map[uuid.UUID][]*domain.WorkflowJournalEntry
}

func newMemoryStore() *memoryStore {
	return &memoryStore{executions: map[uuid.UUID]*domain.WorkflowExecution{}, keys: map[string]uuid.UUID{}, attempts: map[uuid.UUID][]*domain.WorkflowNodeAttempt{}, journal: map[uuid.UUID][]*domain.WorkflowJournalEntry{}}
}

func clone[T any](v T) T {
	data, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(data, &out)
	return out
}

func (s *memoryStore) Create(_ context.Context, e *domain.WorkflowExecution, j *domain.WorkflowJournalEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	x := clone(*e)
	s.executions[e.ID] = &x
	s.keys[e.IdempotencyKey] = e.ID
	entry := clone(*j)
	s.journal[e.ID] = []*domain.WorkflowJournalEntry{&entry}
	return nil
}

func (s *memoryStore) Get(_ context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.executions[id] == nil {
		return nil, nil
	}
	x := clone(*s.executions[id])
	return &x, nil
}

func (s *memoryStore) GetByIdempotencyKey(ctx context.Context, key string) (*domain.WorkflowExecution, error) {
	s.mu.Lock()
	id, ok := s.keys[key]
	s.mu.Unlock()
	if !ok {
		return nil, nil
	}
	return s.Get(ctx, id)
}

func (s *memoryStore) List(_ context.Context, filter repository.WorkflowExecutionListFilter) ([]*domain.WorkflowExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*domain.WorkflowExecution
	for _, execution := range s.executions {
		if filter.Owner != "" && execution.Owner != filter.Owner || filter.WorkflowKey != "" && execution.WorkflowKey != filter.WorkflowKey || filter.Status != "" && execution.Status != filter.Status {
			continue
		}
		value := clone(*execution)
		out = append(out, &value)
	}
	return out, nil
}

func (s *memoryStore) Commit(_ context.Context, c repository.WorkflowCommit) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.executions[c.Execution.ID]
	if current == nil || current.Version != c.ExpectedVersion {
		return false, nil
	}
	x := clone(*c.Execution)
	s.executions[x.ID] = &x
	if c.Attempt != nil {
		list := s.attempts[x.ID]
		updated := false
		for i, a := range list {
			if a.ID == c.Attempt.ID {
				v := clone(*c.Attempt)
				list[i] = &v
				updated = true
			}
		}
		if !updated {
			v := clone(*c.Attempt)
			list = append(list, &v)
		}
		s.attempts[x.ID] = list
	}
	for _, attempt := range c.Attempts {
		list := s.attempts[x.ID]
		v := clone(*attempt)
		list = append(list, &v)
		s.attempts[x.ID] = list
	}
	for _, j := range c.Journal {
		v := clone(*j)
		s.journal[x.ID] = append(s.journal[x.ID], &v)
	}
	return true, nil
}

func (s *memoryStore) GetAttemptByIdempotencyKey(_ context.Context, key string) (*domain.WorkflowNodeAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, list := range s.attempts {
		for _, a := range list {
			if a.IdempotencyKey == key {
				v := clone(*a)
				return &v, nil
			}
		}
	}
	return nil, nil
}

func (s *memoryStore) ListAttempts(_ context.Context, id uuid.UUID) ([]*domain.WorkflowNodeAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.attempts[id]), nil
}

func (s *memoryStore) ListJournal(_ context.Context, id uuid.UUID, after int64, limit int) ([]*domain.WorkflowJournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*domain.WorkflowJournalEntry
	for _, j := range s.journal[id] {
		if j.Sequence > after {
			v := clone(*j)
			out = append(out, &v)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *memoryStore) ListRecoverable(_ context.Context, _ int) ([]*domain.WorkflowExecution, error) {
	return nil, nil
}
