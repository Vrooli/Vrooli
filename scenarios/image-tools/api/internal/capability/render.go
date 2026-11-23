package capability

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// RenderTS renders the generated TypeScript capability catalog consumed by the
// UI. It is deterministic (manifest order), so the drift test compares the file
// byte-for-byte and `make capability-gen` is the only way to change it.
//
// The catalog deliberately carries only facts the UI can act on without new
// copy: family (icon selection), mechanism, ports, and the canonical summary
// (used as a fallback description when a hand-written presentation entry is
// absent). Human labels still come from i18n; this closes the "uncatalogued op
// renders as a bare name with a generic icon" gap without duplicating strings.
func RenderTS() (string, error) {
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : api/internal/capability/manifests.seed.json\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/capability/render_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// One entry per operation across both operation registries. The UI uses it to\n")
	b.WriteString("// pick a family icon and a fallback description for operations that do not yet\n")
	b.WriteString("// have a hand-written presentation entry.\n\n")

	b.WriteString("export type CapabilityEngine = \"ai\" | \"analysis\" | \"compute\" | \"unwired\";\n\n")
	b.WriteString("export interface ExtentStageMeta {\n")
	b.WriteString("  readonly id: string;\n")
	b.WriteString("  readonly kind: string;\n")
	b.WriteString("  readonly labelKey: string;\n")
	b.WriteString("  readonly descKey: string;\n")
	b.WriteString("  readonly replaces?: string;\n")
	b.WriteString("  readonly strengthParam?: string;\n")
	b.WriteString("  readonly option?: ExtentOptionMeta;\n")
	b.WriteString("}\n\n")
	b.WriteString("/** A stage's choice of how it runs (skin tone toward the source or toward the body). */\n")
	b.WriteString("export interface ExtentOptionMeta {\n")
	b.WriteString("  readonly param: string;\n")
	b.WriteString("  readonly default: string;\n")
	b.WriteString("  readonly values: readonly { readonly id: string; readonly labelKey: string; readonly descKey: string }[];\n")
	b.WriteString("}\n\n")
	b.WriteString("export interface ExtentPresetMeta {\n")
	b.WriteString("  readonly id: string;\n")
	b.WriteString("  readonly labelKey: string;\n")
	b.WriteString("  readonly descKey: string;\n")
	b.WriteString("  readonly stages: readonly string[];\n")
	b.WriteString("}\n\n")
	b.WriteString("/** How much of a person an operation carries over: its stages and the presets composed from them. */\n")
	b.WriteString("export interface ExtentMeta {\n")
	b.WriteString("  readonly stages: readonly ExtentStageMeta[];\n")
	b.WriteString("  readonly presets: readonly ExtentPresetMeta[];\n")
	b.WriteString("  readonly defaultPreset: string;\n")
	b.WriteString("}\n\n")
	b.WriteString("export interface CapabilityMeta {\n")
	b.WriteString("  readonly id: string;\n")
	b.WriteString("  readonly engine: CapabilityEngine;\n")
	b.WriteString("  readonly family: string;\n")
	b.WriteString("  readonly mechanism: string;\n")
	b.WriteString("  readonly inputKinds: readonly string[];\n")
	b.WriteString("  readonly outputKind: string;\n")
	b.WriteString("  readonly summary: string;\n")
	b.WriteString("  readonly icon?: string;\n")
	b.WriteString("  readonly labelKey?: string;\n")
	b.WriteString("  readonly descKey?: string;\n")
	b.WriteString("  readonly panel?: string;\n")
	b.WriteString("  readonly extent?: ExtentMeta;\n")
	b.WriteString("}\n\n")

	b.WriteString("export const CAPABILITIES: Readonly<Record<string, CapabilityMeta>> = {\n")
	for _, m := range registry.manifests {
		kinds := make([]string, 0, len(m.IO.Inputs))
		for _, in := range m.IO.Inputs {
			kinds = append(kinds, strconv.Quote(in.Kind))
		}
		pres := ""
		if m.Panel != "" {
			pres = fmt.Sprintf(", panel: %s", strconv.Quote(m.Panel))
		}
		if m.Presentation != nil {
			pres += fmt.Sprintf(", icon: %s, labelKey: %s, descKey: %s",
				strconv.Quote(m.Presentation.Icon),
				strconv.Quote(m.Presentation.LabelKey),
				strconv.Quote(m.Presentation.DescKey))

		}
		pres += renderExtentTS(m.Extent)
		fmt.Fprintf(&b,
			"  %s: { id: %s, engine: %s, family: %s, mechanism: %s, inputKinds: [%s], outputKind: %s, summary: %s%s },\n",
			strconv.Quote(m.ID), strconv.Quote(m.ID), strconv.Quote(string(m.Engine)),
			strconv.Quote(string(m.Family)), strconv.Quote(string(m.Mechanism)),
			strings.Join(kinds, ", "), strconv.Quote(m.IO.Output.Kind), strconv.Quote(m.Summary), pres,
		)
	}
	b.WriteString("};\n\n")
	b.WriteString("export const capabilityIds: readonly string[] = Object.keys(CAPABILITIES);\n\n")
	b.WriteString("export const capabilityMeta = (id: string): CapabilityMeta | undefined => CAPABILITIES[id];\n")
	return b.String(), nil
}

// renderExtentTS renders an operation's swap-extent declaration as the TS object
// literal the workspace builds its stage toggles and presets from.
func renderExtentTS(e *ExtentSpec) string {
	if e == nil {
		return ""
	}
	q := strconv.Quote
	stages := make([]string, 0, len(e.Stages))
	for _, st := range e.Stages {
		extra := ""
		if st.Replaces != "" {
			extra += ", replaces: " + q(st.Replaces)
		}
		if st.StrengthParam != "" {
			extra += ", strengthParam: " + q(st.StrengthParam)
		}
		if st.Option != "" {
			names := make([]string, 0, len(st.Variants))
			for name := range st.Variants {
				names = append(names, name)
			}
			slices.Sort(names)
			values := make([]string, 0, len(names))
			for _, name := range names {
				v := st.Variants[name]
				values = append(values, fmt.Sprintf("{ id: %s, labelKey: %s, descKey: %s }", q(name), q(v.LabelKey), q(v.DescKey)))
			}
			extra += fmt.Sprintf(", option: { param: %s, default: %s, values: [%s] }", q(st.Option), q(st.DefaultVariant), strings.Join(values, ", "))
		}
		stages = append(stages, fmt.Sprintf("{ id: %s, kind: %s, labelKey: %s, descKey: %s%s }", q(st.ID), q(st.Kind), q(st.LabelKey), q(st.DescKey), extra))
	}
	presets := make([]string, 0, len(e.Presets))
	for _, p := range e.Presets {
		ids := make([]string, 0, len(p.Stages))
		for _, id := range p.Stages {
			ids = append(ids, q(id))
		}
		presets = append(presets, fmt.Sprintf("{ id: %s, labelKey: %s, descKey: %s, stages: [%s] }", q(p.ID), q(p.LabelKey), q(p.DescKey), strings.Join(ids, ", ")))
	}
	return fmt.Sprintf(", extent: { stages: [%s], presets: [%s], defaultPreset: %s }", strings.Join(stages, ", "), strings.Join(presets, ", "), q(e.DefaultPreset))
}

// RenderCLI renders the generated Go command table for the CLI's `ai` group.
// It emits data only (a slice of submitSpec); the command construction, flag
// definitions, and submit handler stay hand-written in the CLI package. Adding
// an AI operation therefore becomes a manifest edit rather than a register.go
// edit, and the drift test fails the build if the two disagree.
func RenderCLI() (string, error) {
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : api/internal/capability/manifests.seed.json\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/capability/render_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// Data only: the submit command builder and flag definitions live in register.go.\n\n")
	b.WriteString("package ai\n\n")
	b.WriteString("var generatedSubmitSpecs = []submitSpec{\n")
	for _, m := range registry.CLICommands() {
		needsInput, needsMask, needsReference := false, false, false
		for _, in := range m.IO.Inputs {
			switch in.Kind {
			case "image":
				if in.Name == ReferencePort {
					needsReference = true
					continue
				}
				needsInput = true
			case "mask":
				needsMask = true
			}
		}
		flags := make([]string, 0, len(m.CLI.Flags))
		for _, f := range m.CLI.Flags {
			flags = append(flags, strconv.Quote(f))
		}
		fmt.Fprintf(&b,
			"\t{Verb: %s, Op: %s, Desc: %s, NeedsInput: %t, NeedsMask: %t, NeedsReference: %t, Flags: []string{%s}},\n",
			strconv.Quote(m.CLI.Verb), strconv.Quote(m.ID), strconv.Quote(m.Summary),
			needsInput, needsMask, needsReference, strings.Join(flags, ", "),
		)
	}
	b.WriteString("}\n")
	return b.String(), nil
}
