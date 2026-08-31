package workflowruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowcatalog"

	"github.com/google/uuid"
)

type fixedPromptResolver struct{ resolution PromptResolution }

func (f fixedPromptResolver) Resolve(_ context.Context, _ *domain.WorkflowPromptRef, _ PromptAssignmentIdentity) (PromptResolution, error) {
	return f.resolution, nil
}

func TestEngineFailPersistsExternalWorkerFailure(t *testing.T) {
	engine, store, _ := testEngine(t, baseDefinition())
	execution, err := engine.Start(context.Background(), revision(baseDefinition()), json.RawMessage(`{}`), "external-failure")
	if err != nil {
		t.Fatal(err)
	}
	failed, err := engine.Fail(context.Background(), execution.ID, "nudger_panic", "panic: injected")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.WorkflowExecutionFailed || failed.TerminalReason == nil || failed.TerminalReason.Code != "nudger_panic" {
		t.Fatalf("failed execution = %+v", failed)
	}
	persisted, err := store.Get(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != domain.WorkflowExecutionFailed || persisted.TerminalReason == nil || persisted.TerminalReason.Message != "panic: injected" {
		t.Fatalf("persisted execution = %+v", persisted)
	}
}

func TestEngineRecordDiagnosticSequencesEvidenceWithoutRefreshingLifecycleProgress(t *testing.T) {
	engine, store, _ := testEngine(t, baseDefinition())
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	engine.Now = func() time.Time { return now }
	execution, err := engine.Start(context.Background(), revision(baseDefinition()), json.RawMessage(`{}`), "diagnostic-sequence")
	if err != nil {
		t.Fatal(err)
	}
	progressAt := execution.UpdatedAt
	now = now.Add(time.Hour)
	if _, err := engine.RecordDiagnostic(context.Background(), execution.ID, "first", "first diagnostic"); err != nil {
		t.Fatalf("record first diagnostic: %v", err)
	}
	if _, err := engine.RecordDiagnostic(context.Background(), execution.ID, "second", "second diagnostic"); err != nil {
		t.Fatalf("record second diagnostic: %v", err)
	}
	journal, err := store.ListJournal(context.Background(), execution.ID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(journal) != 3 || journal[1].Sequence != 2 || journal[2].Sequence != 3 {
		t.Fatalf("diagnostic journal sequence = %+v, want input then sequences 2 and 3", journal)
	}
	persisted, err := store.Get(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !persisted.UpdatedAt.Equal(progressAt) {
		t.Fatalf("diagnostics refreshed lifecycle progress: got %s, want %s", persisted.UpdatedAt, progressAt)
	}
}

func TestEngine_AssignsArmedPromptAtAttemptCreation(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "work", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.fast", PromptRef: &domain.WorkflowPromptRef{SkillID: "skill", ExperimentID: "exp-1"}, Bindings: []domain.WorkflowInputBinding{inputBinding("topic", "$.topic")}}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "work"
	definition.Edges = []domain.WorkflowEdge{{From: "work", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	engine.PromptResolver = fixedPromptResolver{resolution: PromptResolution{Content: "Assigned {{.topic}}", ExperimentID: "exp-1", VariantID: "treatment", ContentHash: "sha256:assigned"}}
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{"topic":"A"}`), "armed-attempt")
	if err != nil {
		t.Fatal(err)
	}
	advanced := mustAdvance(t, engine, execution.ID)
	attempts, err := store.ListAttempts(context.Background(), execution.ID)
	if err != nil || len(attempts) != 1 {
		t.Fatalf("advanced=%+v reason=%+v attempts=%+v err=%v", advanced, advanced.TerminalReason, attempts, err)
	}
	got := attempts[0]
	if got.Status != domain.WorkflowAttemptDispatchPending || got.PromptSnapshot != "Assigned A" || got.ExperimentID != "exp-1" || got.VariantID != "treatment" || got.PromptHash != "sha256:assigned" {
		t.Fatalf("assignment=%+v", got)
	}
}

func TestEngineLoopCreatesDistinctFreshRunsAndTerminates(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "slice", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.fast", PromptTemplate: "Work {{.topic}}", Bindings: []domain.WorkflowInputBinding{inputBinding("topic", "$.topic")}}}, {ID: "more", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
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

func TestEngineRendersRunScopePathFromDeclaredBinding(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{
		RoleRef: "code.default", PromptTemplate: "work", ScopePathTemplate: "scenarios/{{.scope_scenario}}",
		Bindings: []domain.WorkflowInputBinding{{Name: "scope_scenario", Source: domain.WorkflowBindingInput, Selector: "$.targetScenario", Limit: 1, MaxBytes: 255, RenderAs: "text", MissingPolicy: "error"}},
	}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, _, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{"targetScenario":"demo"}`), "scoped-run")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 1 || children.requests[0].scopePath != "scenarios/demo" {
		t.Fatalf("scope path = %+v, want scenarios/demo", children.requests)
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

func TestExecutionCostBudgetUsesAuthoritativeChargeOnly(t *testing.T) {
	definition := baseDefinition()
	definition.Budgets.MaxChargeMicroUSD = 1
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work"}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, _, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "charge-budget-1")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	runID := children.requests[0].runID
	state := children.states[runID]
	state.Terminal = true
	state.CostUSD = 100 // historical estimate must not exhaust a charge budget
	state.ChargeMicroUSD = 2
	children.states[runID] = state
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionBudgetExhausted || final.TerminalReason.BudgetName != "charge" || final.BudgetUsage.ChargeMicroUSD != 2 {
		t.Fatalf("execution=%+v", final)
	}
}

func TestInvalidStructuredResultPreservesRawAttemptEvidence(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work", ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionDeterministic, Schema: json.RawMessage(`{"type":"object","required":["answer"]}`)}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "invalid-structured")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	runID := children.requests[0].runID
	state := children.states[runID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: `{"wrong":true}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_mismatch", Path: "$.answer", Message: "required property missing"}}}}
	children.states[runID] = state
	repairPending := mustAdvance(t, engine, execution.ID)
	if repairPending.Status != domain.WorkflowExecutionRunning || repairPending.BudgetUsage.NodeAttempts != 2 {
		t.Fatalf("repairPending=%+v", repairPending)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 2 || attempts[0].RawOutput != `{"wrong":true}` || !strings.Contains(attempts[0].ValidationError, "schema_mismatch") || attempts[1].Strategy != domain.WorkflowAttemptContinue || attempts[1].SourceAttemptID == nil || *attempts[1].SourceAttemptID != attempts[0].ID {
		t.Fatalf("attempt evidence=%+v", attempts)
	}
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[1].source == nil || *children.requests[1].source != runID {
		t.Fatalf("repair did not continue the failed run: %+v", children.requests)
	}
	repairRunID := children.requests[1].runID
	state = children.states[repairRunID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: `{"answer":"fixed"}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: json.RawMessage(`{"answer":"fixed"}`)}}
	children.states[repairRunID] = state
	mustAdvance(t, engine, execution.ID)
	terminal := mustAdvance(t, engine, execution.ID)
	if terminal.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("repaired result did not finish workflow: %+v", terminal)
	}
}

func TestResultSpecSchemaIsInjectedIntoPromptAndRepair(t *testing.T) {
	schema := `{"type":"object","required":["answer"]}`
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work {{.topic}}", Bindings: []domain.WorkflowInputBinding{inputBinding("topic", "$.topic")}, ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionDeterministic, Schema: json.RawMessage(schema)}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{"topic":"A"}`), "schema-in-prompt")
	mustAdvance(t, engine, execution.ID)
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 1 {
		t.Fatalf("attempts=%+v", attempts)
	}
	prompt := attempts[0].PromptSnapshot
	if !strings.HasPrefix(prompt, "work A") || !strings.Contains(prompt, "## Required structured result") || !strings.Contains(prompt, schema) {
		t.Fatalf("run prompt missing schema block: %q", prompt)
	}
	mustAdvance(t, engine, execution.ID)
	runID := children.requests[0].runID
	state := children.states[runID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: `{"wrong":true}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_mismatch", Path: "$.answer", Message: "required property missing"}}}}
	children.states[runID] = state
	mustAdvance(t, engine, execution.ID)
	attempts, _ = store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 2 {
		t.Fatalf("expected repair attempt: %+v", attempts)
	}
	repairPrompt := attempts[1].PromptSnapshot
	if !strings.Contains(repairPrompt, "Required schema:") || !strings.Contains(repairPrompt, schema) || !strings.Contains(repairPrompt, "schema_mismatch") {
		t.Fatalf("repair prompt missing schema: %q", repairPrompt)
	}
}

func TestInvalidStructuredResultCanDisableSchemaRepair(t *testing.T) {
	disabled := 0
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work", ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionDeterministic, Schema: json.RawMessage(`{"type":"object","required":["answer"]}`), SchemaRepairAttempts: &disabled}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "disabled-structured-repair")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	state := children.states[children.requests[0].runID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: `{"wrong":true}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_mismatch"}}}}
	children.states[children.requests[0].runID] = state
	mustAdvance(t, engine, execution.ID)
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 1 || attempts[0].RawOutput == "" || attempts[0].ValidationError == "" || attempts[0].Status != domain.WorkflowAttemptFailed {
		t.Fatalf("disabled repair attempt evidence=%+v", attempts)
	}
}

func TestInvalidStructuredRepairIsBoundedAndAccounted(t *testing.T) {
	definition := baseDefinition()
	definition.Budgets.MaxNodeAttempts = 2
	definition.Budgets.MaxChildren = 2
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work", ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionDeterministic, Schema: json.RawMessage(`{"type":"object","required":["answer"]}`)}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "bounded-structured-repair")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)

	invalid := func(runID uuid.UUID) {
		state := children.states[runID]
		state.Terminal = true
		state.Result = &domain.RunResult{FinalOutput: `{"wrong":true}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_mismatch"}}}}
		children.states[runID] = state
	}
	invalid(children.requests[0].runID)
	repairPending := mustAdvance(t, engine, execution.ID)
	if repairPending.Status != domain.WorkflowExecutionRunning || repairPending.BudgetUsage.NodeAttempts != 2 {
		t.Fatalf("repairPending=%+v", repairPending)
	}
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 {
		t.Fatalf("repair dispatches=%d, want 2", len(children.requests))
	}
	invalid(children.requests[1].runID)
	terminal := mustAdvance(t, engine, execution.ID)
	if terminal.Status != domain.WorkflowExecutionFailed || terminal.TerminalReason.Code != "structured_result_invalid" || terminal.BudgetUsage.NodeAttempts != 2 || terminal.BudgetUsage.Children != 2 {
		t.Fatalf("terminal=%+v", terminal)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 2 || schemaRepairCount(attempts, "run") != 1 {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestInvalidStructuredResultDoesNotScheduleRepairWithoutBudget(t *testing.T) {
	for _, tc := range []struct {
		name            string
		maxNodeAttempts int
	}{
		{name: "node_attempt_capacity", maxNodeAttempts: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := baseDefinition()
			definition.Budgets.MaxNodeAttempts = tc.maxNodeAttempts
			definition.Budgets.MaxChildren = 1
			definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work", ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionDeterministic, Schema: json.RawMessage(`{"type":"object","required":["answer"]}`)}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
			definition.EntryNode = "run"
			definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
			engine, store, children := testEngine(t, definition)
			execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "structured-repair-no-budget-"+tc.name)
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			state := children.states[children.requests[0].runID]
			state.Terminal = true
			state.Result = &domain.RunResult{FinalOutput: `{"wrong":true}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_mismatch"}}}}
			children.states[children.requests[0].runID] = state
			terminal := mustAdvance(t, engine, execution.ID)
			if terminal.Status != domain.WorkflowExecutionFailed || terminal.TerminalReason.Code != "structured_result_invalid" || terminal.BudgetUsage.NodeAttempts != 1 || terminal.BudgetUsage.Children != 1 {
				t.Fatalf("terminal=%+v", terminal)
			}
			attempts, _ := store.ListAttempts(context.Background(), execution.ID)
			if len(attempts) != 1 || len(children.requests) != 1 {
				t.Fatalf("attempts=%+v children=%+v", attempts, children.requests)
			}
		})
	}
}

func TestInvalidStructuredResultRepairsWithinSingleChildBudget(t *testing.T) {
	definition := baseDefinition()
	definition.Budgets.MaxNodeAttempts = 2
	definition.Budgets.MaxChildren = 1
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work", ResultSpec: &domain.ResultSpec{Version: "result-spec/v1", Kind: domain.ResultSpecKindJSONSchema, ExtractionMode: domain.StructuredExtractionDeterministic, Schema: json.RawMessage(`{"type":"object","required":["answer"]}`)}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, _, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "structured-repair-single-child")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	state := children.states[children.requests[0].runID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: `{"wrong":true}`, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_mismatch"}}}}
	children.states[children.requests[0].runID] = state
	repairPending := mustAdvance(t, engine, execution.ID)
	if repairPending.Status != domain.WorkflowExecutionRunning || repairPending.BudgetUsage.NodeAttempts != 2 || repairPending.BudgetUsage.Children != 1 {
		t.Fatalf("repairPending=%+v", repairPending)
	}
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[1].source == nil || *children.requests[1].source != children.requests[0].runID {
		t.Fatalf("repair continuation=%+v", children.requests)
	}
}

func TestBindingClampIsJournaledWithAttempt(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "{{.context}}", Bindings: []domain.WorkflowInputBinding{{Name: "context", Source: domain.WorkflowBindingInput, Selector: "$.context", Limit: 1, MaxBytes: 36, RenderAs: "text", Overflow: "truncate", MissingPolicy: "error"}}}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{"context":"this input is deliberately longer than the small binding budget"}`), "binding-diagnostic")
	mustAdvance(t, engine, execution.ID)
	journal, err := store.ListJournal(context.Background(), execution.ID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range journal {
		if entry.Kind != domain.WorkflowJournalDiagnostic {
			continue
		}
		var diagnostic BindingDiagnostic
		if err := json.Unmarshal(entry.Payload, &diagnostic); err != nil {
			t.Fatal(err)
		}
		if diagnostic.Code == "binding_truncated" && diagnostic.Binding == "context" && diagnostic.DroppedBytes > 0 {
			return
		}
	}
	t.Fatalf("binding truncation journal missing: %+v", journal)
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

func TestSignalBuffersBeforeWaitArmsAndConsumesOnArrival(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work"}},
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", PayloadSchema: json.RawMessage(`{"type":"object","required":["actor"],"properties":{"actor":{"type":"string"}},"additionalProperties":false}`), TimeoutSeconds: 30}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "approval"}, {From: "approval", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "signal-before-wait")
	mustAdvance(t, engine, execution.ID) // persist dispatch intent
	mustAdvance(t, engine, execution.ID) // dispatch run
	children.complete(children.requests[0].runID, "complete")

	// The run has completed, but the nudge has not yet driven the execution to
	// its wait node. The declared contract still accepts and durably buffers the
	// signal.
	buffered, duplicate, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"actor":"operator"}`), "approval-before-arm", 0)
	if err != nil || duplicate || buffered.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("buffered signal execution=%+v duplicate=%t err=%v", buffered, duplicate, err)
	}

	// Completion progression arms and consumes the buffer without a caller-side
	// advance between the terminal run and Signal.
	advanced := mustAdvance(t, engine, execution.ID)
	if advanced.CurrentNodeID != "approval" || advanced.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("wait did not consume buffered signal: %+v", advanced)
	}
	consumed := mustAdvance(t, engine, execution.ID)
	if consumed.CurrentNodeID != "done" || consumed.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("buffered signal was not consumed: %+v", consumed)
	}
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("final=%+v", final)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	var signalEntry *domain.WorkflowJournalEntry
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalSignal {
			signalEntry = entry
			break
		}
	}
	if signalEntry == nil || signalEntry.NodeID != "approval" {
		t.Fatalf("buffered signal journal=%+v", journal)
	}
}

func TestBoundedWaitRoutesTimeoutAndPreservesReason(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 10, OnTimeout: "timed-out"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
		{ID: "timed-out", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "blocked"}},
	}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	engine.Now = func() time.Time { return now }
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "wait-timeout")
	if waiting := mustAdvance(t, engine, execution.ID); waiting.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("waiting=%+v", waiting)
	}
	now = now.Add(11 * time.Second)
	routed := mustAdvance(t, engine, execution.ID)
	if routed.CurrentNodeID != "timed-out" || routed.Status != domain.WorkflowExecutionRunning || routed.TerminalReason == nil || routed.TerminalReason.Code != "wait_timeout" {
		t.Fatalf("routed=%+v", routed)
	}
	terminal := mustAdvance(t, engine, execution.ID)
	if terminal.Status != domain.WorkflowExecutionBlocked || terminal.TerminalReason == nil || terminal.TerminalReason.Code != "wait_timeout" {
		t.Fatalf("terminal=%+v", terminal)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	if journal[len(journal)-1].Kind != domain.WorkflowJournalWaitTimeout {
		t.Fatalf("timeout journal missing: %+v", journal)
	}
}

func TestIndefiniteWaitAndWallTimeExcludePausedDuration(t *testing.T) {
	definition := baseDefinition()
	definition.Budgets.WallTimeSeconds = 5
	definition.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 0}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, _, _ := testEngine(t, definition)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	engine.Now = func() time.Time { return now }
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "indefinite-wait")
	mustAdvance(t, engine, execution.ID)
	now = now.Add(24 * time.Hour)
	stillWaiting := mustAdvance(t, engine, execution.ID)
	if stillWaiting.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("indefinite wait exhausted wall time: %+v", stillWaiting)
	}
	if _, _, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "after-day", 0); err != nil {
		t.Fatal(err)
	}
	now = now.Add(6 * time.Second)
	mustAdvance(t, engine, execution.ID)
	terminal := mustAdvance(t, engine, execution.ID)
	if terminal.Status != domain.WorkflowExecutionBudgetExhausted || terminal.TerminalReason == nil || terminal.TerminalReason.BudgetName != "wall_time" {
		t.Fatalf("active time after wait did not exhaust wall budget: %+v", terminal)
	}
}

func TestRevisitedWaitRequiresVisitScopedSignal(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 30}},
		{ID: "loop", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
	}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "loop", MaxTraversals: 2}, {From: "loop", To: "approval", MaxTraversals: 1}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "wait-revisit")
	firstWait := mustAdvance(t, engine, execution.ID)
	if _, _, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "approval-1", firstWait.Version); err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID) // consume first wait
	mustAdvance(t, engine, execution.ID) // loop back to approval
	secondWait := mustAdvance(t, engine, execution.ID)
	if secondWait.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("revisited wait reused prior signal: %#v", secondWait)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	correlations := []string{}
	for _, entry := range journal {
		if entry.Kind != domain.WorkflowJournalWait {
			continue
		}
		var intent waitIntent
		if json.Unmarshal(entry.Payload, &intent) == nil {
			correlations = append(correlations, intent.CorrelationKey)
		}
	}
	if len(correlations) != 2 || correlations[0] == correlations[1] {
		t.Fatalf("wait correlations are not visit scoped: %v", correlations)
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
	if err != nil || duplicate || cancelled.Status != domain.WorkflowExecutionCancelling {
		t.Fatalf("cancelled=%+v duplicate=%t err=%v", cancelled, duplicate, err)
	}
	if advanced, err := engine.Advance(context.Background(), execution.ID); err != nil || advanced.Status != domain.WorkflowExecutionCancelling {
		t.Fatalf("advance after cancellation=%+v err=%v; cancelling must be a hard barrier until cleanup settles it", advanced, err)
	}
	cancelled, err = engine.RecordCancellationDisposition(context.Background(), execution.ID, 0, 0, nil)
	if err != nil || cancelled.Status != domain.WorkflowExecutionCancelled {
		t.Fatalf("cancellation cleanup=%+v err=%v", cancelled, err)
	}
	if _, duplicate, err = engine.Cancel(context.Background(), execution.ID, "cancel-op", "operator request", 0); err != nil || !duplicate {
		t.Fatalf("duplicate cancel duplicate=%t err=%v", duplicate, err)
	}
	if _, _, err = engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "late-signal", 0); err == nil {
		t.Fatal("late signal changed cancelled execution")
	}
}

func TestCancellationRemainsRecoverableUntilChildCleanupSucceeds(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 30}}}
	definition.EntryNode = "approval"
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "cancel-cleanup")
	waiting := mustAdvance(t, engine, execution.ID)
	if _, _, err := engine.Cancel(context.Background(), execution.ID, "cancel-cleanup", "operator request", waiting.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.RecordCleanupDisposition(context.Background(), execution.ID, 0, 0, []string{"child still running"}); err == nil {
		t.Fatal("incomplete cleanup was accepted")
	}
	persisted, _ := store.Get(context.Background(), execution.ID)
	if persisted.Status != domain.WorkflowExecutionCancelling || persisted.EndedAt != nil {
		t.Fatalf("incomplete cancellation became terminal: %#v", persisted)
	}
	completed, err := engine.RecordCleanupDisposition(context.Background(), execution.ID, 1, 0, nil)
	if err != nil || completed.Status != domain.WorkflowExecutionCancelled || completed.EndedAt == nil {
		t.Fatalf("completed cancellation cleanup=%#v err=%v", completed, err)
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

func TestChildWorkflowPreservesDistinctTerminalStatus(t *testing.T) { // [REQ:REQ-P2-001]
	for _, want := range []domain.WorkflowExecutionStatus{domain.WorkflowExecutionBlocked, domain.WorkflowExecutionAbstained} {
		t.Run(string(want), func(t *testing.T) {
			definition := baseDefinition()
			definition.Nodes = []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "example/child", MaxDepth: 2}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
			definition.EntryNode = "child"
			definition.Edges = []domain.WorkflowEdge{{From: "child", To: "done"}}
			engine, _, _ := testEngine(t, definition)
			children := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
			engine.Subworkflows = children
			execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "child-terminal-"+string(want))
			if err != nil {
				t.Fatal(err)
			}
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			childID := children.byKey[children.starts[0].IdempotencyKey]
			children.states[childID] = SubworkflowState{ExecutionID: childID, Terminal: true, Status: want, Output: json.RawMessage(`{"reason":"child stopped"}`), TerminalReason: &domain.WorkflowTerminalReason{Code: string(want)}}
			final := mustAdvance(t, engine, execution.ID)
			if final.Status != want || string(final.Output) != `{"reason":"child stopped"}` {
				t.Fatalf("parent terminal=%+v", final)
			}
		})
	}
}

func TestParallelBranchDispatchesDistinctProfilesAndJoins(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
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

func TestParallelFanoutWiderThanConcurrencyDispatchesInBatches(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "one", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "one"}},
		{ID: "two", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "two"}},
		{ID: "three", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "three"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "one"}, {From: "fanout", To: "two"}, {From: "fanout", To: "three"}, {From: "one", To: "joined"}, {From: "two", To: "joined"}, {From: "three", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-batches")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 {
		t.Fatalf("first batch size=%d, want maxConcurrency=2", len(children.requests))
	}
	children.complete(children.requests[0].runID, "done")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 3 {
		t.Fatalf("third member was not dispatched after capacity opened: %d", len(children.requests))
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 3 {
		t.Fatalf("fanout membership=%d, want 3", len(attempts))
	}
}

func TestParallelAnyJoinDoesNotWaitForHungLoser(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "winner", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "winner"}},
		{ID: "loser", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "loser"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "any"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "winner"}, {From: "fanout", To: "loser"}, {From: "winner", To: "joined"}, {From: "loser", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-any")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	children.complete(children.requests[0].runID, "winner")
	mustAdvance(t, engine, execution.ID) // persist winner
	joined := mustAdvance(t, engine, execution.ID)
	if joined.CurrentNodeID != "joined" {
		t.Fatalf("any join waited for loser: %#v", joined)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	shortCircuited := false
	for _, attempt := range attempts {
		shortCircuited = shortCircuited || attempt.ErrorCode == "parallel_join_short_circuit"
	}
	if !shortCircuited {
		t.Fatalf("loser was not durably short-circuited: %+v", attempts)
	}
}

func TestParallelQuorumJoinDoesNotWaitForRemainingMember(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "one", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "one"}},
		{ID: "two", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "two"}},
		{ID: "three", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "three"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "quorum", Quorum: 2}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "one"}, {From: "fanout", To: "two"}, {From: "fanout", To: "three"}, {From: "one", To: "joined"}, {From: "two", To: "joined"}, {From: "three", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-quorum")
	mustAdvance(t, engine, execution.ID) // persist all three members
	mustAdvance(t, engine, execution.ID) // dispatch first member
	mustAdvance(t, engine, execution.ID) // dispatch second member; third remains pending
	children.complete(children.requests[0].runID, "one")
	children.complete(children.requests[1].runID, "two")
	mustAdvance(t, engine, execution.ID)           // persist first completion
	mustAdvance(t, engine, execution.ID)           // use the newly free slot before the second completion is observed
	mustAdvance(t, engine, execution.ID)           // persist second completion
	joined := mustAdvance(t, engine, execution.ID) // short-circuit the remaining member
	if joined.CurrentNodeID != "joined" {
		t.Fatalf("quorum join waited for remaining member: %#v", joined)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	shortCircuited := false
	for _, attempt := range attempts {
		shortCircuited = shortCircuited || attempt.ErrorCode == "parallel_join_short_circuit"
	}
	if !shortCircuited {
		t.Fatalf("remaining member was not durably short-circuited: %+v", attempts)
	}
}

func TestParallelBranchRevisitCreatesFreshVisitAttempts(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "left", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "left"}},
		{ID: "right", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "right"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}},
		{ID: "again", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{
		{From: "fanout", To: "left", MaxTraversals: 2},
		{From: "fanout", To: "right", MaxTraversals: 2},
		{From: "left", To: "joined", MaxTraversals: 2},
		{From: "right", To: "joined", MaxTraversals: 2},
		{From: "joined", To: "again", MaxTraversals: 2},
		{From: "again", To: "fanout", Condition: "iteration < 4", MaxTraversals: 1},
		{From: "again", To: "done"},
	}
	engine, store, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-revisit")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	children.complete(children.requests[0].runID, "left-1")
	children.complete(children.requests[1].runID, "right-1")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	expressions, err := NewExpressionEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	engine = &Engine{Store: store, Catalog: fakeCatalog{revision(definition)}, Children: children, Expressions: expressions}
	mustAdvance(t, engine, execution.ID)

	attempts, err := store.ListAttempts(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 4 {
		t.Fatalf("parallel revisit attempts=%d, want 4: %+v", len(attempts), attempts)
	}
	ordinals := map[string][]int{}
	keys := map[string]bool{}
	for _, attempt := range attempts {
		ordinals[attempt.NodeID] = append(ordinals[attempt.NodeID], attempt.Ordinal)
		if keys[attempt.IdempotencyKey] {
			t.Fatalf("parallel revisit reused idempotency key %q", attempt.IdempotencyKey)
		}
		keys[attempt.IdempotencyKey] = true
	}
	for _, nodeID := range []string{"left", "right"} {
		if got := ordinals[nodeID]; len(got) != 2 || got[0] != 1 || got[1] != 2 {
			t.Fatalf("%s ordinals=%v, want [1 2]", nodeID, got)
		}
	}
}

func TestEnginePreservesAuthoredTerminalOutcome(t *testing.T) { // [REQ:REQ-P2-001]
	for _, tc := range []struct {
		authored string
		want     domain.WorkflowExecutionStatus
	}{
		{authored: "blocked", want: domain.WorkflowExecutionBlocked},
		{authored: "abstained", want: domain.WorkflowExecutionAbstained},
	} {
		t.Run(tc.authored, func(t *testing.T) {
			definition := baseDefinition()
			definition.Nodes = []domain.WorkflowNode{{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: tc.authored}}}
			definition.EntryNode = "done"
			engine, _, _ := testEngine(t, definition)
			execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "terminal-"+tc.authored)
			if err != nil {
				t.Fatal(err)
			}
			final := mustAdvance(t, engine, execution.ID)
			if final.Status != tc.want {
				t.Fatalf("status=%s, want %s", final.Status, tc.want)
			}
		})
	}
}

func TestAgentNodeLimitsReachFreshAndContinuationRequests(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "initial", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "initial", MaxTurns: 7, TimeoutSeconds: 90}},
		{ID: "followup", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "initial", PromptTemplate: "follow up", MaxTurns: 2, TimeoutSeconds: 30}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "initial"
	definition.Edges = []domain.WorkflowEdge{{From: "initial", To: "followup"}, {From: "followup", To: "done"}}
	engine, _, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "node-limits")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	children.complete(children.requests[0].runID, "initial")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 {
		t.Fatalf("requests=%d, want 2", len(children.requests))
	}
	if got := children.requests[0]; got.maxTurns != 7 || got.timeout != 90*time.Second {
		t.Fatalf("fresh limits=(%d,%s), want (7,1m30s)", got.maxTurns, got.timeout)
	}
	if got := children.requests[1]; got.maxTurns != 2 || got.timeout != 30*time.Second {
		t.Fatalf("continuation limits=(%d,%s), want (2,30s)", got.maxTurns, got.timeout)
	}
}

func TestAuthoredPhasedPlanUsesReviewCorrectionAndCorrectedOutput(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-correction")
	if err != nil {
		t.Fatal(err)
	}

	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "original", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, false, "replace the handoff")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[1].source == nil || *children.requests[1].source != children.requests[0].runID {
		t.Fatalf("review rejection did not start named correction: %+v", children.requests)
	}
	if !strings.Contains(children.requests[1].prompt, "replace the handoff") {
		t.Fatalf("correction prompt omitted review note: %q", children.requests[1].prompt)
	}
	completeStructured(children, children.requests[1].runID, map[string]any{"outcome": "complete", "handoff": "corrected"})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded || !strings.Contains(string(final.Output), `"handoff":"corrected"`) || strings.Contains(string(final.Output), `"handoff":"original"`) {
		t.Fatalf("terminal output did not select correction: status=%s reason=%+v output=%s", final.Status, final.TerminalReason, final.Output)
	}
}

func TestAuthoredPhasedPlanEnforcesConsumerSliceLimit(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, store, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(1), "authored-max-slices")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "one", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionBudgetExhausted || final.TerminalReason == nil || final.TerminalReason.BudgetName != "authored_limit" {
		t.Fatalf("slice limit terminal=%+v", final)
	}
	attempts, err := store.ListAttempts(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	slices := 0
	for _, attempt := range attempts {
		if attempt.NodeID == "slice" {
			slices++
		}
	}
	if slices != 1 || len(children.requests) != 1 {
		t.Fatalf("maxSlices=1 created %d slice attempts and %d Run requests", slices, len(children.requests))
	}
}

func TestAuthoredPhasedPlanPreservesBlockedAndAbstainedTerminals(t *testing.T) { // [REQ:REQ-P2-001]
	for _, tc := range []struct {
		name  string
		value map[string]any
		want  domain.WorkflowExecutionStatus
	}{
		{name: "blocked", value: map[string]any{"outcome": "blocked", "handoff": "paused", "blocker": map[string]any{"code": "operator_input", "summary": "operator input required", "retryable": true}}, want: domain.WorkflowExecutionBlocked},
		{name: "abstained", value: map[string]any{"outcome": "abstained", "reason": "insufficient evidence"}, want: domain.WorkflowExecutionAbstained},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := loadAuthoredPhasedPlanDefinition(t)
			engine, _, children := testEngine(t, definition)
			execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-terminal-"+tc.name)
			if err != nil {
				t.Fatal(err)
			}
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			completeStructured(children, children.requests[0].runID, tc.value)
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			final := mustAdvance(t, engine, execution.ID)
			if final.Status != tc.want {
				t.Fatalf("terminal=%+v", final)
			}
		})
	}
}

func TestAuthoredPhasedPlanCreatesFreshRunForNextSlice(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-fresh-slices")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "slice-one-handoff", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 || children.requests[0].runID == children.requests[1].runID {
		t.Fatalf("slice loop did not create distinct Runs: %+v", children.requests)
	}
	if children.requests[1].source != nil || !strings.Contains(children.requests[1].prompt, "slice-one-handoff") {
		t.Fatalf("next slice was not fresh with bounded handoff: %+v", children.requests[1])
	}
}

func TestAuthoredPhasedPlanAutoModeSkipsApprovalWait(t *testing.T) { // [REQ:SWM-AUTONOMY-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2, "auto"), "authored-auto-mode")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "auto-slice", "correctionRequired": false, "approvalRequired": true})
	for i := 0; i < 4; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	completeLatestReview(t, reviews, true, "accepted")
	for i := 0; i < 5; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	if len(children.requests) != 2 {
		t.Fatalf("automatic gate still waited for approval: got %d slice runs", len(children.requests))
	}
}

func TestAuthoredPhasedPlanManualModeRestoresApprovalWait(t *testing.T) { // [REQ:SWM-AUTONOMY-002]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2, "manual"), "authored-manual-mode")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "manual-slice", "correctionRequired": false, "approvalRequired": true})
	for i := 0; i < 4; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	completeLatestReview(t, reviews, true, "accepted")
	for i := 0; i < 5; i++ {
		mustAdvance(t, engine, execution.ID)
	}
	if len(children.requests) != 1 {
		t.Fatalf("manual gate bypassed approval: got %d slice runs", len(children.requests))
	}
	state, err := engine.Store.Get(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentNodeID != "approval" {
		t.Fatalf("manual mode current node = %q, want approval", state.CurrentNodeID)
	}
}

func TestAuthoredPhasedPlanFeedsCorrectedHandoffToNextSlice(t *testing.T) { // [REQ:REQ-P2-001]
	definition := loadAuthoredPhasedPlanDefinition(t)
	engine, _, children := testEngine(t, definition)
	reviews := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = reviews
	execution, err := engine.Start(context.Background(), revision(definition), phasedPlanInput(2), "authored-corrected-handoff")
	if err != nil {
		t.Fatal(err)
	}

	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[0].runID, map[string]any{"outcome": "continue", "handoff": "superseded-handoff", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, false, "replace the handoff")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeStructured(children, children.requests[1].runID, map[string]any{"outcome": "continue", "handoff": "corrected-handoff", "correctionRequired": false, "approvalRequired": false})
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	completeLatestReview(t, reviews, true, "accepted")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 3 {
		t.Fatalf("requests=%d, want original, correction, and next slice", len(children.requests))
	}
	nextSlice := children.requests[2]
	if nextSlice.source != nil || !strings.Contains(nextSlice.prompt, "corrected-handoff") {
		t.Fatalf("next slice did not receive corrected handoff: %+v", nextSlice)
	}
}

func loadAuthoredPhasedPlanDefinition(t *testing.T) domain.WorkflowDefinition {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "swarm-manager", ".vrooli", "agent-manager", "phased-plan-drain.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authored phased plan: %v", err)
	}
	parsed, err := workflowcatalog.Parse(raw, nil)
	if err != nil {
		t.Fatalf("parse authored phased plan: %v", err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("authored phased plan diagnostics: %+v", parsed.Diagnostics)
	}
	definition := parsed.Definition
	// Runtime receives reconciled definitions, where prompt-manager content is
	// materialized into promptTemplate. These authored-workflow tests exercise
	// that runtime contract rather than accidentally testing raw source JSON.
	for i := range definition.Nodes {
		node := &definition.Nodes[i]
		if node.Run != nil && node.Run.PromptRef != nil && node.Run.PromptTemplate == "" {
			node.Run.PromptTemplate = reconciledSkillTemplate(t, node.Run.PromptRef.SkillID)
		}
		if node.Continue != nil && node.Continue.PromptRef != nil && node.Continue.PromptTemplate == "" {
			node.Continue.PromptTemplate = reconciledSkillTemplate(t, node.Continue.PromptRef.SkillID)
		}
	}
	return definition
}

func reconciledSkillTemplate(t *testing.T, skillID string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "prompt-manager", "store", "skills", "packs", "core", skillID, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reconciled prompt skill %q: %v", skillID, err)
	}
	if strings.TrimSpace(string(content)) == "" {
		t.Fatalf("reconciled prompt skill %q was empty", skillID)
	}
	return string(content)
}

func phasedPlanInput(maxSlices int, modes ...string) json.RawMessage {
	mode := "manual"
	if len(modes) > 0 {
		mode = modes[0]
	}
	value, _ := json.Marshal(map[string]any{
		"projectRoot":     "/repo",
		"plan":            map[string]any{"reference": "plan-1", "frontierDigest": "sha256:frontier"},
		"planExecutionId": "plan-execution-1",
		"consumer":        map[string]any{"executionId": "execution-1", "entityKind": "execute", "entityName": "plan", "entityVersion": "sha256:entity"},
		"constraints":     map[string]any{"maxSlices": maxSlices, "writeScope": []string{"scenarios/example/**"}, "sliceApprovalMode": mode},
	})
	return value
}

func completeStructured(children *fakeChildren, runID uuid.UUID, value map[string]any) {
	raw, _ := json.Marshal(value)
	state := children.states[runID]
	state.Terminal = true
	state.Result = &domain.RunResult{FinalOutput: string(raw), Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: raw}}
	children.states[runID] = state
}

func completeLatestReview(t *testing.T, reviews *fakeSubworkflows, accepted bool, note string) {
	t.Helper()
	if len(reviews.starts) == 0 {
		t.Fatal("review workflow was not started")
	}
	start := reviews.starts[len(reviews.starts)-1]
	id := reviews.byKey[start.IdempotencyKey]
	output, _ := json.Marshal(map[string]any{"result": map[string]any{"accepted": accepted, "note": note}})
	reviews.states[id] = SubworkflowState{ExecutionID: id, Terminal: true, Status: domain.WorkflowExecutionSucceeded, Output: output}
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

func TestBindingsSelectBoundedChildWorkflowOutput(t *testing.T) { // [REQ:REQ-P2-001]
	payload := json.RawMessage(`{"childExecutionId":"child-1","status":"succeeded","output":{"review":{"accepted":false,"note":"tighten the handoff"}}}`)
	ctx := BindingContext{Journal: []*domain.WorkflowJournalEntry{{Kind: domain.WorkflowJournalChild, NodeID: "review", Sequence: 1, Payload: payload}}}
	binding := domain.WorkflowInputBinding{Name: "review_note", Source: domain.WorkflowBindingChild, Selector: "node=review;$.output.review.note", Order: "desc", Limit: 1, MaxBytes: 128, RenderAs: "text", MissingPolicy: "error"}
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if values["review_note"] != "tighten the handoff" {
		t.Fatalf("child output binding=%#v", values["review_note"])
	}
}
