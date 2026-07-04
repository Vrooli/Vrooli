package orchestrator

import (
	"strings"
	"testing"

	"test-genie/internal/orchestrator/phases"
)

// catalogDefinitions mirrors discoverPhaseDefinitions: the full set of
// registered catalog phases with their Optional flags, which is what the sync
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

// TestRequirementsSyncStillGatedOnProfileSubsets pins the invariant that
// adaptive quick/smoke selections, even when they include business, do NOT
// enable the requirements syncer (which mutates tracked files:
// requirements/*.json, PRD.md checkboxes). Sync only runs when every
// non-Optional catalog phase is present; budgeted profile selections are
// deliberate subsets, so they must stay side-effect-free.
func TestRequirementsSyncStillGatedOnProfileSubsets(t *testing.T) {
	defs := catalogDefinitions(t)
	profileSubset := []string{
		phases.Structure.String(),
		phases.Docs.String(),
		phases.Business.String(),
		phases.Unit.String(),
		phases.Proto.String(),
	}

	for _, profile := range []string{"quick", "smoke"} {
		profile := profile
		t.Run(profile, func(t *testing.T) {
			plan := &phasePlan{Definitions: defs, PresetUsed: profile}
			for _, def := range defs {
				for _, n := range profileSubset {
					if def.Name.String() == n {
						plan.Selected = append(plan.Selected, def)
					}
				}
			}

			decision := newRequirementsSyncDecision(nil, plan, passedResults(profileSubset))
			if decision.Execute {
				t.Fatalf("requirements sync must stay gated on profile %q (reason=%q)", profile, decision.Reason)
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
