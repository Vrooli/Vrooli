package codegen

import (
	"strings"
	"testing"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/testkit"
)

func TestRenderGoEmitsTransitionCoreHelpers(t *testing.T) {
	c := validCodegenContract("workflow.generated.go")
	c.Runtime = contract.Runtime{
		Go: &contract.GoRuntime{Package: "example", StatusType: "Status", EventType: "Event", ConstantPrefix: "Example"},
	}
	flow := testkit.MustCompile(t, c)

	rendered, err := Render(flow, validArtifact(flow))
	if err != nil {
		t.Fatal(err)
	}
	declarations := findGeneratedFile(t, rendered, c.Outputs.DeclarationsPath)

	for _, want := range []string{
		"func ExampleIsValidEvent(status Status, event Event) bool",
		"func ExampleNextStatus(status Status, event Event) Status",
		"func TransitionExampleStatus(status Status, event Event) (Status, error)",
		"return ExampleBusy",
		"fmt.Errorf(\"cannot apply %s from %s\", event, status)",
	} {
		if !strings.Contains(declarations, want) {
			t.Fatalf("generated Go declarations missing %q:\n%s", want, declarations)
		}
	}
}

func TestRenderGoEmitsReplayTest(t *testing.T) {
	c := validCodegenContract("api/internal/example/workflow.generated.go")
	c.Outputs.ReplayTestPath = "api/internal/example/workflow_formal_replay_test.generated.go"
	c.Runtime = contract.Runtime{
		Go: &contract.GoRuntime{Package: "example", StatusType: "Status", EventType: "Event", ConstantPrefix: "Example"},
	}
	c.Replay = contract.Replay{
		Kind:     string(model.ReplayKindGoTest),
		TestPath: c.Outputs.ReplayTestPath,
		Transition: contract.ReplayTransition{
			Function:    "TransitionExample",
			StateType:   "State",
			StatusField: "Status",
		},
	}
	flow := testkit.MustCompile(t, c)

	rendered, err := Render(flow, validArtifact(flow))
	if err != nil {
		t.Fatal(err)
	}
	replayTest := findGeneratedFile(t, rendered, c.Outputs.ReplayTestPath)

	for _, want := range []string{
		"func TestExampleFormalReplay_ReplaysGeneratedModelArtifacts",
		"AssertFormalArtifactFresh",
		"AssertFormalTransitionsReplay",
		"AssertFormalTracesReplay",
		"next, err := example.TransitionExample(example.State{Status: status}, event)",
	} {
		if !strings.Contains(replayTest, want) {
			t.Fatalf("generated Go replay test missing %q:\n%s", want, replayTest)
		}
	}
}

func TestRenderTypeScriptEmitsTransitionCoreHelpers(t *testing.T) {
	c := validCodegenContract("workflow.generated.ts")
	c.Outputs.ReplayHelperPath = "workflow.formal-replay.generated.ts"
	c.Outputs.ReplayTestPath = "workflow.formal.test.generated.ts"
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
	c.Replay = contract.Replay{
		Kind:          string(model.ReplayKindVitest),
		HelperPath:    c.Outputs.ReplayHelperPath,
		TestPath:      c.Outputs.ReplayTestPath,
		FixtureModule: "./workflow.formal-fixtures",
		FixtureExport: "exampleFormalFixtures",
		Transition: contract.ReplayTransition{
			Module:         "./workflow",
			Function:       "transitionExample",
			StatusAccessor: "state.status",
		},
	}
	flow := testkit.MustCompile(t, c)

	rendered, err := Render(flow, validArtifact(flow))
	if err != nil {
		t.Fatal(err)
	}
	declarations := findGeneratedFile(t, rendered, c.Outputs.DeclarationsPath)

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
		if !strings.Contains(declarations, want) {
			t.Fatalf("generated TypeScript declarations missing %q:\n%s", want, declarations)
		}
	}
}

func TestRenderTypeScriptEmitsReplayHarness(t *testing.T) {
	c := validCodegenContract("ui/src/features/example/workflow.generated.ts")
	c.Outputs.ReplayHelperPath = "ui/src/features/example/workflow.formal-replay.generated.ts"
	c.Outputs.ReplayTestPath = "ui/src/features/example/workflow.formal.test.generated.ts"
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
	c.Replay = contract.Replay{
		Kind:          string(model.ReplayKindVitest),
		HelperPath:    c.Outputs.ReplayHelperPath,
		TestPath:      c.Outputs.ReplayTestPath,
		FixtureModule: "./workflow.formal-fixtures",
		FixtureExport: "exampleFormalFixtures",
		Transition: contract.ReplayTransition{
			Module:         "./workflow",
			Function:       "transitionExample",
			StatusAccessor: "state.status",
		},
	}
	flow := testkit.MustCompile(t, c)

	rendered, err := Render(flow, validArtifact(flow))
	if err != nil {
		t.Fatal(err)
	}
	helper := findGeneratedFile(t, rendered, c.Outputs.ReplayHelperPath)
	testFile := findGeneratedFile(t, rendered, c.Outputs.ReplayTestPath)

	for _, want := range []string{
		"export interface ExampleFormalReplayFixtures",
		"assertFormalArtifactFreshFromFiles",
		"assertFormalTransitionsReplay",
		"assertFormalTracesReplay",
		"transitionFromReplayAdapter",
		"statusOf: (state) => state.status",
	} {
		if !strings.Contains(helper, want) {
			t.Fatalf("generated TypeScript replay helper missing %q:\n%s", want, helper)
		}
	}
	for _, want := range []string{
		"import { exampleFormalFixtures } from \"./workflow.formal-fixtures\"",
		"import { assertExampleFormalReplay } from \"./workflow.formal-replay.generated\"",
		"assertExampleFormalReplay(formalArtifact as FormalArtifact, exampleFormalFixtures)",
	} {
		if !strings.Contains(testFile, want) {
			t.Fatalf("generated TypeScript replay test missing %q:\n%s", want, testFile)
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
		Outputs: contract.Outputs{
			ModelPath:        "workflow.qnt",
			ArtifactPath:     "workflow.formal.generated.json",
			DeclarationsPath: declarationsPath,
			ReplayTestPath:   "workflow_formal_replay_test.generated.go",
		},
		States:             []contract.State{{ID: "idle", Quint: "Idle", Initial: true}, {ID: "busy", Quint: "Busy"}},
		Events:             []contract.Event{{ID: "start", Quint: "Start"}},
		TransitionDefaults: contract.TransitionDefaults{Invalid: &contract.DefaultTransition{To: model.SelfTarget, WantError: true}},
		Transitions:        []contract.Transition{{From: contract.StringList{"idle"}, Event: contract.StringList{"start"}, To: "busy"}},
		Invariants:         []contract.Invariant{{ID: "type_ok", Quint: "TypeOK", Description: "Type OK."}},
		Traces:             []contract.Trace{{Name: "success", Initial: "idle", Steps: []contract.TraceStep{{Event: "start", Want: "busy"}}}},
		Replay: contract.Replay{
			Kind:     string(model.ReplayKindGoTest),
			TestPath: "workflow_formal_replay_test.generated.go",
			Transition: contract.ReplayTransition{
				Function:    "TransitionExample",
				StateType:   "State",
				StatusField: "Status",
			},
		},
	}
}

func findGeneratedFile(t *testing.T, rendered RenderResult, path string) string {
	t.Helper()
	for _, file := range rendered.Files {
		if file.Path == path {
			return string(file.Data)
		}
	}
	t.Fatalf("missing generated file %s in %#v", path, rendered.Files)
	return ""
}

func validArtifact(flow model.Flow) artifact.Artifact {
	return artifact.Artifact{
		Source: artifact.Source{
			ContractPath:     flow.ContractPath,
			ContractSHA256:   strings.Repeat("a", 64),
			ModelPath:        flow.Outputs.ModelPath,
			ModelSHA256:      strings.Repeat("b", 64),
			GeneratorPath:    artifact.GeneratorPath,
			GeneratorSHA256:  strings.Repeat("c", 64),
			GeneratorVersion: artifact.GeneratorVersion,
		},
		Invariants:      []string{"TypeOK"},
		GeneratedChecks: []string{artifact.GeneratedCheckTransitionTable},
	}
}
