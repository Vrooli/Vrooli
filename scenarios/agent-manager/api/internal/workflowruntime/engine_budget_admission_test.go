package workflowruntime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

// Hypothesis: known usage bypasses admission on the schema-repair branch.
// Supplying usage directly rules out the competing missing-meter hypothesis.
func TestSchemaRepairCannotSpendAnExhaustedBudget(t *testing.T) {
	for _, spent := range []int{100, 101} {
		d := budgetAdmissionDefinition()
		d.Budgets.MaxTokens = 100
		d.Nodes[0].Run.ResultSpec = &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"type":"object"}`)}
		e, store, children := testEngine(t, d)
		x, err := e.Start(context.Background(), revision(d), json.RawMessage(`{}`), "repair-budget")
		if err != nil {
			t.Fatal(err)
		}
		mustAdvance(t, e, x.ID)
		mustAdvance(t, e, x.ID)
		id := children.requests[0].runID
		state := children.states[id]
		state.Terminal, state.Tokens = true, spent
		state.Result = &domain.RunResult{FinalOutput: "invalid", Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid}}
		children.states[id] = state
		got := mustAdvance(t, e, x.ID)
		attempts, _ := store.ListAttempts(context.Background(), x.ID)
		if got.Status != domain.WorkflowExecutionBudgetExhausted || got.BudgetUsage.Tokens != spent || len(attempts) != 1 {
			t.Fatalf("spent=%d: status=%s usage=%+v attempts=%d; want retained usage and exhaustion without repair", spent, got.Status, got.BudgetUsage, len(attempts))
		}
	}
}

func TestNextAgentDispatchHonorsRemainingBudget(t *testing.T) {
	d := budgetAdmissionDefinition()
	d.Nodes[0].Run.MaxTurns = 50
	d.Nodes[0].Run.TimeoutSeconds = 1000
	e, _, children := testEngine(t, d)
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	x, err := e.Start(context.Background(), revision(d), json.RawMessage(`{}`), "bounded-child")
	if err != nil {
		t.Fatal(err)
	}
	// Time between durable intent and dispatch is not a new allowance.
	mustAdvance(t, e, x.ID)
	now = now.Add(590 * time.Second)
	mustAdvance(t, e, x.ID)
	call := children.requests[0]
	if call.maxTurns != 10 || call.timeout != 10*time.Second {
		t.Fatalf("dispatch limits turns=%d timeout=%s; want remaining 10 turns, 10 seconds", call.maxTurns, call.timeout)
	}
}

func TestExactBudgetCanFinishButCannotStartAnotherAgent(t *testing.T) {
	for _, another := range []bool{false, true} {
		d := budgetAdmissionDefinition()
		if another {
			d.Nodes = append(d.Nodes, domain.WorkflowNode{ID: "second", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "next"}})
			d.Edges = []domain.WorkflowEdge{{From: "run", To: "second"}, {From: "second", To: "done"}}
		}
		e, _, children := testEngine(t, d)
		x, err := e.Start(context.Background(), revision(d), json.RawMessage(`{}`), "exact-budget")
		if err != nil {
			t.Fatal(err)
		}
		mustAdvance(t, e, x.ID)
		mustAdvance(t, e, x.ID)
		id := children.requests[0].runID
		state := children.states[id]
		state.Terminal, state.Tokens = true, d.Budgets.MaxTokens
		children.states[id] = state
		mustAdvance(t, e, x.ID)
		got := mustAdvance(t, e, x.ID)
		want := domain.WorkflowExecutionSucceeded
		if another {
			want = domain.WorkflowExecutionBudgetExhausted
		}
		if got.Status != want || len(children.requests) != 1 {
			t.Fatalf("another=%t: status=%s children=%d; want %s and no further dispatch", another, got.Status, len(children.requests), want)
		}
	}
}

func budgetAdmissionDefinition() domain.WorkflowDefinition {
	d := baseDefinition()
	d.EntryNode = "run"
	d.Nodes = []domain.WorkflowNode{
		{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	d.Edges = []domain.WorkflowEdge{{From: "run", To: "done"}}
	return d
}

func TestAggregateAccountingRequiresEveryChildReceipt(t *testing.T) {
	for _, secondKnown := range []bool{true, false} {
		d := budgetAdmissionDefinition()
		d.Nodes = append(d.Nodes, domain.WorkflowNode{ID: "second", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "second"}})
		d.Edges = []domain.WorkflowEdge{{From: "run", To: "second"}, {From: "second", To: "done"}}
		e, _, children := testEngine(t, d)
		x, err := e.Start(t.Context(), revision(d), json.RawMessage(`{}`), "complete-accounting")
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ {
			mustAdvance(t, e, x.ID)
			mustAdvance(t, e, x.ID)
			id := children.requests[i].runID
			state := children.states[id]
			state.Terminal, state.TokensKnown, state.ChargeMeasured = true, i == 0 || secondKnown, true
			state.Tokens, state.ChargeMicroUSD = 5, 1
			children.states[id] = state
			mustAdvance(t, e, x.ID)
		}
		finished := mustAdvance(t, e, x.ID)
		if finished.Status != domain.WorkflowExecutionSucceeded || finished.BudgetUsage.Tokens != 10 || finished.BudgetUsage.AccountingComplete != secondKnown {
			t.Fatalf("one child concealed unknown accounting: %+v secondKnown=%t", finished, secondKnown)
		}
	}
}
