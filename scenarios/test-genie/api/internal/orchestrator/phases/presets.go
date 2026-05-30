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

// curatedPresets declares the hand-picked preset → phase compositions using
// typed phase Names so a stale or misspelled phase is a compile error rather
// than a silent runtime miss. Validate against the catalog via ValidatePresets.
//
// PresetComprehensive is intentionally absent: it is computed from the catalog
// (every registered phase) in DefaultPresets, so adding a phase auto-joins
// comprehensive and it can never silently drift from the catalog. The curated
// presets below are deliberate subsets and stay explicit.
var curatedPresets = map[Preset][]Name{
	PresetQuick: {Structure, Standards, Docs, Unit},
	PresetSmoke: {Structure, Standards, Lint, Docs, Integration},
	// architecture-audit is the per-surface conformance battery plus the
	// structural cohesion axis — the single command the screaming-
	// architecture skill points at. Excludes runtime phases (unit, smoke,
	// integration, performance). The architecture phase is advisory; the
	// campaign nudge fires from its findings.
	PresetArchitectureAudit: {
		Structure, Contracts, UIHealth, Docs, Standards, Architecture,
	},
}

// DefaultPresets returns the built-in presets as a name→phase-list map suitable
// for the orchestrator's preset merging. The returned slices are copies.
//
// The "comprehensive" preset is derived from the catalog (ValidPhaseNames) so it
// always equals the full set of registered phases — it is never a maintained
// list. A guard test (presets_comprehensive_guard_test.go) enforces this.
func DefaultPresets() map[string][]string {
	out := make(map[string][]string, len(curatedPresets)+1)
	for preset, names := range curatedPresets {
		phases := make([]string, 0, len(names))
		for _, n := range names {
			phases = append(phases, n.String())
		}
		out[preset.String()] = phases
	}
	out[PresetComprehensive.String()] = ValidPhaseNames()
	return out
}

// ValidatePresets ensures every phase referenced by a curated preset resolves in
// the supplied catalog. A non-nil catalog is required; passing nil falls back to
// the default catalog. It returns an error naming the first offending phase so
// misconfiguration fails loudly at startup rather than mid-run. The computed
// "comprehensive" preset needs no validation — it is derived from the catalog.
func ValidatePresets(catalog *Catalog) error {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	for preset, names := range curatedPresets {
		for _, n := range names {
			if _, ok := catalog.Lookup(n.String()); !ok {
				return fmt.Errorf("preset %q references unknown phase %q", preset, n)
			}
		}
	}
	return nil
}
