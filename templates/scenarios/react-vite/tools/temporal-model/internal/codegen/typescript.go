package codegen

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
)

// renderTypeScriptRuntime emits generated/<folder>/runtime.ts: state
// and event unions, transition table, fixture-map types, freshness
// expectation, and runtime helper functions.
func renderTypeScriptRuntime(flow model.Flow, built artifact.Artifact) (string, error) {
	if flow.Runtime.TypeScript == nil {
		return "", fmt.Errorf("runtime.typescript is required for %s", flow.FlowID)
	}
	rt := flow.Runtime.TypeScript
	var b strings.Builder
	b.WriteString(generatedHeader(flow))
	fmt.Fprintf(&b, "export const %s = [\n", rt.StatusesConst)
	for _, state := range flow.States {
		fmt.Fprintf(&b, "  %q,\n", state.ID)
	}
	b.WriteString("] as const;\n\n")
	fmt.Fprintf(&b, "export type %s = (typeof %s)[number];\n\n", rt.StatusType, rt.StatusesConst)
	fmt.Fprintf(&b, "export const %s = [\n", rt.EventsConst)
	for _, event := range flow.Events {
		fmt.Fprintf(&b, "  %q,\n", event.ID)
	}
	b.WriteString("] as const;\n\n")
	fmt.Fprintf(&b, "export type %s = (typeof %s)[number];\n\n", rt.EventType, rt.EventsConst)
	if rt.StateUnionType != "" {
		renderTypeScriptRuntimeUnion(&b, rt.StateUnionType, "status", flow.States, rt.StateVariants, rt.PayloadTypes)
	}
	if rt.EventUnionType != "" {
		renderTypeScriptRuntimeUnion(&b, rt.EventUnionType, "type", flow.Events, rt.EventVariants, rt.PayloadTypes)
	}
	renderTypeScriptFixtureContract(&b, rt)
	tableConst := lowerFirst(strings.TrimSuffix(rt.StatusType, "Status")) + "TransitionTable"
	helperBase := strings.TrimSuffix(rt.StatusType, "Status")
	if helperBase == "" {
		helperBase = rt.StatusType
	}
	fmt.Fprintf(&b, "type %sTransitionRow = {\n", helperBase)
	fmt.Fprintf(&b, "  readonly to: %s;\n", rt.StatusType)
	b.WriteString("  readonly wantError: boolean;\n")
	b.WriteString("};\n\n")
	table, err := renderTypeScriptTransitionTable(flow)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "const %s = %s", tableConst, table)
	fmt.Fprintf(&b, "satisfies Record<%s, Record<%s, %sTransitionRow>>;\n\n", rt.StatusType, rt.EventType, helperBase)
	fmt.Fprintf(&b, "export const is%sEventValid = (status: %s, event: %s): boolean =>\n", helperBase, rt.StatusType, rt.EventType)
	fmt.Fprintf(&b, "  !%s[status][event].wantError;\n\n", tableConst)
	fmt.Fprintf(&b, "export const next%sStatus = (status: %s, event: %s): %s =>\n", helperBase, rt.StatusType, rt.EventType, rt.StatusType)
	fmt.Fprintf(&b, "  %s[status][event].to;\n\n", tableConst)
	fmt.Fprintf(&b, "export const transition%sStatus = (status: %s, event: %s): %s => {\n", helperBase, rt.StatusType, rt.EventType, rt.StatusType)
	fmt.Fprintf(&b, "  const row = %s[status][event];\n", tableConst)
	b.WriteString("  if (row.wantError) {\n")
	b.WriteString("    throw new Error(`cannot apply ${event} from ${status}`);\n")
	b.WriteString("  }\n")
	b.WriteString("  return row.to;\n")
	b.WriteString("};\n\n")
	fmt.Fprintf(&b, "export const %s = {\n", rt.FormalExpectationConst)
	fmt.Fprintf(&b, "  contractPath: %q,\n", built.Source.ContractPath)
	fmt.Fprintf(&b, "  contractSha256: %q,\n", built.Source.ContractSHA256)
	fmt.Fprintf(&b, "  modelPath: %q,\n", built.Source.ModelPath)
	fmt.Fprintf(&b, "  modelSha256: %q,\n", built.Source.ModelSHA256)
	fmt.Fprintf(&b, "  generatorPath: %q,\n", built.Source.GeneratorPath)
	fmt.Fprintf(&b, "  generatorSha256: %q,\n", built.Source.GeneratorSHA256)
	fmt.Fprintf(&b, "  invariants: [%s],\n", tsQuotedList(built.Invariants))
	fmt.Fprintf(&b, "  generatedChecks: [%s],\n", tsQuotedList(built.GeneratedChecks))
	b.WriteString("} as const;\n")
	return b.String(), nil
}

type typeScriptTransitionRow struct {
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

func renderTypeScriptTransitionTable(flow model.Flow) (string, error) {
	var b strings.Builder
	b.WriteString("{\n")
	for _, state := range flow.States {
		fmt.Fprintf(&b, "  %q: {\n", state.ID)
		for _, transition := range flow.Matrix.RowsFrom(state.ID) {
			row := typeScriptTransitionRow{To: transition.To, WantError: transition.WantError}
			data, err := json.Marshal(row)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "    %q: %s,\n", transition.Event, data)
		}
		b.WriteString("  },\n")
	}
	b.WriteString("} ")
	return b.String(), nil
}

// renderTypeScriptReplayHelper emits generated/<folder>/replay.helper.ts.
// It exports a single function runFormalReplay so the hand-authored
// .test.ts at the top of the feature folder can delegate the entire
// replay matrix in one call.
func renderTypeScriptReplayHelper(flow model.Flow) (string, error) {
	if flow.Runtime.TypeScript == nil {
		return "", fmt.Errorf("runtime.typescript is required for %s", flow.FlowID)
	}
	rt := flow.Runtime.TypeScript
	base := strings.TrimSuffix(rt.StatusType, "Status")
	if base == "" {
		base = rt.StatusType
	}
	lowerBase := lowerFirst(base)
	helperPath := flow.Layout.ReplayHelperPath
	artifactImport := tsRelativeImport(helperPath, flow.Layout.ArtifactPath)
	runtimeImport := tsRelativeImport(helperPath, strings.TrimSuffix(flow.Layout.RuntimePath, ".ts"))
	testUtilsImport := tsRelativeImport(helperPath, "ui/src/test-utils")
	formalNodeImport := tsRelativeImport(helperPath, "ui/src/test-utils/modeltest/formal.node")
	statusField := strings.TrimPrefix(flow.Replay.Transition.StatusAccessor, "state.")
	wrapperModule := flow.Replay.Transition.Module
	wrapperImport := tsRelativeImport(helperPath, filepath.ToSlash(filepath.Join(filepath.Dir(filepath.ToSlash(flow.ContractPath)), strings.TrimPrefix(wrapperModule, "./"))))
	wrapperImport = strings.TrimSuffix(wrapperImport, ".ts")
	var b strings.Builder
	b.WriteString(generatedHeader(flow))
	b.WriteString("import { describe, it } from \"vitest\";\n\n")
	fmt.Fprintf(&b, "import formalArtifact from %q;\n", artifactImport)
	fmt.Fprintf(&b, "import type { FormalArtifact } from %q;\n", testUtilsImport)
	b.WriteString("import {\n")
	b.WriteString("  assertFormalTransitionsReplay,\n")
	b.WriteString("  assertFormalTracesReplay,\n")
	b.WriteString("  transitionFromReplayAdapter,\n")
	fmt.Fprintf(&b, "} from %q;\n", testUtilsImport)
	fmt.Fprintf(&b, "import { assertFormalArtifactFreshFromFiles } from %q;\n", formalNodeImport)
	b.WriteString("import {\n")
	fmt.Fprintf(&b, "  %sFormalExpectation,\n", lowerBase)
	fmt.Fprintf(&b, "  %sReplayFixtureContract,\n", lowerBase)
	fmt.Fprintf(&b, "  type %sEventFixtureMap,\n", base)
	fmt.Fprintf(&b, "  type %sStateFixtureMap,\n", base)
	fmt.Fprintf(&b, "} from %q;\n", runtimeImport)
	fmt.Fprintf(&b, "import { %s } from %q;\n\n", flow.Replay.Transition.Function, wrapperImport)
	fmt.Fprintf(&b, "export interface %sFormalReplayFixtures {\n", base)
	fmt.Fprintf(&b, "  readonly stateFor: %sStateFixtureMap;\n", base)
	fmt.Fprintf(&b, "  readonly eventFor: %sEventFixtureMap;\n", base)
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "export interface RunFormalReplayOptions {\n")
	fmt.Fprintf(&b, "  readonly transition?: typeof %s;\n", flow.Replay.Transition.Function)
	fmt.Fprintf(&b, "  readonly fixtures: %sFormalReplayFixtures;\n", base)
	b.WriteString("}\n\n")
	b.WriteString("// runFormalReplay is the single entry point the hand-authored\n")
	b.WriteString("// .test.ts file must invoke at module top level. The lint pass in\n")
	b.WriteString("// tools/temporal-model rejects any test file that imports this\n")
	b.WriteString("// helper without calling it.\n")
	b.WriteString("export const runFormalReplay = (options: RunFormalReplayOptions): void => {\n")
	fmt.Fprintf(&b, "  const transitionImpl = options.transition ?? %s;\n", flow.Replay.Transition.Function)
	b.WriteString("  const transitionStatus = transitionFromReplayAdapter({\n")
	fmt.Fprintf(&b, "    states: %sReplayFixtureContract.states,\n", lowerBase)
	fmt.Fprintf(&b, "    events: %sReplayFixtureContract.events,\n", lowerBase)
	b.WriteString("    stateFor: options.fixtures.stateFor,\n")
	b.WriteString("    eventFor: options.fixtures.eventFor,\n")
	fmt.Fprintf(&b, "    statusOf: (state) => state.%s,\n", statusField)
	b.WriteString("    transition: transitionImpl,\n")
	b.WriteString("  });\n\n")
	fmt.Fprintf(&b, "  describe(%q, () => {\n", base+" formal workflow")
	b.WriteString("    it(\"replays generated formal model artifacts\", () => {\n")
	fmt.Fprintf(&b, "      assertFormalArtifactFreshFromFiles(formalArtifact as FormalArtifact, %sFormalExpectation);\n", lowerBase)
	b.WriteString("      assertFormalTransitionsReplay(\n")
	b.WriteString("        formalArtifact as FormalArtifact,\n")
	fmt.Fprintf(&b, "        %sReplayFixtureContract.states,\n", lowerBase)
	fmt.Fprintf(&b, "        %sReplayFixtureContract.events,\n", lowerBase)
	b.WriteString("        transitionStatus,\n")
	b.WriteString("      );\n")
	b.WriteString("      assertFormalTracesReplay(\n")
	b.WriteString("        formalArtifact as FormalArtifact,\n")
	fmt.Fprintf(&b, "        %sReplayFixtureContract.states,\n", lowerBase)
	fmt.Fprintf(&b, "        %sReplayFixtureContract.events,\n", lowerBase)
	b.WriteString("        transitionStatus,\n")
	b.WriteString("      );\n")
	b.WriteString("    });\n")
	b.WriteString("  });\n")
	b.WriteString("};\n")
	return b.String(), nil
}

func renderTypeScriptRuntimeUnion[T interface {
	contract.State | contract.Event
}](b *strings.Builder, typeName string, discriminator string, items []T, variants map[string]map[string]string, payloadTypes map[string]string) {
	if len(variants) == 0 {
		return
	}
	fmt.Fprintf(b, "export type %s =\n", typeName)
	for i, item := range items {
		id := itemID(item)
		prefix := "  | "
		if i == 0 {
			prefix = "  "
		}
		fmt.Fprintf(b, "%s{ readonly %s: %q", prefix, discriminator, id)
		fields := sortedKeys(variants[id])
		for _, field := range fields {
			alias := variants[id][field]
			fmt.Fprintf(b, "; readonly %s: %s", field, payloadTypes[alias])
		}
		b.WriteString(" }")
		if i == len(items)-1 {
			b.WriteString(";\n\n")
		} else {
			b.WriteString("\n")
		}
	}
}

func itemID(item any) string {
	switch value := item.(type) {
	case contract.State:
		return value.ID
	case contract.Event:
		return value.ID
	default:
		return ""
	}
}

func renderTypeScriptFixtureContract(b *strings.Builder, rt *contract.TypeScriptRuntime) {
	stateFixtureType := strings.TrimSuffix(rt.StatusType, "Status") + "StateFixtureMap"
	if rt.StateUnionType != "" {
		stateFixtureType = rt.StateUnionType + "FixtureMap"
		fmt.Fprintf(b, "export type %s<RuntimeState = %s> = Record<%s, () => RuntimeState>;\n", stateFixtureType, rt.StateUnionType, rt.StatusType)
	} else {
		fmt.Fprintf(b, "export type %s<RuntimeState> = Record<%s, () => RuntimeState>;\n", stateFixtureType, rt.StatusType)
	}
	eventFixtureType := strings.TrimSuffix(rt.EventType, "EventType") + "EventFixtureMap"
	if eventFixtureType == rt.EventType+"EventFixtureMap" {
		eventFixtureType = strings.TrimSuffix(rt.EventType, "Event") + "EventFixtureMap"
	}
	if rt.EventUnionType != "" {
		fmt.Fprintf(b, "export type %s<RuntimeEvent = %s> = Record<%s, () => RuntimeEvent>;\n\n", eventFixtureType, rt.EventUnionType, rt.EventType)
	} else {
		fmt.Fprintf(b, "export type %s<RuntimeEvent> = Record<%s, () => RuntimeEvent>;\n\n", eventFixtureType, rt.EventType)
	}
	base := strings.TrimSuffix(rt.StatusType, "Status")
	if base == "" {
		base = rt.StatusType
	}
	fmt.Fprintf(b, "export const %sReplayFixtureContract = {\n", lowerFirst(base))
	fmt.Fprintf(b, "  states: %s,\n", rt.StatusesConst)
	fmt.Fprintf(b, "  events: %s,\n", rt.EventsConst)
	fmt.Fprintf(b, "} as const satisfies { readonly states: readonly %s[]; readonly events: readonly %s[] };\n\n", rt.StatusType, rt.EventType)
}

func tsRelativeImport(fromPath string, targetPath string) string {
	fromDir := filepath.Dir(filepath.ToSlash(fromPath))
	rel, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(targetPath))
	if err != nil {
		return "./" + filepath.ToSlash(targetPath)
	}
	out := filepath.ToSlash(rel)
	if !strings.HasPrefix(out, ".") {
		out = "./" + out
	}
	return out
}
