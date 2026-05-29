package phases

import "fmt"

// Preset is a named bundle of phases for a common use case (fast feedback,
// pre-push, full validation). Preset identity is centralized here so callers
// reference typed constants instead of bare strings.
type Preset string

// Canonical preset names.
const (
	PresetQuick             Preset = "quick"
	PresetSmoke             Preset = "smoke"
	PresetComprehensive     Preset = "comprehensive"
	PresetArchitectureAudit Preset = "architecture-audit"
)

// String returns the preset's wire name.
func (p Preset) String() string { return string(p) }

// defaultPresets declares the built-in preset → phase composition using typed
// phase Names so a stale or misspelled phase is a compile error rather than a
// silent runtime miss. Validate against the catalog via ValidatePresets.
var defaultPresets = map[Preset][]Name{
	PresetQuick: {Structure, Standards, Docs, Unit},
	PresetSmoke: {Structure, Standards, Lint, Docs, Integration},
	PresetComprehensive: {
		Structure, Contracts, Standards, Dependencies, Lint, Docs,
		Performance, Smoke, Unit, Integration, Playbooks, Business,
	},
	// architecture-audit is the per-surface conformance battery plus the
	// structural cohesion axis — the single command the screaming-
	// architecture skill points at. Excludes runtime phases (unit, smoke,
	// integration, performance). The architecture phase is advisory; the
	// migration nudge fires from its findings.
	PresetArchitectureAudit: {
		Structure, Contracts, UIHealth, Docs, Standards, Architecture,
	},
}

// DefaultPresets returns the built-in presets as a name→phase-list map suitable
// for the orchestrator's preset merging. The returned slices are copies.
func DefaultPresets() map[string][]string {
	out := make(map[string][]string, len(defaultPresets))
	for preset, names := range defaultPresets {
		phases := make([]string, 0, len(names))
		for _, n := range names {
			phases = append(phases, n.String())
		}
		out[preset.String()] = phases
	}
	return out
}

// ValidatePresets ensures every phase referenced by a built-in preset resolves
// in the supplied catalog. A non-nil catalog is required; passing nil falls back
// to the default catalog. It returns an error naming the first offending phase
// so misconfiguration fails loudly at startup rather than mid-run.
func ValidatePresets(catalog *Catalog) error {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	for preset, names := range defaultPresets {
		for _, n := range names {
			if _, ok := catalog.Lookup(n.String()); !ok {
				return fmt.Errorf("preset %q references unknown phase %q", preset, n)
			}
		}
	}
	return nil
}
