package contract

import (
	"strings"
	"testing"
)

func TestValidateAndExpandValidCompactContract(t *testing.T) {
	c := validContract()
	if err := ValidateAndExpand(&c); err != nil {
		t.Fatalf("ValidateAndExpand() error = %v", err)
	}
	if got, want := len(c.ExpandedTransitions), len(c.States)*len(c.Events); got != want {
		t.Fatalf("expanded transitions = %d, want %d", got, want)
	}
	assertTransition(t, c, "idle", "start", "busy", false)
	assertTransition(t, c, "done", "finish", "done", true)
}

func TestValidateAndExpandRejectsDuplicateStateIDs(t *testing.T) {
	c := validContract()
	c.States[1].ID = "idle"
	requireErrorContains(t, ValidateAndExpand(&c), "duplicate states id idle")
}

func TestValidateAndExpandRejectsDuplicateEventIDs(t *testing.T) {
	c := validContract()
	c.Events[1].ID = "start"
	requireErrorContains(t, ValidateAndExpand(&c), "duplicate events id start")
}

func TestValidateAndExpandRejectsDuplicateQuintTags(t *testing.T) {
	c := validContract()
	c.Events[0].Quint = "Idle"
	requireErrorContains(t, ValidateAndExpand(&c), "duplicate Quint tag Idle")
}

func TestValidateAndExpandRejectsMissingInitialState(t *testing.T) {
	c := validContract()
	c.States[0].Initial = false
	requireErrorContains(t, ValidateAndExpand(&c), "states must declare exactly one initial state, got 0")
}

func TestValidateAndExpandRejectsMultipleInitialStates(t *testing.T) {
	c := validContract()
	c.States[1].Initial = true
	requireErrorContains(t, ValidateAndExpand(&c), "states must declare exactly one initial state, got 2")
}

func TestValidateAndExpandRejectsUnknownTransitionStateAndEvent(t *testing.T) {
	c := validContract()
	c.Transitions = append(c.Transitions, Transition{From: StringList{"ghost"}, Event: StringList{"missing"}, To: "done"})
	requireErrorContains(t, ValidateAndExpand(&c), "references unknown")
}

func TestValidateAndExpandRejectsDuplicateTransitionPair(t *testing.T) {
	c := validContract()
	c.Transitions = append(c.Transitions, Transition{From: StringList{"idle"}, Event: StringList{"start"}, To: "busy"})
	requireErrorContains(t, ValidateAndExpand(&c), "duplicate transition pair idle/start")
}

func TestValidateAndExpandRejectsUnknownInvariantReference(t *testing.T) {
	c := validContract()
	c.Model.Verify.Invariants = append(c.Model.Verify.Invariants, "GhostInvariant")
	requireErrorContains(t, ValidateAndExpand(&c), "references unknown invariant GhostInvariant")
}

func TestValidateAndExpandRejectsInvalidQuintName(t *testing.T) {
	c := validContract()
	c.States[0].Quint = "not-valid"
	requireErrorContains(t, ValidateAndExpand(&c), "invalid Quint identifier not-valid")
}

func validContract() Contract {
	return Contract{
		SchemaVersion: 2,
		FlowID:        "example.flow",
		Domain:        "example",
		Description:   "Example flow.",
		Model: Model{
			Module:     "ExampleFlow",
			Seed:       "1",
			MaxSteps:   4,
			TraceCount: 2,
			Verify:     Verify{Invariants: []string{"TypeOK", "TerminalClosure"}},
		},
		Outputs: Outputs{ModelPath: "model.qnt", ArtifactPath: "model.formal.generated.json"},
		States: []State{
			{ID: "idle", Quint: "Idle", Initial: true},
			{ID: "busy", Quint: "Busy"},
			{ID: "done", Quint: "Done", Terminal: true},
		},
		Events: []Event{
			{ID: "start", Quint: "Start"},
			{ID: "finish", Quint: "Finish"},
		},
		TransitionDefaults: TransitionDefaults{
			Invalid:  &DefaultTransition{To: "self", WantError: true},
			Terminal: &DefaultTransition{To: "self", WantError: true},
		},
		Transitions: []Transition{
			{From: StringList{"idle"}, Event: StringList{"start"}, To: "busy"},
			{From: StringList{"busy"}, Event: StringList{"finish"}, To: "done"},
		},
		Invariants: []Invariant{
			{ID: "type_ok", Quint: "TypeOK", Description: "Types remain valid."},
			{ID: "terminal_closure", Quint: "TerminalClosure", Description: "Terminal states are closed."},
		},
		Traces: []Trace{
			{Name: "success", Initial: "idle", Steps: []TraceStep{{Event: "start", Want: "busy"}, {Event: "finish", Want: "done"}}},
		},
	}
}

func assertTransition(t *testing.T, c Contract, from string, event string, to string, wantError bool) {
	t.Helper()
	for _, transition := range c.ExpandedTransitions {
		if transition.From == from && transition.Event == event {
			if transition.To != to || transition.WantError != wantError {
				t.Fatalf("%s/%s = %#v, want to=%s wantError=%v", from, event, transition, to, wantError)
			}
			return
		}
	}
	t.Fatalf("missing transition %s/%s", from, event)
}

func requireErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}
