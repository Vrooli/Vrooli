package orchestrator

import (
	"testing"

	phasespkg "test-genie/internal/orchestrator/phases"
)

// An unspecified preset must name itself comprehensive AND select the same
// phases it selected before. The second half is what keeps this an honesty fix
// rather than a behavior change: Git Control Tower's reuse predicate keys on
// the name, so the name had to become true, not the work had to change.
func TestUnspecifiedPresetIsNamedComprehensiveWithoutChangingSelection(t *testing.T) {
	defs := []phasespkg.Definition{
		{Name: phasespkg.Name("structure")},
		{Name: phasespkg.Name("unit")},
		{Name: phasespkg.Name("docs")},
	}
	presets := map[string][]string{
		"quick":         {"structure"},
		"comprehensive": {"structure", "unit", "docs"},
	}

	selected, preset, _, err := selectPhases(defs, presets, SuiteExecutionRequest{}, PhaseToggleConfig{})
	if err != nil {
		t.Fatalf("selectPhases: %v", err)
	}
	if preset != phasespkg.PresetComprehensive.String() {
		t.Fatalf("preset = %q, want comprehensive", preset)
	}
	if len(selected) != len(defs) {
		t.Fatalf("selected %d phases, want all %d — naming the preset must not change selection", len(selected), len(defs))
	}
}

// An explicit --phases request is exact user intent and is still not a preset.
func TestExplicitPhasesCarryNoPreset(t *testing.T) {
	defs := []phasespkg.Definition{
		{Name: phasespkg.Name("structure")},
		{Name: phasespkg.Name("unit")},
	}
	_, preset, _, err := selectPhases(defs, map[string][]string{}, SuiteExecutionRequest{Phases: []string{"unit"}}, PhaseToggleConfig{})
	if err != nil {
		t.Fatalf("selectPhases: %v", err)
	}
	if preset != "" {
		t.Fatalf("preset = %q, want empty for an explicit phase request", preset)
	}
}

// A planner-resolved selection narrows what runs while keeping the preset name.
// This is the adaptive-profile path: `quick` is budget-fitted in the plan
// service and the executor cannot re-derive the trim, so the set must be
// carried — but carrying it as explicit intent is what erased the preset.
func TestResolvedPhasesNarrowSelectionAndKeepPreset(t *testing.T) {
	defs := []phasespkg.Definition{
		{Name: phasespkg.Name("structure")},
		{Name: phasespkg.Name("unit")},
		{Name: phasespkg.Name("performance")},
	}
	presets := map[string][]string{"quick": {"structure", "unit", "performance"}}

	selected, preset, _, err := selectPhases(defs, presets, SuiteExecutionRequest{
		Preset:         "quick",
		ResolvedPhases: []string{"structure", "unit"},
	}, PhaseToggleConfig{})
	if err != nil {
		t.Fatalf("selectPhases: %v", err)
	}
	if preset != "quick" {
		t.Fatalf("preset = %q, want quick — a planner resolution is not an explicit request", preset)
	}
	if len(selected) != 2 {
		t.Fatalf("selected %d phases, want the 2 the profile fitted", len(selected))
	}
}

// Explicit operator intent still wins over a planner resolution.
func TestExplicitPhasesOverrideResolvedPhases(t *testing.T) {
	defs := []phasespkg.Definition{
		{Name: phasespkg.Name("structure")},
		{Name: phasespkg.Name("unit")},
	}
	selected, preset, _, err := selectPhases(defs, map[string][]string{}, SuiteExecutionRequest{
		Phases:         []string{"unit"},
		ResolvedPhases: []string{"structure", "unit"},
	}, PhaseToggleConfig{})
	if err != nil {
		t.Fatalf("selectPhases: %v", err)
	}
	if preset != "" {
		t.Fatalf("preset = %q, want empty for an explicit request", preset)
	}
	if len(selected) != 1 || selected[0].Name.Key() != "unit" {
		t.Fatalf("selected = %+v, want only unit", selected)
	}
}
