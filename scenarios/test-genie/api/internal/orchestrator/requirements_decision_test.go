package orchestrator

import (
	"strings"
	"testing"

	"test-genie/internal/orchestrator/phases"
)

// catalogDefinitions mirrors discoverPhaseDefinitions' catalog path: the full
// set of registered phases with their Optional flags, which is what the sync
// decision treats as "required coverage".
func catalogDefinitions(t *testing.T) []phases.Definition {
	t.Helper()
	specs := phases.DefaultCatalog().All()
	if len(specs) == 0 {
		t.Fatal("default catalog is empty")
	}
	defs := make([]phases.Definition, 0, len(specs))
	for _, spec := range specs {
		defs = append(defs, phases.Definition{
			Name:     spec.Name,
			Runner:   spec.Runner,
			Timeout:  spec.DefaultTimeout,
			Optional: spec.Optional,
		})
	}
	return defs
}

func passedResults(names []string) []PhaseExecutionResult {
	results := make([]PhaseExecutionResult, 0, len(names))
	for _, name := range names {
		results = append(results, PhaseExecutionResult{Name: name, Status: "passed"})
	}
	return results
}

// TestRequirementsSyncStillGatedOnCuratedPresets pins the invariant that
// promoting the business phase into the quick/smoke presets does NOT enable
// the requirements syncer (which mutates tracked files: requirements/*.json,
// PRD.md checkboxes). Sync only runs when every non-Optional catalog phase is
// present; the curated presets are deliberate subsets, so they must stay
// side-effect-free.
func TestRequirementsSyncStillGatedOnCuratedPresets(t *testing.T) {
	defs := catalogDefinitions(t)
	presets := phases.DefaultPresets()

	for _, preset := range []string{"quick", "smoke"} {
		preset := preset
		t.Run(preset, func(t *testing.T) {
			names, ok := presets[preset]
			if !ok {
				t.Fatalf("preset %q missing from DefaultPresets", preset)
			}

			// Pin WS-B itself: the business phase is part of the preset.
			found := false
			for _, n := range names {
				if n == phases.Business.String() {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("preset %q must include the business phase, got %v", preset, names)
			}

			plan := &phasePlan{Definitions: defs, PresetUsed: preset}
			for _, def := range defs {
				for _, n := range names {
					if def.Name.String() == n {
						plan.Selected = append(plan.Selected, def)
					}
				}
			}

			decision := newRequirementsSyncDecision(nil, plan, passedResults(names))
			if decision.Execute {
				t.Fatalf("requirements sync must stay gated on preset %q (reason=%q)", preset, decision.Reason)
			}
			if !strings.Contains(decision.Reason, "missing required phases") {
				t.Fatalf("expected gating reason to name missing phases, got %q", decision.Reason)
			}
		})
	}
}

// TestRequirementsSyncRunsWithFullCoverage is the counter-case proving the
// gate above is not vacuous: when every non-Optional phase passed, sync runs.
func TestRequirementsSyncRunsWithFullCoverage(t *testing.T) {
	defs := catalogDefinitions(t)
	all := make([]string, 0, len(defs))
	for _, def := range defs {
		all = append(all, def.Name.String())
	}

	plan := &phasePlan{Definitions: defs, Selected: defs, PresetUsed: "comprehensive"}
	decision := newRequirementsSyncDecision(nil, plan, passedResults(all))
	if !decision.Execute {
		t.Fatalf("sync should execute with full phase coverage, got gated: %q", decision.Reason)
	}
}
