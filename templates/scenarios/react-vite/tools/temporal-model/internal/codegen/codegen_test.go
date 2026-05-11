package codegen

import (
	"strings"
	"testing"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/contract"
)

func TestRenderGoEmitsTransitionCoreHelpers(t *testing.T) {
	c := validCodegenContract("workflow.generated.go")
	c.Runtime = contract.Runtime{
		Go: &contract.GoRuntime{Package: "example", StatusType: "Status", EventType: "Event", ConstantPrefix: "Example"},
	}
	if err := contract.ValidateAndExpand(&c); err != nil {
		t.Fatal(err)
	}

	rendered, err := Render(c, validArtifact(c))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"func ExampleIsValidEvent(status Status, event Event) bool",
		"func ExampleNextStatus(status Status, event Event) Status",
		"func TransitionExampleStatus(status Status, event Event) (Status, error)",
		"return ExampleBusy",
		"fmt.Errorf(\"cannot apply %s from %s\", event, status)",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("generated Go declarations missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderTypeScriptEmitsTransitionCoreHelpers(t *testing.T) {
	c := validCodegenContract("workflow.generated.ts")
	c.Runtime = contract.Runtime{
		TypeScript: &contract.TypeScriptRuntime{
			StatusType:             "ExampleStatus",
			EventType:              "ExampleEvent",
			StatusesConst:          "exampleStatuses",
			EventsConst:            "exampleEvents",
			FormalExpectationConst: "exampleFormalExpectation",
			StateUnionType:         "ExampleState",
			EventUnionType:         "ExampleRuntimeEvent",
			PayloadTypes:           map[string]string{"file": "File", "message": "string"},
			StateVariants: map[string]map[string]string{
				"idle": {},
				"busy": {"file": "file"},
			},
			EventVariants: map[string]map[string]string{
				"start": {"message": "message"},
			},
		},
	}
	if err := contract.ValidateAndExpand(&c); err != nil {
		t.Fatal(err)
	}

	rendered, err := Render(c, validArtifact(c))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"const exampleTransitionTable",
		"export const isExampleEventValid",
		"export const nextExampleStatus",
		"export const transitionExampleStatus",
		"Record<ExampleStatus, Record<ExampleEvent, ExampleTransitionRow>>",
		"export type ExampleState =",
		"{ readonly status: \"busy\"; readonly file: File }",
		"export type ExampleRuntimeEvent =",
		"{ readonly type: \"start\"; readonly message: string }",
		"export type ExampleStateFixtureMap<RuntimeState = ExampleState>",
		"export type ExampleEventFixtureMap<RuntimeEvent = ExampleRuntimeEvent>",
		"export const exampleReplayFixtureContract",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("generated TypeScript declarations missing %q:\n%s", want, rendered)
		}
	}
}

func validCodegenContract(declarationsPath string) contract.Contract {
	return contract.Contract{
		SchemaVersion: contract.SchemaVersion,
		FlowID:        "example.flow",
		Domain:        "example",
		Description:   "Example.",
		ContractPath:  "workflow.flow.json",
		Model: contract.Model{
			Module:     "Example",
			Seed:       "1",
			MaxSteps:   2,
			TraceCount: 1,
			Verify:     contract.Verify{Invariants: []string{"TypeOK"}},
		},
		Outputs:            contract.Outputs{ModelPath: "workflow.qnt", ArtifactPath: "workflow.formal.generated.json", DeclarationsPath: declarationsPath},
		States:             []contract.State{{ID: "idle", Quint: "Idle", Initial: true}, {ID: "busy", Quint: "Busy"}},
		Events:             []contract.Event{{ID: "start", Quint: "Start"}},
		TransitionDefaults: contract.TransitionDefaults{Invalid: &contract.DefaultTransition{To: "self", WantError: true}},
		Transitions:        []contract.Transition{{From: contract.StringList{"idle"}, Event: contract.StringList{"start"}, To: "busy"}},
		Invariants:         []contract.Invariant{{ID: "type_ok", Quint: "TypeOK", Description: "Type OK."}},
		Traces:             []contract.Trace{{Name: "success", Initial: "idle", Steps: []contract.TraceStep{{Event: "start", Want: "busy"}}}},
		Replay:             contract.Replay{Bindings: []contract.ReplayBinding{{Kind: "go-test", Path: "workflow_test.go", Assertion: "TestWorkflow_ReplaysFormalModelArtifacts"}}},
	}
}

func validArtifact(c contract.Contract) artifact.Artifact {
	return artifact.Artifact{
		Source: artifact.Source{
			ContractPath:     c.ContractPath,
			ContractSHA256:   strings.Repeat("a", 64),
			ModelPath:        c.Outputs.ModelPath,
			ModelSHA256:      strings.Repeat("b", 64),
			GeneratorPath:    artifact.GeneratorPath,
			GeneratorSHA256:  strings.Repeat("c", 64),
			GeneratorVersion: artifact.GeneratorVersion,
		},
		Invariants:      []string{"TypeOK"},
		GeneratedChecks: []string{TransitionTableGeneratedCheck},
	}
}
