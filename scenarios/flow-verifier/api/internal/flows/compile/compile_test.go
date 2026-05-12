package compile_test

import (
	"testing"

	"flow-verifier/internal/flows/compile"
	"flow-verifier/internal/flows/contract"
	"flow-verifier/internal/flows/model"
	"flow-verifier/internal/testkit"
)

func TestCompileValidContractBuildsCompleteMatrix(t *testing.T) {
	flow := testkit.MustCompile(t, testkit.ValidRawContract())
	if got, want := flow.Matrix.Len(), len(flow.States)*len(flow.Events); got != want {
		t.Fatalf("matrix rows = %d, want %d", got, want)
	}
	transition, ok := flow.Matrix.Lookup("idle", "start")
	assertTransition(t, transition, ok, "busy", false)
	transition, ok = flow.Matrix.Lookup("done", "finish")
	assertTransition(t, transition, ok, "done", true)
}

func TestCompileRejectsDuplicateStateIDs(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.States[1].ID = "idle"
	testkit.RequireErrorContains(t, compileErr(raw), "duplicate states id idle")
}

func TestCompileRejectsDuplicateEventIDs(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Events[1].ID = "start"
	testkit.RequireErrorContains(t, compileErr(raw), "duplicate events id start")
}

func TestCompileRejectsDuplicateQuintTags(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Events[0].Quint = "Idle"
	testkit.RequireErrorContains(t, compileErr(raw), "duplicate Quint tag Idle")
}

func TestCompileRejectsMissingInitialState(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.States[0].Initial = false
	testkit.RequireErrorContains(t, compileErr(raw), "states must declare exactly one initial state, got 0")
}

func TestCompileRejectsMultipleInitialStates(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.States[1].Initial = true
	testkit.RequireErrorContains(t, compileErr(raw), "states must declare exactly one initial state, got 2")
}

func TestCompileRejectsUnknownTransitionStateAndEvent(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Transitions = append(raw.Transitions, contract.Transition{From: contract.StringList{"ghost"}, Event: contract.StringList{"missing"}, To: "done"})
	testkit.RequireErrorContains(t, compileErr(raw), "references unknown")
}

func TestCompileRejectsDuplicateTransitionPair(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Transitions = append(raw.Transitions, contract.Transition{From: contract.StringList{"idle"}, Event: contract.StringList{"start"}, To: "busy"})
	testkit.RequireErrorContains(t, compileErr(raw), "duplicate transition pair idle/start")
}

func TestCompileRejectsUnknownInvariantReference(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Model.Verify.Invariants = append(raw.Model.Verify.Invariants, "GhostInvariant")
	testkit.RequireErrorContains(t, compileErr(raw), "references unknown invariant GhostInvariant")
}

func TestCompileRejectsTraceTargetDrift(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Traces[0].Steps[0].Want = "done"
	testkit.RequireErrorContains(t, compileErr(raw), "expanded transition wants busy wantError=false")
}

func TestCompileRejectsTraceWantErrorDrift(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Traces[0].Steps[0].WantError = true
	testkit.RequireErrorContains(t, compileErr(raw), "declares want=busy wantError=true")
}

func TestCompileAcceptsSelfTransitionErrorTrace(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Traces = append(raw.Traces, contract.Trace{
		Name:    "rejected_terminal_event_preserves_state",
		Initial: "done",
		Steps: []contract.TraceStep{
			{Event: "start", Want: "done", WantError: true},
			{Event: "finish", Want: "done", WantError: true},
		},
	})
	if _, err := compile.Compile(raw); err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
}

func TestCompileRejectsInvalidQuintName(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.States[0].Quint = "not-valid"
	testkit.RequireErrorContains(t, compileErr(raw), "invalid Quint identifier not-valid")
}

func TestCompileRejectsMissingReplayTransitionFunction(t *testing.T) {
	raw := testkit.ValidRawContract()
	raw.Replay.Transition.Function = ""
	testkit.RequireErrorContains(t, compileErr(raw), "replay.transition.function is required")
}

func TestCompileRejectsNonExhaustiveTypeScriptStateVariants(t *testing.T) {
	raw := testkit.ValidTypeScriptRawContract()
	delete(raw.Runtime.TypeScript.StateVariants, "done")
	testkit.RequireErrorContains(t, compileErr(raw), "runtime.typescript.stateVariants missing variant for done")
}

func TestCompileRejectsUnknownTypeScriptEventVariant(t *testing.T) {
	raw := testkit.ValidTypeScriptRawContract()
	raw.Runtime.TypeScript.EventVariants["ghost"] = map[string]string{}
	testkit.RequireErrorContains(t, compileErr(raw), "runtime.typescript.eventVariants references unknown id ghost")
}

func TestCompileRejectsUnknownTypeScriptPayloadAlias(t *testing.T) {
	raw := testkit.ValidTypeScriptRawContract()
	raw.Runtime.TypeScript.StateVariants["busy"]["file"] = "missing"
	testkit.RequireErrorContains(t, compileErr(raw), "runtime.typescript.stateVariants.busy.file references unknown payload alias missing")
}

func compileErr(raw contract.Contract) error {
	_, err := compile.Compile(raw)
	return err
}

func assertTransition(t *testing.T, transition model.Transition, ok bool, to string, wantError bool) {
	t.Helper()
	if !ok {
		t.Fatal("missing transition")
	}
	if transition.To != to || transition.WantError != wantError {
		t.Fatalf("transition = %#v, want to=%s wantError=%v", transition, to, wantError)
	}
}
