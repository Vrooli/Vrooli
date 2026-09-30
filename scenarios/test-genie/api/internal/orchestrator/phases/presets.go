package phases

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

// PhasesForProfile returns the descriptor-declared phase set for a concrete
// profile. Adaptive profiles are intentionally excluded because profileplanner
// selects them from applicable phases and measured history at plan time.
func PhasesForProfile(catalog *Catalog, profile Preset) []string {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	name := NormalizeKey(profile.String())
	var out []string
	for _, spec := range catalog.All() {
		for _, member := range spec.ProfileMembership {
			if NormalizeKey(member) == name {
				out = append(out, spec.Name.String())
				break
			}
		}
	}
	return out
}

// FreshnessRequired returns the descriptor-declared phase set for run-freshness
// checks (RunsService.CheckFreshness, the hygiene advisory pre-commit step).
func FreshnessRequired() []string {
	return FreshnessRequiredForCatalog(DefaultCatalog())
}

// FreshnessRequiredForCatalog returns phases whose descriptors declare that
// they participate in freshness. Applicability filtering happens later during
// planning/runs; this projection only names the provider-owned contract.
func FreshnessRequiredForCatalog(catalog *Catalog) []string {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	var out []string
	for _, spec := range catalog.All() {
		switch spec.FreshnessRequirement {
		case "always", "when_applicable":
			out = append(out, spec.Name.String())
		}
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
	catalog := DefaultCatalog()
	out := map[string][]string{
		PresetArchitectureAudit.String(): PhasesForProfile(catalog, PresetArchitectureAudit),
		PresetComprehensive.String():     ValidPhaseNames(),
	}
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

// ValidatePresets ensures descriptor-declared concrete profiles resolve to at
// least one phase in the supplied catalog. The computed "comprehensive" preset
// needs no validation — it is derived from the catalog.
func ValidatePresets(catalog *Catalog) error {
	if catalog == nil {
		catalog = DefaultCatalog()
	}
	if len(PhasesForProfile(catalog, PresetArchitectureAudit)) == 0 {
		return ErrEmptyProfile(PresetArchitectureAudit)
	}
	return nil
}

type ErrEmptyProfile Preset

func (e ErrEmptyProfile) Error() string {
	return "preset " + Preset(e).String() + " has no descriptor-declared phases"
}
