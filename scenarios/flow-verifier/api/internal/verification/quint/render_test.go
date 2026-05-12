package quint_test

import (
	"strings"
	"testing"

	"flow-verifier/internal/flows/contract"
	"flow-verifier/internal/flows/model"
	"flow-verifier/internal/testkit"
	"flow-verifier/internal/verification/quint"
)

func TestRenderKeepsTransitionTableOutOfVerifiedInvariants(t *testing.T) {
	c := testkit.ValidRawContract()
	c.States = []contract.State{{ID: "idle", Quint: "Idle", Initial: true}}
	c.Events = []contract.Event{{ID: "tick", Quint: "Tick"}}
	c.TransitionDefaults = contract.TransitionDefaults{Invalid: &contract.DefaultTransition{To: model.SelfTarget, WantError: true}}
	c.Transitions = []contract.Transition{{From: contract.StringList{"idle"}, Event: contract.StringList{"tick"}, To: model.SelfTarget, WantError: boolPtr(true)}}
	c.Invariants = []contract.Invariant{{ID: "type_ok", Quint: "TypeOK", Description: "Type OK."}}
	c.Traces = []contract.Trace{{Name: "idle", Initial: "idle", Steps: nil}}
	c.Model.MaxSteps = 1
	c.Model.TraceCount = 1
	c.Model.Verify = contract.Verify{Invariants: []string{"TypeOK"}}
	flow := testkit.MustCompile(t, c)
	rendered := quint.Render(flow)
	if strings.Contains(rendered, "AllDeclaredTransitionsCovered") {
		t.Fatalf("rendered model still contains fake invariant:\n%s", rendered)
	}
	if !strings.Contains(rendered, "run transitionTable") {
		t.Fatalf("rendered model does not contain transitionTable run:\n%s", rendered)
	}
}

func boolPtr(value bool) *bool {
	return &value
}
