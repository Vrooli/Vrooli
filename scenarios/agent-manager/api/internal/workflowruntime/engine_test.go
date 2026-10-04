package workflowruntime

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"
)

type fixedPromptResolver struct{ resolution PromptResolution }

func TestTypedReviewCannotCompleteWithoutValidVerdict(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		for _, kind := range []string{"missing", "untyped", "invalid", "false-success", "goal-complete"} {
			t.Run(fmt.Sprintf("parallel=%t/%s", parallel, kind), func(t *testing.T) {
				definition := baseDefinition()
				zero := 0
				spec := &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, SchemaRepairAttempts: &zero, Schema: json.RawMessage(`{"type":"object","required":["accepted"],"properties":{"accepted":{"type":"boolean"}}}`)}
				definition.EntryNode = "review"
				definition.Nodes = []domain.WorkflowNode{{ID: "review", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "review", ResultSpec: spec}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
				definition.Edges = []domain.WorkflowEdge{{From: "review", To: "done"}}
				if parallel {
					definition.EntryNode = "fork"
					definition.Nodes = append(definition.Nodes, domain.WorkflowNode{ID: "fork", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}}, domain.WorkflowNode{ID: "other", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "other"}}, domain.WorkflowNode{ID: "join", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}})
					definition.Edges = []domain.WorkflowEdge{{From: "fork", To: "review"}, {From: "fork", To: "other"}, {From: "review", To: "join"}, {From: "other", To: "join"}, {From: "join", To: "done"}}
				}
				engine, store, children := testEngine(t, definition)
				x, err := engine.Start(t.Context(), revision(definition), json.RawMessage(`{}`), "typed-review")
				if err != nil {
					t.Fatal(err)
				}
				for step := 0; step < 10 && !x.Status.Terminal(); step++ {
					x = mustAdvance(t, engine, x.ID)
					for id, state := range children.states {
						state.Terminal = true
						switch kind {
						case "untyped":
							state.Result = &domain.RunResult{FinalOutput: "accepted"}
						case "invalid":
							state.Result = &domain.RunResult{Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid}}
						case "false-success":
							state.Result = &domain.RunResult{Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: json.RawMessage(`{"wrong":true}`)}}
						case "goal-complete":
							state.GoalStatus = runner.GoalStatusComplete
						}
						children.states[id] = state
					}
				}
				if x.Status != domain.WorkflowExecutionFailed {
					t.Fatalf("missing/invalid review passed: %+v", x)
				}
				attempts, _ := store.ListAttempts(t.Context(), x.ID)
				for _, attempt := range attempts {
					if attempt.NodeID == "review" && (attempt.ErrorCode != "structured_result_invalid" || attempt.ValidationError == "") {
						t.Fatalf("missing retained refusal: %+v", attempt)
					}
				}
			})
		}
	}
}

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

func TestEngine_InheritsOwnerEffectGrantIntoChildRequest(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "work", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.fast", PromptTemplate: "do work"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "work"
	definition.Edges = []domain.WorkflowEdge{{From: "work", To: "done"}}
	engine, _, children := testEngine(t, definition)
	grant := &domain.WorkflowEngagementGrant{MaxTokens: 200, MaxWallTimeSeconds: 30, AllowedEffects: []string{"filesystem.write[paths=scenarios/example/**]"}}
	execution, err := engine.StartWithGrant(context.Background(), revision(definition), json.RawMessage(`{}`), "effect-inheritance", grant)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Advance(context.Background(), execution.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Advance(context.Background(), execution.ID); err != nil {
		t.Fatal(err)
	}
	if len(children.requests) != 1 || len(children.requests[0].effects) != 1 || children.requests[0].effects[0] != grant.AllowedEffects[0] {
		t.Fatalf("child effect grant=%+v, want=%v", children.requests, grant.AllowedEffects)
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

func TestRecoveryRetainsRevisionSandboxAuthority(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "work", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{
			RoleRef: "code.default", PromptTemplate: "work",
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected,
				NetworkMode: domain.NetworkAccessNone, ManualReview: true,
				WritePolicy: &domain.WorkspaceWritePolicy{Paths: []string{"src"}}},
		}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "work"
	definition.Edges = []domain.WorkflowEdge{{From: "work", To: "done"}}
	engine, store, children := testEngine(t, definition)
	x, err := engine.Start(t.Context(), revision(definition), json.RawMessage(`{}`), "pinned-authority")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, x.ID) // Persist intent before losing the engine instance.
	// Reload the revision as durable JSON; do not rely on an in-memory pointer.
	encoded, err := json.Marshal(revision(definition))
	if err != nil {
		t.Fatal(err)
	}
	var retained domain.WorkflowRevision
	if err := json.Unmarshal(encoded, &retained); err != nil {
		t.Fatal(err)
	}
	expressions, _ := NewExpressionEvaluator()
	restarted := &Engine{Store: store, Catalog: fakeCatalog{&retained}, Children: children, Expressions: expressions}
	mustAdvance(t, restarted, x.ID)
	mustAdvance(t, restarted, x.ID)
	if len(children.requests) != 1 {
		t.Fatalf("recovery dispatched %d children, want one", len(children.requests))
	}
	policy := children.requests[0].sandboxConfig
	if policy == nil || policy.Mode != domain.SandboxModeProtected || policy.NetworkMode != domain.NetworkAccessNone ||
		!policy.ManualReview || policy.WritePolicy == nil || len(policy.WritePolicy.Paths) != 1 || policy.WritePolicy.Paths[0] != "src" {
		t.Fatalf("pinned authority lost across recovery: %+v", policy)
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
