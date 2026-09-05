package workflowruntime

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

func TestBindingPresentationVocabularyAndTruncationAreDeterministic(t *testing.T) {
	ctx := BindingContext{Input: json.RawMessage(`{"title":"Hello","object":{"a":1}}`)}
	cases := []struct {
		name    string
		binding domain.WorkflowInputBinding
		want    string
	}{
		{"text", domain.WorkflowInputBinding{Name: "title", Source: domain.WorkflowBindingInput, Selector: "$.title", Limit: 1, MaxBytes: 100, RenderAs: "text", MissingPolicy: "error"}, "Hello"},
		{"pretty", domain.WorkflowInputBinding{Name: "object", Source: domain.WorkflowBindingInput, Selector: "$.object", Limit: 1, MaxBytes: 100, RenderAs: "json_pretty", MissingPolicy: "error"}, "{\n  \"a\": 1\n}"},
		{"xml", domain.WorkflowInputBinding{Name: "title", Source: domain.WorkflowBindingInput, Selector: "$.title", Limit: 1, MaxBytes: 100, RenderAs: "xml", WrapTag: "context", MissingPolicy: "error"}, "<context>\nHello\n</context>"},
		{"markdown", domain.WorkflowInputBinding{Name: "title", Source: domain.WorkflowBindingInput, Selector: "$.title", Limit: 1, MaxBytes: 100, RenderAs: "markdown", MissingPolicy: "error"}, "## title\n\nHello"},
		{"fenced", domain.WorkflowInputBinding{Name: "title", Source: domain.WorkflowBindingInput, Selector: "$.title", Limit: 1, MaxBytes: 100, RenderAs: "fenced", Lang: "text", MissingPolicy: "error"}, "```text\nHello\n```"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first, err := EvaluateBindings([]domain.WorkflowInputBinding{tc.binding}, ctx)
			if err != nil || first[tc.binding.Name] != tc.want {
				t.Fatalf("first=%#v err=%v", first, err)
			}
			second, err := EvaluateBindings([]domain.WorkflowInputBinding{tc.binding}, ctx)
			if err != nil || second[tc.binding.Name] != first[tc.binding.Name] {
				t.Fatalf("renderer was not deterministic: second=%#v err=%v", second, err)
			}
		})
	}
	truncated := domain.WorkflowInputBinding{Name: "title", Source: domain.WorkflowBindingInput, Selector: "$.title", Limit: 1, MaxBytes: 30, RenderAs: "text", Overflow: "truncate", MissingPolicy: "error"}
	values, diagnostics, err := EvaluateBindingsWithDiagnostics([]domain.WorkflowInputBinding{truncated}, BindingContext{Input: json.RawMessage(`{"title":"this is deliberately much longer than the binding budget"}`)})
	if err != nil || !strings.Contains(values["title"].(string), "truncated") || len(values["title"].(string)) > truncated.MaxBytes {
		t.Fatalf("truncation=%#v err=%v", values, err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "binding_truncated" || diagnostics[0].DroppedBytes <= 0 {
		t.Fatalf("truncation diagnostics=%+v", diagnostics)
	}
}

func TestXMLListBindingEvictsWholeProvenanceTaggedItems(t *testing.T) {
	ctx := BindingContext{Journal: []*domain.WorkflowJournalEntry{
		{Kind: domain.WorkflowJournalHandoff, NodeID: "slice", Sequence: 1, Payload: json.RawMessage(`{"summary":"first framing handoff"}`)},
		{Kind: domain.WorkflowJournalHandoff, NodeID: "slice", Sequence: 2, Payload: json.RawMessage(`{"summary":"middle handoff that should be elided"}`)},
		{Kind: domain.WorkflowJournalHandoff, NodeID: "slice", Sequence: 3, Payload: json.RawMessage(`{"summary":"latest handoff"}`)},
	}}
	binding := domain.WorkflowInputBinding{Name: "priorHandoffs", Source: domain.WorkflowBindingHandoff, Selector: "node=slice;$.summary", Limit: 10, MaxBytes: 260, RenderAs: "xml", ItemTag: "handoff", ItemMaxBytes: 16, EvictionPolicy: "keep_ends", KeepFirst: 1, MissingPolicy: "error"}
	values, diagnostics, err := EvaluateBindingsWithDiagnostics([]domain.WorkflowInputBinding{binding}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	rendered := values["priorHandoffs"].(string)
	if len(rendered) > binding.MaxBytes || !strings.Contains(rendered, `sequence="1"`) || !strings.Contains(rendered, `sequence="3"`) || !strings.Contains(rendered, "elided") || !strings.Contains(rendered, "truncated") {
		t.Fatalf("rendered list=%q", rendered)
	}
	var itemClamp, eviction bool
	for _, diagnostic := range diagnostics {
		itemClamp = itemClamp || diagnostic.Code == "binding_item_truncated"
		eviction = eviction || diagnostic.Code == "binding_items_evicted"
	}
	if !itemClamp || !eviction {
		t.Fatalf("list diagnostics=%+v", diagnostics)
	}
	again, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx)
	if err != nil || again["priorHandoffs"] != rendered {
		t.Fatalf("list rendering is nondeterministic: %#v err=%v", again, err)
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
	return domain.WorkflowDefinition{SchemaVersion: domain.WorkflowSchemaVersionV1, Owner: "example", Key: "example/flow", Version: "1.0.0", InputSchema: schema, OutputSchema: schema, Budgets: domain.WorkflowBudgets{WallTimeSeconds: 600, MaxTurns: 10, MaxTokens: 10000, MaxChargeMicroUSD: 10, MaxNodeAttempts: 10, MaxChildren: 10, MaxConcurrency: 2, MaxRecursion: 2, MaxRetries: 2, MaxWaitSeconds: 60}}
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
		runID     uuid.UUID
		source    *uuid.UUID
		prompt    string
		profile   string
		scopePath string
		maxTurns  int
		timeout   time.Duration
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
		f.requests = append(f.requests, childCall{runID: id, source: req.SourceRunID, prompt: req.Prompt, profile: req.ProfileKey, scopePath: req.ScopePath, maxTurns: req.MaxTurns, timeout: req.Timeout})
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

func (f *fakeChildren) Stop(_ context.Context, id uuid.UUID) error {
	state := f.states[id]
	state.Terminal, state.Failed = true, true
	f.states[id] = state
	return nil
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
		updated := false
		for i, existing := range list {
			if existing.ID == attempt.ID {
				v := clone(*attempt)
				list[i] = &v
				updated = true
				break
			}
		}
		if !updated {
			v := clone(*attempt)
			list = append(list, &v)
		}
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

func (s *memoryStore) ExecutionIDForRun(_ context.Context, runID uuid.UUID) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var newest *domain.WorkflowNodeAttempt
	for execID := range s.attempts {
		for _, attempt := range s.attempts[execID] {
			if attempt.RunID == nil || *attempt.RunID != runID {
				continue
			}
			if newest == nil || attempt.CreatedAt.After(newest.CreatedAt) {
				newest = attempt
			}
		}
	}
	if newest == nil {
		return uuid.Nil, nil
	}
	return newest.ExecutionID, nil
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
