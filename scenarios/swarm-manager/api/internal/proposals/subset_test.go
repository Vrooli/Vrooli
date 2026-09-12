package proposals

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
)

// Accepting a mutation whose premise was rejected must fail before anything
// is written, not partway through the batch. The full list validates because
// the rejected add_item supplies the edge's endpoint; the subset does not.
func TestApplyRejectsAnIncoherentAcceptedSubset(t *testing.T) {
	env := newApplyEnv(t)
	proposal := Proposal{Form: FormMutationList, Mutations: []Mutation{
		{ID: "m1", Op: OpAddItem, Item: &ItemSpec{Kind: "execute", Name: "new-work", Title: "New work"}},
		{ID: "m2", Op: OpAddEdge, From: "execute/foo", To: "execute/new-work"},
	}}
	if err := Validate(proposal, env.currentState()); err != nil {
		t.Fatalf("the full list should validate: %v", err)
	}
	// Accept only the edge, rejecting the item that gives it an endpoint.
	_, err := env.applier.Apply(context.Background(), proposal, env.currentState(), []string{"m2"}, Source{MilestoneName: "work/ui-rewrite"})
	if err == nil {
		t.Fatal("Apply accepted a subset whose edge endpoint was never created")
	}
	if !strings.Contains(err.Error(), "accepted subset") {
		t.Fatalf("error should name the subset as the problem, got: %v", err)
	}
	if _, loadErr := env.backlog.LoadItem(backlog.BacklogKind("execute"), "new-work"); loadErr == nil {
		t.Fatal("rejected item was created anyway")
	}
}

func TestApplyAcceptsACoherentSubset(t *testing.T) {
	env := newApplyEnv(t)
	proposal := Proposal{Form: FormMutationList, Mutations: []Mutation{
		{ID: "m1", Op: OpChangePriority, Target: "execute/foo", Priority: intPtr(3)},
		{ID: "m2", Op: OpChangePriority, Target: "execute/bar", Priority: intPtr(4)},
	}}
	result, err := env.applier.Apply(context.Background(), proposal, env.currentState(), []string{"m1"}, Source{MilestoneName: "work/ui-rewrite"})
	if err != nil {
		t.Fatalf("a coherent subset should apply: %v", err)
	}
	if result.Applied != 1 || result.Skipped != 1 {
		t.Fatalf("expected 1 applied and 1 skipped, got applied=%d skipped=%d", result.Applied, result.Skipped)
	}
}
