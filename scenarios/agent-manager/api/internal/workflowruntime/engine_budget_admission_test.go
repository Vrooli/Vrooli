package workflowruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowcatalog"
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

// The deployed deep run completed its typed draft after 94,163 non-cache and
// 429,568 cache-read tokens. Workflow budgets count their cumulative sum, not
// the 86,988-token context size. Replay against the actual owning declaration.
func TestStagedMessageDeclaredBudgetPreservesSelectiveDraftAndRealExhaustion(t *testing.T) {
	for _, workflow := range []string{"staged-message-deep", "staged-message"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "git-control-tower", ".vrooli", "agent-manager", workflow+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var declared domain.WorkflowDefinition
		if err = json.Unmarshal(raw, &declared); err != nil {
			t.Fatal(err)
		}
		prompt, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "git-control-tower", "skills", "git-control-tower-"+workflow, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		// A test-owned revision must not bypass the authored catalog gates.
		resolved := declared
		resolved.Nodes = append([]domain.WorkflowNode(nil), declared.Nodes...)
		resolvedRun := *resolved.Nodes[0].Run
		resolvedRun.PromptRef, resolvedRun.PromptTemplate = nil, string(prompt)
		resolved.Nodes[0].Run = &resolvedRun
		validated, err := workflowcatalog.Validate(resolved, nil)
		if err != nil || domain.HasBlockingDiagnostic(validated.Diagnostics) {
			t.Fatalf("actual drafting declaration cannot reconcile: %v %+v", err, validated.Diagnostics)
		}
		measured := 94163 + 429568
		boundedFollowup := 1_500_000
		if workflow == "staged-message" {
			measured = 18243 + 8320
			boundedFollowup = 180000
		}
		for _, scenario := range []string{"observed completed usage", "bounded selective followup", "genuine exhaustion"} {
			t.Run(workflow+"/"+scenario, func(t *testing.T) {
				// Reuse the declaration's actual budgets and output bindings. Prompt
				// resolution is orthogonal to terminal usage, so no provider is invoked.
				d := declared
				d.InputSchema = json.RawMessage(`{"type":"object"}`)
				d.Nodes = append([]domain.WorkflowNode(nil), declared.Nodes...)
				run := *d.Nodes[0].Run
				run.PromptRef = nil
				run.PromptTemplate = "Replay completed drafting usage"
				run.Bindings = nil
				run.ProjectRootTemplate = ""
				d.Nodes[0].Run = &run
				e, _, children := testEngine(t, d)
				e.Children = &meteredChildren{fakeChildren: children}
				x, err := e.Start(t.Context(), revision(d), json.RawMessage(`{}`), scenario)
				if err != nil {
					t.Fatal(err)
				}
				for step := 0; step < 6 && len(children.requests) == 0 && !x.Status.Terminal(); step++ {
					x = mustAdvance(t, e, x.ID)
				}
				if len(children.requests) != 1 {
					t.Fatalf("expected one bounded child dispatch, got %d", len(children.requests))
				}
				spent := measured
				if scenario == "bounded selective followup" {
					spent = boundedFollowup
				}
				if scenario == "genuine exhaustion" {
					spent = declared.Budgets.MaxTokens + 1
				}
				result := json.RawMessage(`{"snapshot_digest":"fixture","title":"Summarize the complete staged selection","body":"Describe the affected themes from inventory and selected details.","trailers":[],"source_ids":[],"unknowns":[],"covered_file_ids":[1,285],"message_scope":"whole_selection","message_scope_explanation":"The complete inventory and selective patch evidence support every identified theme."}`)
				if workflow == "staged-message" {
					result = json.RawMessage(`{"snapshot_digest":"fixture","title":"Summarize staged changes","body":"Describe representative details.","trailers":[],"source_ids":[],"unknowns":[],"covered_file_ids":[1]}`)
				}
				id := children.requests[0].runID
				state := children.states[id]
				state.Terminal, state.TokensKnown, state.ChargeMeasured = true, true, true
				state.Tokens = spent
				state.Result = &domain.RunResult{FinalOutput: string(result), Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: result}}
				children.states[id] = state
				for step := 0; step < 6 && !x.Status.Terminal(); step++ {
					x = mustAdvance(t, e, x.ID)
				}
				want := domain.WorkflowExecutionSucceeded
				if scenario == "genuine exhaustion" {
					want = domain.WorkflowExecutionBudgetExhausted
				}
				if x.Status != want || x.BudgetUsage.Tokens != spent || len(children.requests) != 1 {
					t.Fatalf("budget replay status=%s usage=%d calls=%d want=%s", x.Status, x.BudgetUsage.Tokens, len(children.requests), want)
				}
				if want == domain.WorkflowExecutionSucceeded {
					var output map[string]json.RawMessage
					if err := json.Unmarshal(x.Output, &output); err != nil {
						t.Fatal(err)
					}
					var got, expected any
					if json.Unmarshal(output["result"], &got) != nil || json.Unmarshal(result, &expected) != nil || !reflect.DeepEqual(got, expected) {
						t.Fatalf("completed typed output lost: %s", x.Output)
					}
				} else if x.TerminalReason == nil || x.TerminalReason.BudgetName != "tokens" {
					t.Fatalf("exhaustion reason lost: %+v", x.TerminalReason)
				}
			})
		}
	}
}
