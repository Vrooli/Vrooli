package main

import (
	"testing"

	"swarm-manager/internal/proposals"
)

func TestMutationScopeRef_UsesSessionScopeForInitiativeRecreation(t *testing.T) {
	mutation := proposals.Mutation{Op: proposals.OpRecreateInitiative, Target: "stale-initiative"}
	if got := mutationScopeRef(mutation); got != "" {
		t.Fatalf("mutationScopeRef(recreate_initiative) = %q, want session scope", got)
	}
}
