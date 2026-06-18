package phases

import (
	"fmt"
	"strings"
	"time"
)

// docPathConvention returns the repo-relative documentation location for a
// phase. Every catalog phase has a docs/phases/<name>/README.md by convention,
// so adding a phase auto-derives its doc path with no separate mapping to keep
// in sync.
func docPathConvention(name Name) string {
	key := name.Key()
	if key == "" {
		return ""
	}
	return fmt.Sprintf("scenarios/test-genie/docs/phases/%s/README.md", key)
}

// DocPaths returns the repo-relative documentation paths for a phase. It returns
// nil for names that are not registered in the default catalog, so doc lookups
// stay in lockstep with the catalog rather than a hand-maintained map.
func DocPaths(raw string) []string {
	spec, ok := DefaultCatalog().Lookup(raw)
	if !ok || spec.Doc == "" {
		return nil
	}
	return []string{spec.Doc}
}

// RenderPhasesMarkdown renders the catalog-owned phase overview. The committed
// docs are guarded against this output so phase docs drift with the catalog, not
// with hand-maintained tables.
func RenderPhasesMarkdown(catalog *Catalog) string {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	var b strings.Builder
	b.WriteString("# Test Genie Phases\n\n")
	b.WriteString("Test Genie phases are declared in the catalog at [`api/internal/orchestrator/phases/catalog.go`](../../api/internal/orchestrator/phases/catalog.go). This document is generated from that catalog; edit the catalog when phase metadata changes.\n\n")
	b.WriteString("## Phase Summary\n\n")
	b.WriteString("| Order | Phase | Timeout | Optional | Runtime | Source | Purpose |\n")
	b.WriteString("|-------|-------|---------|----------|---------|--------|---------|\n")
	for index, spec := range catalog.All() {
		b.WriteString(fmt.Sprintf("| %d | [%s](%s) | %s | %s | %s | %s | %s |\n",
			index+1,
			displayName(spec.Name.String()),
			docLink(spec.Doc),
			formatDuration(spec.DefaultTimeout),
			yesNo(spec.Optional),
			yesNo(requiresRuntime(spec)),
			escapeMarkdown(spec.Source),
			escapeMarkdown(spec.Description),
		))
	}
	b.WriteString("\n## Static Phases\n\n")
	writePhaseList(&b, catalog, false)
	b.WriteString("\n## Runtime Phases\n\n")
	writePhaseList(&b, catalog, true)
	b.WriteString("\n## Running Phases\n\n")
	b.WriteString("```bash\n")
	b.WriteString("test-genie execute my-scenario --phases structure,unit\n")
	b.WriteString("test-genie execute my-scenario --preset comprehensive\n")
	b.WriteString("```\n\n")
	b.WriteString("## Configuration\n\n")
	b.WriteString("Per-phase overrides live in `.vrooli/testing.json` under `phases.<phase>` and are validated by [`schemas/testing.schema.json`](../../schemas/testing.schema.json).\n\n")
	b.WriteString("## Presets\n\n")
	b.WriteString("Preset membership is generated from the same catalog and documented in [Presets Reference](../reference/presets.md).\n")
	return b.String()
}

// RenderPresetsMarkdown renders the committed preset reference from catalog
// descriptors and DefaultPresets.
func RenderPresetsMarkdown(catalog *Catalog) string {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	specsByName := make(map[string]Spec)
	for _, spec := range catalog.All() {
		specsByName[spec.Name.String()] = spec
	}
	presets := DefaultPresets()
	order := []Preset{PresetQuick, PresetSmoke, PresetArchitectureAudit, PresetComprehensive}
	var b strings.Builder
	b.WriteString("# Test Presets Reference\n\n")
	b.WriteString("Test Genie presets bundle catalog phases for common validation loops. This document is generated from `api/internal/orchestrator/phases`; edit the catalog or preset declarations instead of hand-editing these tables.\n\n")
	b.WriteString("Timeout values are runtime budgets, not estimates. Runtime estimates are calculated from recent per-phase history when available.\n\n")
	b.WriteString("## Available Presets\n\n")
	for _, preset := range order {
		names, ok := presets[preset.String()]
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n\n", displayName(preset.String())))
		b.WriteString(presetPurpose(preset))
		b.WriteString("\n\n")
		b.WriteString("```bash\n")
		b.WriteString(fmt.Sprintf("test-genie execute my-scenario --preset %s\n", preset))
		b.WriteString("```\n\n")
		b.WriteString("| Phase | Description | Timeout |\n")
		b.WriteString("|-------|-------------|---------|\n")
		for _, name := range names {
			spec, ok := specsByName[name]
			if !ok {
				continue
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				displayName(name),
				escapeMarkdown(spec.Description),
				formatDuration(spec.DefaultTimeout),
			))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Preset Comparison\n\n")
	b.WriteString("| Phase | Quick | Smoke | Architecture Audit | Comprehensive |\n")
	b.WriteString("|-------|-------|-------|--------------------|---------------|\n")
	for _, spec := range catalog.All() {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			displayName(spec.Name.String()),
			markPreset(presets, PresetQuick, spec.Name.String()),
			markPreset(presets, PresetSmoke, spec.Name.String()),
			markPreset(presets, PresetArchitectureAudit, spec.Name.String()),
			markPreset(presets, PresetComprehensive, spec.Name.String()),
		))
	}
	b.WriteString("\n## Custom Presets\n\n")
	b.WriteString("Define custom presets in `.vrooli/testing.json`:\n\n")
	b.WriteString("```json\n")
	b.WriteString("{\n")
	b.WriteString("  \"presets\": {\n")
	b.WriteString("    \"ci-fast\": [\"structure\", \"unit\"],\n")
	b.WriteString("    \"nightly\": [\"structure\", \"dependencies\", \"unit\", \"integration\", \"business\", \"performance\"]\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	b.WriteString("```\n")
	return b.String()
}

func writePhaseList(b *strings.Builder, catalog *Catalog, runtime bool) {
	wrote := false
	for _, spec := range catalog.All() {
		if requiresRuntime(spec) != runtime {
			continue
		}
		wrote = true
		b.WriteString(fmt.Sprintf("- [%s](%s) - %s\n", displayName(spec.Name.String()), docLink(spec.Doc), escapeMarkdown(spec.Description)))
	}
	if !wrote {
		b.WriteString("- None\n")
	}
}

func requiresRuntime(spec Spec) bool {
	caps := spec.Capabilities
	return caps.NeedsAPI || caps.NeedsUI || caps.MutatesLifecycle || caps.LifecycleDecisionDeferred
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%ds", int(d/time.Second))
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func displayName(raw string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		switch strings.ToLower(part) {
		case "api", "bas", "cli", "docs", "json", "ui":
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func docLink(path string) string {
	const prefix = "scenarios/test-genie/docs/phases/"
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

func presetPurpose(p Preset) string {
	switch p {
	case PresetQuick:
		return "Fast sanity check during development."
	case PresetSmoke:
		return "Core validation before pushing or handing off changes."
	case PresetArchitectureAudit:
		return "Surface conformance and architectural shape without runtime-heavy phases."
	case PresetComprehensive:
		return "Full validation before release or deployment."
	default:
		return "Custom validation bundle."
	}
}

func markPreset(presets map[string][]string, preset Preset, phase string) string {
	for _, name := range presets[preset.String()] {
		if name == phase {
			return "Yes"
		}
	}
	return "No"
}
