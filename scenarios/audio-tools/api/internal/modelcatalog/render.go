package modelcatalog

import (
	"fmt"
	"strings"
)

// RenderMarkdown renders docs/reference/models.md from the loaded seed. The
// output is deterministic: seed order for prose lists, no host facts, no
// timestamps outside the seed's own generated date.
func (c *Catalog) RenderMarkdown() string {
	var b strings.Builder

	b.WriteString("# Model and voice registry\n\n")
	b.WriteString("<!-- Generated from api/internal/modelcatalog/models.seed.json. Do not edit by hand.\n")
	b.WriteString("     Regenerate from api/ with: go run ./internal/modelcatalog/cmd/render-models -->\n\n")
	fmt.Fprintf(&b, "Generated %s from `api/internal/modelcatalog/models.seed.json` ", c.seed.Generated)
	fmt.Fprintf(&b, "(schema `%s`, %d records).\n\n", c.seed.SchemaVersion, len(c.seed.Models))
	b.WriteString("Model choice is data plus evidence. This is the audio-tools analogue of image-tools' ")
	b.WriteString("`models/registry.seed.json`: one record per model or voice family, carrying stable provider ids, ")
	b.WriteString("STT engine ids, licence and commercial eligibility, readiness, the hardware envelope, and ")
	b.WriteString("benchmark evidence. Consumers ask `api/internal/modelcatalog` instead of hardcoding engine ids.\n\n")
	b.WriteString("The registry is a runtime surface, not only documentation: ")
	b.WriteString("`ModelRegistryService.ListModels` returns these records and ")
	b.WriteString("`ModelRegistryService.ResolveModel` applies the selection policy for a capability and ")
	b.WriteString("language. The CLI exposes them as `audio-tools model list` / `audio-tools model resolve`, ")
	b.WriteString("and the UI renders them on the Models page. Edit the seed, regenerate, and every surface ")
	b.WriteString("moves together.\n\n")

	b.WriteString("## How selection works\n\n")
	b.WriteString("Ranking is deterministic:\n\n")
	b.WriteString("1. readiness (`available` > `candidate` > `experimental` > `watch`)\n")
	b.WriteString("2. role signature (`default` > `fast` > `quality` > `edge`)\n")
	b.WriteString("3. commercial clarity (`yes` > `unknown` > `no`)\n")
	b.WriteString("4. model id ascending\n\n")
	b.WriteString("Only `available` records are default-eligible. A default candidate must also carry the `default` ")
	b.WriteString("role; a merely available record is an alternative, not a default. A `candidate`, ")
	b.WriteString("`experimental`, or `watch` record may carry the `default` role only with ")
	b.WriteString("`allow_default: true`, and availability still ranks first among default candidates. ")
	b.WriteString("`CommercialDefaultFor` fails closed: it returns only records whose `commercial_use` is ")
	b.WriteString("explicitly `yes`.\n\n")
	b.WriteString("In the tables, `Default` is `yes` for an available promoted default, `opt-in` for a ")
	b.WriteString("non-available record that carries the default role with `allow_default`, and `—` for a record ")
	b.WriteString("that is a valid alternative but not a promoted default.\n\n")
	b.WriteString("Entry points: `All`, `ByCapability`, `ByProvider`, `ByEngine`, `CandidatesFor`, ")
	b.WriteString("`CandidatesForEngine`, `DefaultFor`, `DefaultForEngine`, `CommercialDefaultFor`. ")
	b.WriteString("`DefaultFor(capability, language, platform)` matches language by primary subtag (`mul` matches ")
	b.WriteString("every language), filters by declared os/arch, and admits a record only when its CPU/GPU/RAM/VRAM ")
	b.WriteString("envelope fits the supplied platform. Unknown platform facts are not treated as a shortfall; a known ")
	b.WriteString("shortfall is.\n\n")

	for _, capability := range Capabilities {
		models := c.byCapability[capability]
		fmt.Fprintf(&b, "## %s\n\n", capabilityHeading(capability))
		if len(models) == 0 {
			b.WriteString("_No records._\n\n")
			continue
		}
		b.WriteString("| Model | Provider ids | Engine ids | Runtime | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Default role | Roles |\n")
		b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
		for _, m := range models {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				m.ID,
				joinOrDash(m.ProviderIDs),
				joinOrDash(m.EngineIDs),
				m.Runtime,
				formatArtifact(m.Artifact),
				m.Params,
				formatSize(m.SizeMBApprox),
				joinOrDash(m.Languages),
				m.License,
				m.Commercial,
				m.Readiness,
				defaultEligibility(m),
				joinOrDash(m.Roles),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Declared defaults\n\n")
	b.WriteString("Computed with an unconstrained platform (`DefaultFor(cap, \"\", Platform{})`), so ")
	b.WriteString("GPU-required records are excluded. The commercial column applies the fail-closed filter.\n\n")
	b.WriteString("| Capability | Default | Commercial default |\n")
	b.WriteString("|---|---|---|\n")
	for _, capability := range Capabilities {
		def, defErr := c.DefaultFor(capability, "", Platform{})
		commercial, commercialErr := c.CommercialDefaultFor(capability, "", Platform{})
		defCell := "_none_"
		if defErr == nil {
			defCell = "`" + def.ID + "`"
		}
		commercialCell := "_none_"
		if commercialErr == nil {
			commercialCell = "`" + commercial.ID + "`"
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", capability, defCell, commercialCell)
	}
	b.WriteString("\n")

	b.WriteString("## Resource footprint\n\n")
	b.WriteString("Default selection is also a resource decision. Each capability's promoted default must fit its ")
	b.WriteString("declared budget (`DefaultFootprintBudgetMB`); a seed edit that promotes a larger model fails the ")
	b.WriteString("budget gate instead of silently reserving more memory. Budgets: ")
	for i, capability := range Capabilities {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s %d MB", capability, FootprintBudget(capability))
	}
	b.WriteString(".\n\n")
	b.WriteString("| Capability | Default | Runtime | Declared size | Budget | Artifact |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, capability := range Capabilities {
		def, err := c.DefaultFor(capability, "", Platform{})
		if err != nil {
			fmt.Fprintf(&b, "| %s | _none_ | — | — | %d MB | — |\n", capability, FootprintBudget(capability))
			continue
		}
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s | %d MB | %s |\n",
			capability, def.ID, def.Runtime, formatSize(def.SizeMBApprox), FootprintBudget(capability), formatArtifact(def.Artifact))
	}
	fmt.Fprintf(&b, "\nPromoted defaults (the weight disk a fresh install of the referenced resources needs): **%s**. ",
		formatSize(c.PromotedDefaultFootprintMB()))
	fmt.Fprintf(&b, "Every available local record declared in this registry totals **%s**; that is a catalogue total, ",
		formatSize(c.DeclaredLocalFootprintMB()))
	b.WriteString("not what an install pulls: a resource installs its eager artifact and leaves on-demand tiers unpulled. ")
	b.WriteString("Remote (`byok`/`gateway`) and `browser` records carry no local weight footprint.\n\n")

	b.WriteString("## Cautions\n\n")
	b.WriteString("Records that are not both available and commercially clear. A caution is not a selection ")
	b.WriteString("ban by itself: the readiness and commercial filters already enforce the hard rules.\n\n")
	for _, m := range c.seed.Models {
		if m.Readiness == ReadinessAvailable && m.Commercial == CommercialYes {
			continue
		}
		reason := strings.TrimSpace(m.Notes)
		if m.Commercial != CommercialYes && strings.TrimSpace(m.LicenseNotes) != "" {
			reason = strings.TrimSpace(m.LicenseNotes)
		}
		fmt.Fprintf(&b, "- `%s` — readiness `%s`, commercial `%s`. %s\n", m.ID, m.Readiness, m.Commercial, reason)
	}
	b.WriteString("\n")

	b.WriteString("## Provider and engine vocabulary\n\n")
	b.WriteString("Provider ids are the stable health/lifecycle ids owned by `internal/providercatalog` (exposed through `internal/capabilities`). ")
	b.WriteString("Engine ids are STT stream-selection ids owned by `internal/sttengine`; non-STT records, ")
	b.WriteString("BYOK/gateway/browser providers, and batch adapters with no engine leave `engine_ids` empty. ")
	b.WriteString("An engine id that no provider owns is drift: the cross-catalog conformance test rejects it.\n\n")
	fmt.Fprintf(&b, "Provider ids: %s.\n\n", codeList(c.seed.KnownProviderIDs))
	fmt.Fprintf(&b, "Engine ids: %s.\n", codeList(c.EngineIDs()))

	return b.String()
}

func capabilityHeading(capability Capability) string {
	switch capability {
	case CapabilitySTT:
		return "Speech to text (stt)"
	case CapabilityTTS:
		return "Text to speech (tts)"
	case CapabilitySummarize:
		return "Summarization (summarize)"
	case CapabilitySpeaker:
		return "Speaker (speaker)"
	default:
		return string(capability)
	}
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	return strings.Join(values, ", ")
}

func formatSize(sizeMB float64) string {
	if sizeMB <= 0 {
		return "—"
	}
	return fmt.Sprintf("%g MB", sizeMB)
}

// formatArtifact renders the resource artifact identity that supplies local
// weights. The manifest owns URL/checksum; this column only names the
// declaration the conformance gate verifies.
func formatArtifact(artifact *Artifact) string {
	if artifact == nil {
		return "—"
	}
	return "`" + artifact.Resource + "/" + artifact.Name + "`"
}

func defaultEligibility(m Model) string {
	if !m.HasRole("default") {
		return "—"
	}
	if m.DefaultEligible() {
		if m.Readiness != ReadinessAvailable {
			return "opt-in"
		}
		return "yes"
	}
	return "no"
}

func codeList(values []string) string {
	if len(values) == 0 {
		return "_none_"
	}
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = "`" + value + "`"
	}
	return strings.Join(out, ", ")
}
