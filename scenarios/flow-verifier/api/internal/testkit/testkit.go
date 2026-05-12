package testkit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/flows/compile"
	"flow-verifier/internal/flows/contract"
	"flow-verifier/internal/flows/layout"
	"flow-verifier/internal/flows/model"
	"flow-verifier/internal/verification/quint"
)

func ValidRawContract() contract.Contract {
	contractPath := "api/internal/example/flow/flow.json"
	flowID := "example.workflow.api"
	lay, _ := layout.Derive(contractPath, layout.LanguageGo)
	return contract.Contract{
		SchemaVersion: model.SchemaVersion,
		FlowID:        flowID,
		Domain:        "example",
		Description:   "Example flow.",
		ContractPath:  contractPath,
		Layout:        lay,
		Model: contract.Model{
			Module:     "ExampleFlow",
			Seed:       "1",
			MaxSteps:   4,
			TraceCount: 2,
			Verify:     contract.Verify{Invariants: []string{"TypeOK", "TerminalClosure"}},
		},
		States: []contract.State{
			{ID: "idle", Quint: "Idle", Initial: true},
			{ID: "busy", Quint: "Busy"},
			{ID: "done", Quint: "Done", Terminal: true},
		},
		Events: []contract.Event{
			{ID: "start", Quint: "Start"},
			{ID: "finish", Quint: "Finish"},
		},
		TransitionDefaults: contract.TransitionDefaults{
			Invalid:  &contract.DefaultTransition{To: model.SelfTarget, WantError: true},
			Terminal: &contract.DefaultTransition{To: model.SelfTarget, WantError: true},
		},
		Transitions: []contract.Transition{
			{From: contract.StringList{"idle"}, Event: contract.StringList{"start"}, To: "busy"},
			{From: contract.StringList{"busy"}, Event: contract.StringList{"finish"}, To: "done"},
		},
		Invariants: []contract.Invariant{
			{ID: "type_ok", Quint: "TypeOK", Description: "Types remain valid."},
			{ID: "terminal_closure", Quint: "TerminalClosure", Description: "Terminal states are closed."},
		},
		Traces: []contract.Trace{
			{Name: "success", Initial: "idle", Steps: []contract.TraceStep{{Event: "start", Want: "busy"}, {Event: "finish", Want: "done"}}},
		},
		Runtime: contract.Runtime{
			Go: &contract.GoRuntime{Package: "example", StatusType: "Status", EventType: "Event", ConstantPrefix: "Example"},
		},
		Replay: contract.Replay{
			Transition: contract.ReplayTransition{
				Function:    "TransitionExample",
				StateType:   "State",
				StatusField: "Status",
			},
		},
	}
}

func ValidTypeScriptRawContract() contract.Contract {
	raw := ValidRawContract()
	raw.ContractPath = "ui/src/features/example/flow/flow.json"
	raw.FlowID = "example.workflow.ui"
	raw.Layout, _ = layout.Derive(raw.ContractPath, layout.LanguageTypeScript)
	raw.Runtime = contract.Runtime{
		TypeScript: &contract.TypeScriptRuntime{
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
	raw.Replay = contract.Replay{
		Transition: contract.ReplayTransition{
			Function:       "transitionExample",
			StatusAccessor: "state.status",
		},
	}
	return raw
}

func MustCompile(t *testing.T, raw contract.Contract) model.Flow {
	t.Helper()
	flow, err := compile.Compile(raw)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return flow
}

func MustJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func WriteJSONMap(t *testing.T, path string, body map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	WriteFile(t, filepath.Dir(path), filepath.Base(path), string(data))
}

func WriteFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func WriteFlowJSON(t *testing.T, root string, rel string, raw contract.Contract) {
	t.Helper()
	raw.ContractPath = ""
	body := MustJSONMap(t, raw)
	WriteJSONMap(t, filepath.Join(root, filepath.FromSlash(rel)), body)
}

func RequireErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}

type FakeRunner struct{}

func (FakeRunner) Run(_ context.Context, command quint.Command) (quint.Result, error) {
	if len(command.Args) >= 2 && command.Args[1] == "run" {
		pattern := command.Args[len(command.Args)-1]
		path := strings.Replace(pattern, "{seq}", "1", 1)
		body, err := json.Marshal(fakeITFTrace{
			States: []fakeITFState{
				{Status: fakeITFTag{Tag: "Idle"}, Event: fakeITFTag{Tag: "Start"}},
				{Status: fakeITFTag{Tag: "Busy"}, Event: fakeITFTag{Tag: "Start"}},
			},
		})
		if err != nil {
			return quint.Result{}, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return quint.Result{}, err
		}
	}
	return quint.Result{Stdout: "ok"}, nil
}

type fakeITFTrace struct {
	States []fakeITFState `json:"states"`
}

type fakeITFState struct {
	Status   fakeITFTag `json:"status"`
	Event    fakeITFTag `json:"event"`
	Rejected bool       `json:"rejected"`
}

type fakeITFTag struct {
	Tag string `json:"tag"`
}
