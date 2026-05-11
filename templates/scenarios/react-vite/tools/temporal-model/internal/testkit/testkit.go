package testkit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/compile"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/quint"
)

func ValidRawContract() contract.Contract {
	return contract.Contract{
		SchemaVersion: model.SchemaVersion,
		FlowID:        "example.flow",
		Domain:        "example",
		Description:   "Example flow.",
		ContractPath:  "example.flow.json",
		Model: contract.Model{
			Module:     "ExampleFlow",
			Seed:       "1",
			MaxSteps:   4,
			TraceCount: 2,
			Verify:     contract.Verify{Invariants: []string{"TypeOK", "TerminalClosure"}},
		},
		Outputs: contract.Outputs{
			ModelPath:        "model.qnt",
			ArtifactPath:     "model.formal.generated.json",
			DeclarationsPath: "model.generated.go",
			ReplayTestPath:   "model_formal_replay_test.generated.go",
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
			Kind:     string(model.ReplayKindGoTest),
			TestPath: "model_formal_replay_test.generated.go",
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
	raw.Outputs.DeclarationsPath = "workflow.generated.ts"
	raw.Outputs.ReplayHelperPath = "workflow.formal-replay.generated.ts"
	raw.Outputs.ReplayTestPath = "workflow.formal.test.generated.ts"
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
		Kind:          string(model.ReplayKindVitest),
		HelperPath:    "workflow.formal-replay.generated.ts",
		TestPath:      "workflow.formal.test.generated.ts",
		FixtureModule: "./workflow.formal-fixtures",
		FixtureExport: "exampleFormalFixtures",
		Transition: contract.ReplayTransition{
			Module:         "./workflow",
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
		body := `{"states":[{"status":{"tag":"Idle"},"event":{"tag":"Start"},"rejected":false},{"status":{"tag":"Busy"},"event":{"tag":"Start"},"rejected":false}]}`
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return quint.Result{}, err
		}
	}
	return quint.Result{Stdout: "ok"}, nil
}
