package codegen

import (
	"fmt"
	goformat "go/format"
	"path/filepath"
	"strings"

	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/layout"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/verification/artifact"
)

// renderGoRuntime emits <flow>/generated/runtime.go: the state/event
// consts, transition table helpers, and freshness hashes for the flow.
// The package is always `generated`; the hand-authored wrapper (package
// `flow`) imports it as e.g. generated.StatusIdle.
func renderGoRuntime(flow model.Flow, built artifact.Artifact) (string, error) {
	if flow.Runtime.Go == nil {
		return "", fmt.Errorf("runtime.go is required for %s", flow.FlowID)
	}
	rt := flow.Runtime.Go
	pkg := layout.GeneratedDirName
	var b strings.Builder
	b.WriteString(generatedHeader(flow))
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString("import \"fmt\"\n\n")
	fmt.Fprintf(&b, "type %s string\n", rt.StatusType)
	fmt.Fprintf(&b, "type %s string\n\n", rt.EventType)
	b.WriteString("const (\n")
	for _, state := range flow.States {
		fmt.Fprintf(&b, "\t%s%s %s = %q\n", rt.ConstantPrefix, pascal(state.ID), rt.StatusType, state.ID)
	}
	b.WriteString(")\n\n")
	b.WriteString("const (\n")
	for _, event := range flow.Events {
		fmt.Fprintf(&b, "\t%s%s %s = %q\n", rt.ConstantPrefix, pascal(event.ID), rt.EventType, event.ID)
	}
	b.WriteString(")\n\n")
	fmt.Fprintf(&b, "const %sContractPath = %q\n", rt.ConstantPrefix, built.Source.ContractPath)
	fmt.Fprintf(&b, "const %sModelPath = %q\n", rt.ConstantPrefix, built.Source.ModelPath)
	fmt.Fprintf(&b, "const %sGeneratorPath = %q\n", rt.ConstantPrefix, built.Source.GeneratorPath)
	fmt.Fprintf(&b, "const %sContractSHA256 = %q\n", rt.ConstantPrefix, built.Source.ContractSHA256)
	fmt.Fprintf(&b, "const %sModelSHA256 = %q\n", rt.ConstantPrefix, built.Source.ModelSHA256)
	fmt.Fprintf(&b, "const %sGeneratorSHA256 = %q\n\n", rt.ConstantPrefix, built.Source.GeneratorSHA256)
	fmt.Fprintf(&b, "var %sFormalInvariants = []string{%s}\n", lowerFirst(rt.ConstantPrefix), quotedList(built.Invariants))
	fmt.Fprintf(&b, "var %sFormalGeneratedChecks = []string{%s}\n\n", lowerFirst(rt.ConstantPrefix), quotedList(built.GeneratedChecks))
	fmt.Fprintf(&b, "func All%sStatuses() []%s {\n", rt.ConstantPrefix, rt.StatusType)
	fmt.Fprintf(&b, "\treturn []%s{%s}\n", rt.StatusType, goStatusList(flow, rt))
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func All%sEvents() []%s {\n", rt.ConstantPrefix, rt.EventType)
	fmt.Fprintf(&b, "\treturn []%s{%s}\n", rt.EventType, goEventList(flow, rt))
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func %sIsValidEvent(status %s, event %s) bool {\n", rt.ConstantPrefix, rt.StatusType, rt.EventType)
	b.WriteString("\tswitch status {\n")
	for _, state := range flow.States {
		fmt.Fprintf(&b, "\tcase %s%s:\n", rt.ConstantPrefix, pascal(state.ID))
		b.WriteString("\t\tswitch event {\n")
		for _, transition := range flow.Matrix.RowsFrom(state.ID) {
			if !transition.WantError {
				fmt.Fprintf(&b, "\t\tcase %s%s:\n", rt.ConstantPrefix, pascal(transition.Event))
				b.WriteString("\t\t\treturn true\n")
			}
		}
		b.WriteString("\t\t}\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("\treturn false\n")
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func %sNextStatus(status %s, event %s) %s {\n", rt.ConstantPrefix, rt.StatusType, rt.EventType, rt.StatusType)
	b.WriteString("\tswitch status {\n")
	for _, state := range flow.States {
		fmt.Fprintf(&b, "\tcase %s%s:\n", rt.ConstantPrefix, pascal(state.ID))
		b.WriteString("\t\tswitch event {\n")
		for _, transition := range flow.Matrix.RowsFrom(state.ID) {
			fmt.Fprintf(&b, "\t\tcase %s%s:\n", rt.ConstantPrefix, pascal(transition.Event))
			fmt.Fprintf(&b, "\t\t\treturn %s%s\n", rt.ConstantPrefix, pascal(transition.To))
		}
		b.WriteString("\t\t}\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("\treturn status\n")
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func Transition%sStatus(status %s, event %s) (%s, error) {\n", rt.ConstantPrefix, rt.StatusType, rt.EventType, rt.StatusType)
	fmt.Fprintf(&b, "\tnext := %sNextStatus(status, event)\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\tif !%sIsValidEvent(status, event) {\n", rt.ConstantPrefix)
	b.WriteString("\t\treturn next, fmt.Errorf(\"cannot apply %s from %s\", event, status)\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn next, nil\n")
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func %sFormalExpectedInvariants() []string {\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\treturn append([]string(nil), %sFormalInvariants...)\n", lowerFirst(rt.ConstantPrefix))
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func %sFormalExpectedGeneratedChecks() []string {\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\treturn append([]string(nil), %sFormalGeneratedChecks...)\n", lowerFirst(rt.ConstantPrefix))
	b.WriteString("}\n")
	return formatGo(b.String())
}

// renderGoReplayHelper emits generated/<folder>/replay.go, which lives
// in the same subpackage as runtime.go. It exports RunReplay so a
// hand-authored top-level _test.go can pass in a Transition closure and
// delegate the entire transition+trace replay matrix.
func renderGoReplayHelper(flow model.Flow, opts Options) (string, error) {
	if flow.Runtime.Go == nil {
		return "", fmt.Errorf("runtime.go is required for %s", flow.FlowID)
	}
	rt := flow.Runtime.Go
	pkg := layout.GeneratedDirName
	artifactFile := filepath.Base(flow.Layout.ArtifactPath)
	modulePath := opts.GoModulePath
	if modulePath == "" {
		modulePath = "{{SCENARIO_ID}}"
	}
	var b strings.Builder
	b.WriteString(generatedHeader(flow))
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString("import (\n")
	b.WriteString("\t\"testing\"\n\n")
	fmt.Fprintf(&b, "\t%q\n", "github.com/vrooli/vrooli/packages/proto/modeltest")
	b.WriteString(")\n\n")
	fmt.Fprintf(&b, "// Transition is the shape the hand-authored test must supply. The\n")
	fmt.Fprintf(&b, "// wrapper around %s should produce one of these for delegation.\n", rt.StatusType)
	fmt.Fprintf(&b, "type Transition = modeltest.Transition[%s, %s]\n\n", rt.StatusType, rt.EventType)
	b.WriteString("// RunReplay loads the canonical formal artifact, asserts it is\n")
	b.WriteString("// fresh, and replays every expanded transition and named trace\n")
	b.WriteString("// against the caller's transition.\n")
	b.WriteString("func RunReplay(t *testing.T, transition Transition) {\n")
	b.WriteString("\tt.Helper()\n")
	fmt.Fprintf(&b, "\tartifact := modeltest.LoadFormalArtifact(t, %q)\n", artifactFile)
	b.WriteString("\tmodeltest.AssertFormalArtifactFresh(t, artifact, modeltest.FormalArtifactExpectation{\n")
	fmt.Fprintf(&b, "\t\tContractPath:    %sContractPath,\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tContractSHA256:  %sContractSHA256,\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tModelPath:       %sModelPath,\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tModelSHA256:     %sModelSHA256,\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tGeneratorPath:   %sGeneratorPath,\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tGeneratorSHA256: %sGeneratorSHA256,\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tInvariants:      %sFormalExpectedInvariants(),\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tGeneratedChecks: %sFormalExpectedGeneratedChecks(),\n", rt.ConstantPrefix)
	b.WriteString("\t})\n")
	fmt.Fprintf(&b, "\tmodeltest.AssertFormalTransitionsReplay(t, artifact, All%sStatuses(), All%sEvents(), transition)\n", rt.ConstantPrefix, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\tmodeltest.AssertFormalTracesReplay(t, artifact, All%sStatuses(), All%sEvents(), transition)\n", rt.ConstantPrefix, rt.ConstantPrefix)
	b.WriteString("}\n")
	return formatGo(b.String())
}

func formatGo(source string) (string, error) {
	formatted, err := goformat.Source([]byte(source))
	if err != nil {
		return "", fmt.Errorf("format go source: %w\n--- source ---\n%s", err, source)
	}
	return string(formatted), nil
}

func goStatusList(flow model.Flow, rt *contract.GoRuntime) string {
	values := make([]string, 0, len(flow.States))
	for _, state := range flow.States {
		values = append(values, rt.ConstantPrefix+pascal(state.ID))
	}
	return strings.Join(values, ", ")
}

func goEventList(flow model.Flow, rt *contract.GoRuntime) string {
	values := make([]string, 0, len(flow.Events))
	for _, event := range flow.Events {
		values = append(values, rt.ConstantPrefix+pascal(event.ID))
	}
	return strings.Join(values, ", ")
}

// GoSubpackageImportPath returns the import path of the generated
// runtime subpackage, suitable for inclusion in a hand-authored test
// file's import list.
func GoSubpackageImportPath(flow model.Flow) string {
	return layout.SubpackageImportPath(flow.Layout)
}
