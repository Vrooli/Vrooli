package workflowruntime

import (
	"encoding/json"
	"strconv"
	"testing"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestOrdinaryCancellationWaitsForTerminalAccountingAfterRestart(t *testing.T) {
	for _, tokens := range []int{0, 23} {
		t.Run(strconv.Itoa(tokens), func(t *testing.T) {
			d := budgetAdmissionDefinition()
			e, store, children := testEngine(t, d)
			x, err := e.Start(t.Context(), revision(d), json.RawMessage(`{}`), "ordinary-cancel")
			if err != nil {
				t.Fatal(err)
			}
			mustAdvance(t, e, x.ID)
			mustAdvance(t, e, x.ID)
			id := children.requests[0].runID
			if _, _, err := e.Cancel(t.Context(), x.ID, "cancel-once", "operator", 0); err != nil {
				t.Fatal(err)
			}
			state := children.states[id]
			state.Tokens, state.ChargeMeasured = tokens, true
			children.states[id] = state
			for _, terminal := range []bool{false, true} {
				state.Terminal = terminal
				children.states[id] = state
				if _, err := e.RecordCleanupDisposition(t.Context(), x.ID, 1, 0, nil); err == nil {
					t.Fatal("stop acknowledgement or unknown final usage settled cancellation")
				}
				stored, _ := store.Get(t.Context(), x.ID)
				if stored.Status != domain.WorkflowExecutionCancelling || stored.EndedAt != nil || stored.BudgetUsage.Tokens != 0 || stored.BudgetUsage.AccountingComplete {
					t.Fatalf("premature settlement: %+v", stored)
				}
			}
			// Fresh engine, same durable store and child identity; no dispatch.
			restarted := &Engine{Store: store, Catalog: e.Catalog, Children: children, Expressions: e.Expressions}
			state.TokensKnown, state.ChargeMeasured = true, false
			children.states[id] = state
			if _, err := restarted.RecordCleanupDisposition(t.Context(), x.ID, 1, 0, nil); err == nil {
				t.Fatal("unknown charge was replaced by measured zero")
			}
			state.ChargeMeasured, state.Turns = true, 2
			children.states[id] = state
			settled, err := restarted.RecordCleanupDisposition(t.Context(), x.ID, 1, 0, nil)
			if err != nil || settled.Status != domain.WorkflowExecutionCancelled || !settled.BudgetUsage.AccountingComplete || !settled.BudgetUsage.ChargeMeasured || settled.BudgetUsage.Tokens != tokens || settled.BudgetUsage.Turns != 2 || settled.BudgetUsage.Children != 1 {
				t.Fatalf("late receipt not settled: %+v %v", settled, err)
			}
			repeated, err := restarted.RecordCleanupDisposition(t.Context(), x.ID, 1, 0, nil)
			if err != nil || repeated.Version != settled.Version || repeated.BudgetUsage != settled.BudgetUsage || len(children.requests) != 1 {
				t.Fatalf("repeat duplicated usage/dispatch: %+v %v", repeated, err)
			}
		})
	}
}

func TestOrdinaryCancellationRebuildsCompletedAndNestedAttemptAccounting(t *testing.T) {
	for _, nested := range []bool{false, true} {
		d := budgetAdmissionDefinition()
		d.Nodes = append(d.Nodes, domain.WorkflowNode{ID: "next", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "run", PromptTemplate: "continue"}})
		if nested {
			d.Nodes[2] = domain.WorkflowNode{ID: "next", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "owner/review", MaxDepth: 1}}
		}
		d.Edges = []domain.WorkflowEdge{{From: "run", To: "next"}, {From: "next", To: "done"}}
		e, _, children := testEngine(t, d)
		sub := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
		e.Subworkflows = sub
		x, err := e.Start(t.Context(), revision(d), json.RawMessage(`{}`), "cancel-completed-and-active")
		if err != nil {
			t.Fatal(err)
		}
		mustAdvance(t, e, x.ID)
		mustAdvance(t, e, x.ID)
		firstID := children.requests[0].runID
		first := children.states[firstID]
		first.Terminal, first.TokensKnown, first.ChargeMeasured, first.Tokens, first.Turns = true, true, true, 10, 1
		children.states[firstID] = first
		mustAdvance(t, e, x.ID)
		mustAdvance(t, e, x.ID)
		mustAdvance(t, e, x.ID)
		if _, _, err := e.Cancel(t.Context(), x.ID, "cancel", "operator", 0); err != nil {
			t.Fatal(err)
		}
		if _, err := e.RecordCleanupDisposition(t.Context(), x.ID, 1, 1, nil); err == nil {
			t.Fatal("active child settled on acknowledgement")
		}
		wantChildren, wantAttempts := 2, 2
		if nested {
			for id, state := range sub.states {
				state.Terminal = true
				state.BudgetUsage = domain.WorkflowBudgetUsage{AccountingComplete: true, ChargeMeasured: true, Tokens: 20, Turns: 2, Children: 1, NodeAttempts: 1, Retries: 1}
				sub.states[id] = state
			}
			wantChildren, wantAttempts = 3, 3
		} else {
			id := children.requests[1].runID
			state := children.states[id]
			state.Terminal, state.TokensKnown, state.ChargeMeasured, state.Tokens, state.Turns = true, true, true, 20, 2
			children.states[id] = state
		}
		settled, err := e.RecordCleanupDisposition(t.Context(), x.ID, 1, 1, nil)
		if err != nil || settled.BudgetUsage.Tokens != 30 || settled.BudgetUsage.Turns != 3 || settled.BudgetUsage.Children != wantChildren || settled.BudgetUsage.NodeAttempts != wantAttempts || !settled.BudgetUsage.AccountingComplete {
			t.Fatalf("nested=%t: rebuilt usage=%+v err=%v", nested, settled, err)
		}
		again, err := e.RecordCleanupDisposition(t.Context(), x.ID, 1, 1, nil)
		if err != nil || again.BudgetUsage != settled.BudgetUsage || again.Version != settled.Version {
			t.Fatalf("nested=%t: charged completed attempts again: %+v %v", nested, again, err)
		}
	}
}
