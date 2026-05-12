package codegen

import (
	"strings"
	"testing"

	"flow-verifier/internal/verification/artifact"
	"flow-verifier/internal/flows/model"
	"flow-verifier/internal/testkit"
)

func TestRenderGoEmitsTransitionCoreHelpers(t *testing.T) {
	raw := testkit.ValidRawContract()
	flow := testkit.MustCompile(t, raw)

	rendered, err := Render(flow, validArtifact(flow), Options{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := findGeneratedFile(t, rendered, flow.Layout.RuntimePath)

	for _, want := range []string{
		"package " + "generated",
		"type Status string",
		"type Event string",
		"func ExampleIsValidEvent(status Status, event Event) bool",
		"func ExampleNextStatus(status Status, event Event) Status",
		"func TransitionExampleStatus(status Status, event Event) (Status, error)",
		"return ExampleBusy",
		"fmt.Errorf(\"cannot apply %s from %s\", event, status)",
	} {
		if !strings.Contains(runtime, want) {
			t.Fatalf("generated Go runtime missing %q:\n%s", want, runtime)
		}
	}
}

func TestRenderGoEmitsReplayHelper(t *testing.T) {
	raw := testkit.ValidRawContract()
	flow := testkit.MustCompile(t, raw)

	rendered, err := Render(flow, validArtifact(flow), Options{})
	if err != nil {
		t.Fatal(err)
	}
	replay := findGeneratedFile(t, rendered, flow.Layout.ReplayHelperPath)

	for _, want := range []string{
		"package " + "generated",
		"type Transition = modeltest.Transition[Status, Event]",
		"func RunReplay(t *testing.T, transition Transition)",
		"AssertFormalArtifactFresh",
		"AssertFormalTransitionsReplay",
		"AssertFormalTracesReplay",
	} {
		if !strings.Contains(replay, want) {
			t.Fatalf("generated Go replay helper missing %q:\n%s", want, replay)
		}
	}
}

func TestRenderTypeScriptEmitsRuntimeHelpers(t *testing.T) {
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())

	rendered, err := Render(flow, validArtifact(flow), Options{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := findGeneratedFile(t, rendered, flow.Layout.RuntimePath)

	for _, want := range []string{
		"const exampleTransitionTable",
		"export const isExampleEventValid",
		"export const nextExampleStatus",
		"export const transitionExampleStatus",
		"export type ExampleState =",
		"export type ExampleEvent =",
		"export const exampleReplayFixtureContract",
	} {
		if !strings.Contains(runtime, want) {
			t.Fatalf("generated TypeScript runtime missing %q:\n%s", want, runtime)
		}
	}
}

func TestRenderTypeScriptEmitsReplayHelper(t *testing.T) {
	flow := testkit.MustCompile(t, testkit.ValidTypeScriptRawContract())

	rendered, err := Render(flow, validArtifact(flow), Options{})
	if err != nil {
		t.Fatal(err)
	}
	helper := findGeneratedFile(t, rendered, flow.Layout.ReplayHelperPath)

	for _, want := range []string{
		"export interface ExampleFormalReplayFixtures",
		"export const runFormalReplay",
		"transitionFromReplayAdapter",
		"statusOf: (state) => state.status",
		"assertFormalTransitionsReplay",
		"assertFormalTracesReplay",
	} {
		if !strings.Contains(helper, want) {
			t.Fatalf("generated TypeScript replay helper missing %q:\n%s", want, helper)
		}
	}
}

func findGeneratedFile(t *testing.T, rendered RenderResult, path string) string {
	t.Helper()
	for _, file := range rendered.Files {
		if file.Path == path {
			return string(file.Data)
		}
	}
	paths := make([]string, 0, len(rendered.Files))
	for _, file := range rendered.Files {
		paths = append(paths, file.Path)
	}
	t.Fatalf("missing generated file %s; have: %v", path, paths)
	return ""
}

func validArtifact(flow model.Flow) artifact.Artifact {
	return artifact.Artifact{
		Source: artifact.Source{
			ContractPath:     flow.ContractPath,
			ContractSHA256:   strings.Repeat("a", 64),
			ModelPath:        flow.Layout.ModelPath,
			ModelSHA256:      strings.Repeat("b", 64),
			GeneratorPath:    artifact.GeneratorPath,
			GeneratorSHA256:  strings.Repeat("c", 64),
			GeneratorVersion: artifact.GeneratorVersion,
		},
		Invariants:      []string{"TypeOK"},
		GeneratedChecks: []string{artifact.GeneratedCheckTransitionTable},
	}
}
