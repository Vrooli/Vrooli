package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestValidateAndExpandRejectsTraceTargetDrift(t *testing.T) {
	c := validContract()
	c.Traces[0].Steps[0].Want = "done"
	requireErrorContains(t, ValidateAndExpand(&c), "expanded transition wants busy wantError=false")
}

func TestValidateAndExpandRejectsTraceWantErrorDrift(t *testing.T) {
	c := validContract()
	c.Traces[0].Steps[0].WantError = true
	requireErrorContains(t, ValidateAndExpand(&c), "declares want=busy wantError=true")
}

func TestValidateAndExpandAcceptsSelfTransitionErrorTrace(t *testing.T) {
	c := validContract()
	c.Traces = append(c.Traces, Trace{
		Name:    "rejected_terminal_event_preserves_state",
		Initial: "done",
		Steps: []TraceStep{
			{Event: "start", Want: "done", WantError: true},
			{Event: "finish", Want: "done", WantError: true},
		},
	})
	if err := ValidateAndExpand(&c); err != nil {
		t.Fatalf("ValidateAndExpand() error = %v", err)
	}
}

func TestValidateAndExpandRejectsInvalidQuintName(t *testing.T) {
	c := validContract()
	c.States[0].Quint = "not-valid"
	requireErrorContains(t, ValidateAndExpand(&c), "invalid Quint identifier not-valid")
}

func TestValidateAndExpandRejectsMissingReplayBindings(t *testing.T) {
	c := validContract()
	c.Replay.Bindings = nil
	requireErrorContains(t, ValidateAndExpand(&c), "replay.bindings must declare at least one production replay test binding")
}

func TestValidateAndExpandRejectsReplayBindingPathTraversal(t *testing.T) {
	c := validContract()
	c.Replay.Bindings[0].Path = "../outside_test.go"
	requireErrorContains(t, ValidateAndExpand(&c), "path must be a relative path inside the scenario root")
}

func TestValidateAndExpandRejectsNonExhaustiveTypeScriptStateVariants(t *testing.T) {
	c := validTypeScriptContract()
	delete(c.Runtime.TypeScript.StateVariants, "done")
	requireErrorContains(t, ValidateAndExpand(&c), "runtime.typescript.stateVariants missing variant for done")
}

func TestValidateAndExpandRejectsUnknownTypeScriptEventVariant(t *testing.T) {
	c := validTypeScriptContract()
	c.Runtime.TypeScript.EventVariants["ghost"] = map[string]string{}
	requireErrorContains(t, ValidateAndExpand(&c), "runtime.typescript.eventVariants references unknown id ghost")
}

func TestValidateAndExpandRejectsUnknownTypeScriptPayloadAlias(t *testing.T) {
	c := validTypeScriptContract()
	c.Runtime.TypeScript.StateVariants["busy"]["file"] = "missing"
	requireErrorContains(t, ValidateAndExpand(&c), "runtime.typescript.stateVariants.busy.file references unknown payload alias missing")
}

func TestLoadRejectsSchemaViolations(t *testing.T) {
	root := t.TempDir()
	for name, mutate := range map[string]func(map[string]any){
		"unknown property": func(body map[string]any) { body["unexpected"] = true },
		"missing required": func(body map[string]any) { delete(body, "flowId") },
		"invalid enum": func(body map[string]any) {
			replay := body["replay"].(map[string]any)
			bindings := replay["bindings"].([]any)
			binding := bindings[0].(map[string]any)
			binding["kind"] = "jest"
		},
	} {
		t.Run(name, func(t *testing.T) {
			body := mustJSONMap(t, validContract())
			mutate(body)
			path := filepath.Join(root, strings.ReplaceAll(name, " ", "_")+".flow.json")
			writeJSON(t, path, body)
			_, err := Load(path, filepath.Base(path))
			requireErrorContains(t, err, "schema validation failed")
		})
	}
}

func TestValidateReplayBindingsRequiresMarkerAndHelpers(t *testing.T) {
	root := t.TempDir()
	c := validContract()
	writeBindingFile(t, root, "workflow_test.go", "TestWorkflow_ReplaysFormalModelArtifacts\nAssertFormalArtifactFresh\nAssertFormalTransitionsReplay\n")
	requireErrorContains(t, ValidateReplayBindings(c, root), "AssertFormalTracesReplay")
	writeBindingFile(t, root, "workflow_test.go", "TestWorkflow_ReplaysFormalModelArtifacts\nAssertFormalArtifactFresh\nAssertFormalTransitionsReplay\nAssertFormalTracesReplay\n")
	if err := ValidateReplayBindings(c, root); err != nil {
		t.Fatalf("ValidateReplayBindings() error = %v", err)
	}
}

func validContract() Contract {
	return Contract{
		SchemaVersion: SchemaVersion,
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
		Outputs: Outputs{ModelPath: "model.qnt", ArtifactPath: "model.formal.generated.json", DeclarationsPath: "model.generated.go"},
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
		Runtime: Runtime{
			Go: &GoRuntime{Package: "example", StatusType: "Status", EventType: "Event", ConstantPrefix: "Example"},
		},
		Replay: Replay{Bindings: []ReplayBinding{{Kind: "go-test", Path: "workflow_test.go", Assertion: "TestWorkflow_ReplaysFormalModelArtifacts"}}},
	}
}

func validTypeScriptContract() Contract {
	c := validContract()
	c.Outputs.DeclarationsPath = "workflow.generated.ts"
	c.Runtime = Runtime{
		TypeScript: &TypeScriptRuntime{
			StatusType:             "ExampleStatus",
			EventType:              "ExampleEventType",
			StatusesConst:          "exampleStatuses",
			EventsConst:            "exampleEvents",
			FormalExpectationConst: "exampleFormalExpectation",
			StateUnionType:         "ExampleState",
			EventUnionType:         "ExampleEvent",
			PayloadTypes:           map[string]string{"file": "File", "message": "string"},
			StateVariants: map[string]map[string]string{
				"idle": {},
				"busy": {"file": "file"},
				"done": {"message": "message"},
			},
			EventVariants: map[string]map[string]string{
				"start":  {"file": "file"},
				"finish": {},
			},
		},
	}
	c.Replay.Bindings = []ReplayBinding{{Kind: "vitest", Path: "workflow.test.ts", Assertion: "replays generated formal model artifacts"}}
	return c
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

func writeBindingFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustJSONMap(t *testing.T, c Contract) map[string]any {
	t.Helper()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func writeJSON(t *testing.T, path string, body map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
