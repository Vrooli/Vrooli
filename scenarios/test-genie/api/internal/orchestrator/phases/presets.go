package phases

import "fmt"

// Preset is a named validation loop. Some presets are adaptive profiles and
// some are concrete phase bundles. Preset identity is centralized here so
// callers reference typed constants instead of bare strings.
type Preset string

// ProfileDefinition describes a budgeted adaptive preset. Profiles select from
// applicable phases at plan time; they are not curated phase bundles.
type ProfileDefinition struct {
	Name          Preset
	BudgetSeconds int
	Strategy      string
}

// Canonical preset names.
const (
	PresetQuick             Preset = "quick"
	PresetSmoke             Preset = "smoke"
	PresetComprehensive     Preset = "comprehensive"
	PresetArchitectureAudit Preset = "architecture-audit"
)

const (
	ProfileStrategyBudgetFastFeedback = "budget_fast_feedback"
	ProfileStrategyBudgetSmoke        = "budget_smoke"
)

var adaptiveProfiles = map[Preset]ProfileDefinition{
	PresetQuick: {
		Name:          PresetQuick,
		BudgetSeconds: 180,
		Strategy:      ProfileStrategyBudgetFastFeedback,
	},
	PresetSmoke: {
		Name:          PresetSmoke,
		BudgetSeconds: 420,
		Strategy:      ProfileStrategyBudgetSmoke,
	},
}

// String returns the preset's wire name.
func (p Preset) String() string { return string(p) }

func AdaptiveProfile(name string) (ProfileDefinition, bool) {
	profile, ok := adaptiveProfiles[Preset(NormalizeKey(name))]
	return profile, ok
}

// curatedPresets declares concrete preset → phase compositions using
// typed phase Names so a stale or misspelled phase is a compile error rather
// than a silent runtime miss. Validate against the catalog via ValidatePresets.
//
// PresetComprehensive is intentionally absent: it is computed from the catalog
// (every registered phase) in DefaultPresets, so adding a phase auto-joins
// comprehensive and it can never silently drift from the catalog. Adaptive
// presets such as quick and smoke are intentionally absent: profileplanner
// selects them from applicable phases and measured history at plan time.
var curatedPresets = map[Preset][]Name{
	// architecture-audit is the per-surface conformance battery plus the
	// structural cohesion axis — the single command the screaming-
	// architecture skill points at. Excludes runtime phases (unit,
	// performance). The architecture phase remains advisory
	// for low-confidence authority, but high-confidence blocker findings
	// gate by default.
	PresetArchitectureAudit: {
		Structure, Contracts, UIHealth, API, Docs, Architecture, Proto,
	},
}

var freshnessRequired = []Name{Structure, Docs, Business, Unit, Proto}

// FreshnessRequired returns the global required-phase set for run-freshness
// checks (RunsService.CheckFreshness, the hygiene advisory pre-commit step).
// Freshness is deliberately independent from adaptive quick/smoke profiles:
// adaptive profiles can vary by applicability, budget, and measured history,
// while freshness needs a stable repo-wide evidence contract.
//
// This is deliberately a code-level SSOT and NOT configurable per scenario
// (operator decision): a `.vrooli/testing.json` knob would let an agent delete
// required phases to silence the freshness checker instead of running tests.
// A repo-global change, if ever needed, is a code change to freshnessRequired.
func FreshnessRequired() []string {
	out := make([]string, 0, len(freshnessRequired))
	for _, n := range freshnessRequired {
		out = append(out, n.String())
	}
	return out
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

// MergePresets applies test-genie's preset precedence in one catalog-adjacent
// place: file overrides first, testing.json overrides next, then defaults fill
// missing presets. Every phase name is normalized and filtered to allowed.
func MergePresets(defaults, fileOverrides, configOverrides map[string][]string, allowed map[string]struct{}) map[string][]string {
	presets := make(map[string][]string)
	applyPresets := func(source map[string][]string, allowDelete bool, replace bool) {
		for key, phaseNames := range source {
			name := NormalizeKey(key)
			if name == "" {
				continue
			}
			filtered := filterPresetPhases(phaseNames, allowed)
			if len(filtered) == 0 {
				if allowDelete {
					delete(presets, name)
				}
				continue
			}
			if _, exists := presets[name]; exists && !replace {
				continue
			}
			presets[name] = filtered
		}
	}
	applyPresets(fileOverrides, false, true)
	applyPresets(configOverrides, true, true)
	applyPresets(defaults, false, false)
	return presets
}

func filterPresetPhases(phaseNames []string, allowed map[string]struct{}) []string {
	if len(phaseNames) == 0 || len(allowed) == 0 {
		return nil
	}
	var filtered []string
	seen := make(map[string]struct{}, len(phaseNames))
	for _, phase := range phaseNames {
		normalized := NormalizeKey(phase)
		if normalized == "" {
			continue
		}
		if _, exists := allowed[normalized]; !exists {
			continue
		}
		if _, present := seen[normalized]; present {
			continue
		}
		seen[normalized] = struct{}{}
		filtered = append(filtered, normalized)
	}
	return filtered
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
