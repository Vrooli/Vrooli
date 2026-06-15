package baseline

// Surfaces are presentation views over a SINGLE comprehensive test-genie run.
// A baseline captures one durable run (all phases) and pins it once; each
// surface is just a phase-set lens over that run's per-phase results. This file
// is the SSOT for that {surface → phase-set} grouping, used by both Create
// (to label each surface's pointer) and Diff (to bucket the empty-phase
// CompareRuns PhaseDiff[] back into surfaces locally — option-c).

// surfacePhases maps each phase-backed surface to the test-genie phases it
// aggregates. The visuals surface is intentionally absent: it is not a
// phase-set view but a run-artifact view (screenshots produced by the smoke
// phase under the baseline capture profile), diffed at the metadata level.
var surfacePhases = map[string][]string{
	SurfaceStructure: {"structure"},
	SurfaceRules:     {"standards"},
	SurfaceTests:     {"unit", "integration", "smoke"},
	SurfaceWorkflows: {"playbooks"},
}

// phaseSurface is the inverse index: phase → owning surface. Built once from
// surfacePhases so Diff can bucket a CompareRuns PhaseDiff back to its surface
// in O(1). A phase with no registered surface is dropped (it contributes to no
// surface view).
var phaseSurface = func() map[string]string {
	m := map[string]string{}
	for surface, phases := range surfacePhases {
		for _, p := range phases {
			m[p] = surface
		}
	}
	return m
}()
