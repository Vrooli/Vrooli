package codegen

import (
	"fmt"
	goformat "go/format"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
)

func renderGoDeclarations(flow model.Flow, built artifact.Artifact) (string, error) {
	if flow.Runtime.Go == nil {
		return "", fmt.Errorf("runtime.go is required for %s", flow.FlowID)
	}
	rt := flow.Runtime.Go
	var b strings.Builder
	b.WriteString(generatedHeader(flow))
	b.WriteString("package ")
	b.WriteString(rt.Package)
	b.WriteString("\n\n")
	b.WriteString("import \"fmt\"\n\n")
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

func renderGoReplayTest(flow model.Flow) (string, error) {
	if flow.Runtime.Go == nil {
		return "", fmt.Errorf("runtime.go is required for %s", flow.FlowID)
	}
	rt := flow.Runtime.Go
	var b strings.Builder
	b.WriteString(generatedHeader(flow))
	fmt.Fprintf(&b, "package %s_test\n\n", rt.Package)
	b.WriteString("import (\n")
	b.WriteString("\t\"testing\"\n\n")
	fmt.Fprintf(&b, "\t%q\n", goRuntimeImportPath(flow))
	b.WriteString("\t\"{{SCENARIO_ID}}/internal/testutil/modeltest\"\n")
	b.WriteString(")\n\n")
	fmt.Fprintf(&b, "func Test%sFormalReplay_ReplaysGeneratedModelArtifacts(t *testing.T) {\n", rt.ConstantPrefix)
	fmt.Fprintf(&b, "\tartifact := modeltest.LoadFormalArtifact(t, %q)\n", filepath.Base(flow.Outputs.ArtifactPath))
	fmt.Fprintf(&b, "\tassert%sFormalReplay(t, artifact)\n", rt.ConstantPrefix)
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func assert%sFormalReplay(t *testing.T, artifact modeltest.FormalArtifact) {\n", rt.ConstantPrefix)
	b.WriteString("\tt.Helper()\n\n")
	b.WriteString("\tmodeltest.AssertFormalArtifactFresh(t, artifact, modeltest.FormalArtifactExpectation{\n")
	fmt.Fprintf(&b, "\t\tContractPath:    %s.%sContractPath,\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tContractSHA256:  %s.%sContractSHA256,\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tModelPath:       %s.%sModelPath,\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tModelSHA256:     %s.%sModelSHA256,\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tGeneratorPath:   %s.%sGeneratorPath,\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tGeneratorSHA256: %s.%sGeneratorSHA256,\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tInvariants:      %s.%sFormalExpectedInvariants(),\n", rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\t\tGeneratedChecks: %s.%sFormalExpectedGeneratedChecks(),\n", rt.Package, rt.ConstantPrefix)
	b.WriteString("\t})\n\n")
	fmt.Fprintf(&b, "\ttransition := func(status %s.%s, event %s.%s) (%s.%s, error) {\n", rt.Package, rt.StatusType, rt.Package, rt.EventType, rt.Package, rt.StatusType)
	fmt.Fprintf(&b, "\t\tnext, err := %s.%s(%s.%s{%s: status}, event)\n", rt.Package, flow.Replay.Transition.Function, rt.Package, flow.Replay.Transition.StateType, flow.Replay.Transition.StatusField)
	fmt.Fprintf(&b, "\t\treturn next.%s, err\n", flow.Replay.Transition.StatusField)
	b.WriteString("\t}\n")
	fmt.Fprintf(&b, "\tmodeltest.AssertFormalTransitionsReplay(t, artifact, %s.All%sStatuses(), %s.All%sEvents(), transition)\n", rt.Package, rt.ConstantPrefix, rt.Package, rt.ConstantPrefix)
	fmt.Fprintf(&b, "\tmodeltest.AssertFormalTracesReplay(t, artifact, %s.All%sStatuses(), %s.All%sEvents(), transition)\n", rt.Package, rt.ConstantPrefix, rt.Package, rt.ConstantPrefix)
	b.WriteString("}\n")
	return formatGo(b.String())
}

func formatGo(source string) (string, error) {
	formatted, err := goformat.Source([]byte(source))
	if err != nil {
		return "", err
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

func goRuntimeImportPath(flow model.Flow) string {
	dir := filepath.ToSlash(filepath.Dir(flow.Outputs.DeclarationsPath))
	dir = strings.TrimPrefix(dir, "api/")
	return "{{SCENARIO_ID}}/" + dir
}
