package workflowruntime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

type meteredChildren struct {
	*fakeChildren
	stops     int
	stopError error
}

func (c *meteredChildren) InspectMetered(ctx context.Context, id uuid.UUID) (ChildState, error) {
	return c.Inspect(ctx, id)
}

func (c *meteredChildren) Stop(context.Context, uuid.UUID) error {
	c.stops++
	return c.stopError // Stop acknowledgement is not terminal settlement.
}

func TestMeteredStopSurvivesRestartAndSettlesOvershootOnce(t *testing.T) {
	ctx := context.Background()
	d := budgetAdmissionDefinition()
	d.Budgets.Enforcement = domain.WorkflowBudgetMeteredCancellation
	d.Budgets.MaxTokens = 100
	e, store, fake := testEngine(t, d)
	children := &meteredChildren{fakeChildren: fake, stopError: errors.New("owner temporarily unavailable")}
	e.Children = children
	x, err := e.Start(ctx, revision(d), json.RawMessage(`{}`), "metered")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, e, x.ID)
	mustAdvance(t, e, x.ID)
	id := fake.requests[0].runID
	state := fake.states[id]
	state.Tokens, state.TokensKnown = 100, true
	fake.states[id] = state
	if _, err = e.Advance(ctx, x.ID); err == nil {
		t.Fatal("stop failure was hidden")
	}
	retained, _ := store.Get(ctx, x.ID)
	attempts, _ := store.ListAttempts(ctx, x.ID)
	if retained.Status.Terminal() || retained.BudgetUsage.Tokens != 0 || attempts[0].ErrorCode != meteredStopPrefix+"tokens" {
		t.Fatalf("intent/usage %+v %+v", retained, attempts[0])
	}
	// Recreate the engine. Retry the durable stop intent, even if the next
	// meter reconciles downwards. No dispatch or allowance reset is permitted.
	restarted := *e
	children.stopError = nil
	state.Tokens = 90
	fake.states[id] = state
	mustAdvance(t, &restarted, x.ID)
	if children.stops != 2 || len(fake.requests) != 1 {
		t.Fatal("stop intent was not retried")
	}
	state.Terminal, state.Failed, state.Tokens = true, true, 123
	fake.states[id] = state
	settled := mustAdvance(t, &restarted, x.ID)
	if settled.Status != domain.WorkflowExecutionBudgetExhausted || settled.BudgetUsage.Tokens != 123 {
		t.Fatalf("settlement: %+v", settled)
	}
	again := mustAdvance(t, &restarted, x.ID)
	if again.Version != settled.Version || again.BudgetUsage.Tokens != 123 {
		t.Fatal("reconciliation charged twice")
	}
}

func TestMeteredWallDeadlineWaitsForKnownFinalUsage(t *testing.T) {
	d := budgetAdmissionDefinition()
	d.Budgets.Enforcement = domain.WorkflowBudgetMeteredCancellation
	e, _, fake := testEngine(t, d)
	c := &meteredChildren{fakeChildren: fake}
	e.Children = c
	now := time.Now()
	e.Now = func() time.Time { return now }
	x, err := e.Start(context.Background(), revision(d), json.RawMessage(`{}`), "deadline")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, e, x.ID)
	mustAdvance(t, e, x.ID)
	now = now.Add(601 * time.Second)
	x = mustAdvance(t, e, x.ID)
	if x.Status.Terminal() || c.stops != 1 {
		t.Fatal("deadline discarded running attempt")
	}
	id := fake.requests[0].runID
	state := fake.states[id]
	state.Terminal = true
	fake.states[id] = state
	if _, err = e.Advance(context.Background(), x.ID); err == nil {
		t.Fatal("unknown final usage settled as zero")
	}
	state.TokensKnown, state.Tokens = true, 21
	fake.states[id] = state
	x = mustAdvance(t, e, x.ID)
	if x.Status != domain.WorkflowExecutionBudgetExhausted || x.BudgetUsage.Tokens != 21 {
		t.Fatalf("deadline settlement: %+v", x)
	}
}

func TestMeteredAdmissionRejectsUnqualifiedPaths(t *testing.T) {
	for _, mode := range []string{"hard-ceiling", "invented", domain.WorkflowBudgetMeteredCancellation} {
		d := budgetAdmissionDefinition()
		d.Budgets.Enforcement = mode
		e, _, fake := testEngine(t, d)
		// The legacy launcher has no live-meter contract.
		if _, err := e.Start(context.Background(), revision(d), json.RawMessage(`{}`), mode); err == nil || len(fake.requests) != 0 {
			t.Fatalf("unqualified mode %s admitted", mode)
		}
	}
	for _, kind := range []domain.WorkflowNodeKind{domain.WorkflowNodeContinue, domain.WorkflowNodeChild, domain.WorkflowNodeJoin} {
		d := budgetAdmissionDefinition()
		d.Budgets.Enforcement = domain.WorkflowBudgetMeteredCancellation
		d.Nodes[0].Kind = kind
		if err := domain.ValidateWorkflowBudgetPolicy(d); err == nil {
			t.Fatalf("unqualified %s admitted", kind)
		}
	}
}

func TestMeteredOperatorCancellationWaitsForTerminalAccounting(t *testing.T) {
	d := budgetAdmissionDefinition()
	d.Budgets.Enforcement = domain.WorkflowBudgetMeteredCancellation
	e, _, fake := testEngine(t, d)
	e.Children = &meteredChildren{fakeChildren: fake}
	ctx := context.Background()
	x, err := e.Start(ctx, revision(d), json.RawMessage(`{}`), "operator-cancel")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, e, x.ID)
	mustAdvance(t, e, x.ID)
	if _, _, err = e.Cancel(ctx, x.ID, "cancel", "operator stop", 0); err != nil {
		t.Fatal(err)
	}
	if _, err = e.RecordCleanupDisposition(ctx, x.ID, 1, 0, nil); err == nil {
		t.Fatal("stop acknowledgement became terminal accounting")
	}
	id := fake.requests[0].runID
	state := fake.states[id]
	state.Terminal, state.TokensKnown, state.Tokens = true, true, 123
	fake.states[id] = state
	settled, err := e.RecordCleanupDisposition(ctx, x.ID, 1, 0, nil)
	if err != nil || settled.Status != domain.WorkflowExecutionCancelled || settled.BudgetUsage.Tokens != 123 {
		t.Fatalf("cancel reconciliation: %+v %v", settled, err)
	}
	again, err := e.RecordCleanupDisposition(ctx, x.ID, 1, 0, nil)
	if err != nil || again.Version != settled.Version || again.BudgetUsage.Tokens != 123 {
		t.Fatal("cancel retry charged twice")
	}
}
