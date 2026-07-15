package opsrunner

import (
	"context"
	"errors"
	"testing"

	"swarm-manager/internal/agentops"
)

func reviewPolicy() agentops.TransitionPolicy {
	return agentops.TransitionPolicy{
		Kind: "agentops-transition-policy", ID: "p", Version: "1.0.0", DomainKind: "backlog-item",
		Transitions: []agentops.PolicyTransition{
			{FromState: "running", OnOutcome: "accepted", Action: agentops.ActionOpenReview, ToState: "awaiting-decision"},
			{FromState: "awaiting-decision", OnOutcome: "accepted", Action: agentops.ActionCompleteItem, ToState: "terminal-complete"},
		},
	}
}

func TestEvaluateTransitionSelectsRule(t *testing.T) {
	sel, err := EvaluateTransition(reviewPolicy(), agentops.WorkflowRunning, agentops.OpWorkshopRound, "accepted")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Action != agentops.ActionOpenReview || sel.ToState != agentops.WorkflowAwaitingDecision {
		t.Fatalf("selected %+v", sel)
	}
}

func TestEvaluateTransitionNoMatch(t *testing.T) {
	if _, err := EvaluateTransition(reviewPolicy(), agentops.WorkflowRunning, agentops.OpWorkshopRound, "blocked"); !errors.Is(err, ErrNoTransition) {
		t.Fatalf("unmatched outcome must be ErrNoTransition, got %v", err)
	}
}

func TestEvaluateTransitionAmbiguityFailsClosed(t *testing.T) {
	p := reviewPolicy()
	p.Transitions = append(p.Transitions, agentops.PolicyTransition{
		FromState: "running", OnOutcome: "accepted", Action: agentops.ActionFailItem, ToState: "terminal-failed",
	})
	if _, err := EvaluateTransition(p, agentops.WorkflowRunning, agentops.OpWorkshopRound, "accepted"); err == nil {
		t.Fatalf("two rules for one (state,outcome) must fail closed")
	}
}

// TestEvaluateTransitionOperationSpecificBeatsGeneric proves an operation-pinned
// rule overrides the generic rule for the same (state, outcome) — so one workflow
// state routes different completing operations to different actions — while a
// non-matching operation still falls through to the generic rule.
func TestEvaluateTransitionOperationSpecificBeatsGeneric(t *testing.T) {
	p := agentops.TransitionPolicy{
		Kind: "agentops-transition-policy", ID: "p", Version: "1.0.0", DomainKind: "backlog-item",
		Transitions: []agentops.PolicyTransition{
			{FromState: "running", OnOutcome: "completed", Action: agentops.ActionOpenReview, ToState: "awaiting-decision"},
			{FromState: "running", OnOutcome: "completed", Operation: agentops.OpWorkshopFinalize, Action: agentops.ActionBindPlan, ToState: "running"},
		},
	}
	// The operation-specific rule wins for its operation.
	sel, err := EvaluateTransition(p, agentops.WorkflowRunning, agentops.OpWorkshopFinalize, "completed")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Action != agentops.ActionBindPlan {
		t.Fatalf("finalize completed must bind-plan, got %q", sel.Action)
	}
	// Any other operation falls through to the generic rule.
	sel, err = EvaluateTransition(p, agentops.WorkflowRunning, agentops.OpWorkshopRound, "completed")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Action != agentops.ActionOpenReview {
		t.Fatalf("round completed must open-review, got %q", sel.Action)
	}
}

// TestDispatchCommitsAndDeduplicates proves a dispatch commits the state change,
// and a replay with the same idempotency key is a no-op (no double effect).
func TestDispatchCommitsAndDeduplicates(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	target := TargetRef{Kind: agentops.TargetBacklogItem, ID: "fix/x"}
	var handlerCalls int
	reg := NewActionRegistry()
	reg.Register(agentops.ActionOpenReview, func(context.Context, ActionContext) error {
		handlerCalls++
		return nil
	})
	d := NewDispatcher(reg, repo)

	w, _ := repo.CreateOrLoad(target.Kind, target.ID)
	running := cloneWorkflow(w)
	running.State = agentops.WorkflowRunning
	running.Version = 1
	if err := repo.Commit(0, running); err != nil {
		t.Fatal(err)
	}

	sel := SelectedTransition{Action: agentops.ActionOpenReview, ToState: agentops.WorkflowAwaitingDecision}
	res, err := d.Dispatch(context.Background(), target, running, sel, "accepted", agentops.OpReviewRound, "key-1", DispatchDelivery{})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if !res.Applied || res.State != agentops.WorkflowAwaitingDecision {
		t.Fatalf("dispatch result %+v", res)
	}
	// Replay with the same key against the now-updated workflow: no-op.
	updated, _, _ := repo.Load(target.Kind, target.ID)
	replay, err := d.Dispatch(context.Background(), target, updated, sel, "accepted", agentops.OpReviewRound, "key-1", DispatchDelivery{})
	if err != nil {
		t.Fatalf("replay dispatch: %v", err)
	}
	if !replay.Replayed {
		t.Fatalf("replay must be deduplicated")
	}
	if handlerCalls != 1 {
		t.Fatalf("handler ran %d times, want exactly once", handlerCalls)
	}
}

// TestDispatchRejectsUnregisteredAction proves data can never invoke code outside
// the closed registry.
func TestDispatchRejectsUnregisteredAction(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	d := NewDispatcher(NewActionRegistry(), repo)
	w := baseWorkflow(agentops.TargetBacklogItem, "fix/x")
	sel := SelectedTransition{Action: agentops.ActionName("run-shell"), ToState: agentops.WorkflowRunning}
	if _, err := d.Dispatch(context.Background(), TargetRef{Kind: agentops.TargetBacklogItem, ID: "fix/x"}, w, sel, "x", agentops.OpReviewRound, "k", DispatchDelivery{}); err == nil {
		t.Fatalf("unregistered action must be rejected")
	}
}

// TestDispatchHandlerFailureLeavesNoStateChange proves a failed handler neither
// advances the workflow nor consumes the idempotency key (a clean retry).
func TestDispatchHandlerFailureLeavesNoStateChange(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	target := TargetRef{Kind: agentops.TargetBacklogItem, ID: "fix/x"}
	reg := NewActionRegistry()
	reg.Register(agentops.ActionOpenReview, func(context.Context, ActionContext) error {
		return errors.New("boom")
	})
	d := NewDispatcher(reg, repo)
	w, _ := repo.CreateOrLoad(target.Kind, target.ID)
	running := cloneWorkflow(w)
	running.State = agentops.WorkflowRunning
	running.Version = 1
	if err := repo.Commit(0, running); err != nil {
		t.Fatal(err)
	}
	sel := SelectedTransition{Action: agentops.ActionOpenReview, ToState: agentops.WorkflowAwaitingDecision}
	if _, err := d.Dispatch(context.Background(), target, running, sel, "accepted", agentops.OpReviewRound, "k", DispatchDelivery{}); err == nil {
		t.Fatalf("handler failure must propagate")
	}
	got, _, _ := repo.Load(target.Kind, target.ID)
	if got.State != agentops.WorkflowRunning {
		t.Fatalf("state advanced despite handler failure: %s", got.State)
	}
	if HasIdempotencyKey(got, "k") {
		t.Fatalf("idempotency key consumed despite handler failure")
	}
}

func TestActionRegistryRejectsUnregisteredRegistration(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("registering an unregistered action must panic")
		}
	}()
	NewActionRegistry().Register(agentops.ActionName("bogus"), func(context.Context, ActionContext) error { return nil })
}
