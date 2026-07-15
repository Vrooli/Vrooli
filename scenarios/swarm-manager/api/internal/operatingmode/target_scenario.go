package operatingmode

import (
	"context"
	"fmt"
	"strings"
)

// Stable prompt alias for the scenario target read. Its logical id and capability
// source are authored in the spec-sync mode's input_contract.
const ReadScenarioName = "SCENARIO_NAME"

// scenarioTargetAdapter is the target adapter for a plain scenario repository
// workspace, identified by scenario name. Unlike the backlog-item, initiative,
// and plan-execution adapters it does not resolve from any store or Plan Manager:
// a scenario is a directory that already exists on disk (the archive flow
// validated its path before queueing), so the ref alone fully identifies it. The
// deliverable is the scenario's own spec artifacts (PRD/README/docs), which the
// agent reads and syncs in place.
type scenarioTargetAdapter struct{}

func (scenarioTargetAdapter) Kind() TargetKind { return TargetScenario }

func (scenarioTargetAdapter) Resolve(_ context.Context, _ *Service, def Definition, _ PhaseDefinition, ref string) (TargetInstance, error) {
	name := strings.TrimSpace(ref)
	if name == "" {
		return TargetInstance{}, fmt.Errorf("mode %q targets a scenario: a scenario name is required", def.Mode)
	}
	return TargetInstance{
		Kind:  TargetScenario,
		ID:    name,
		Title: name,
		// Write-scope the spec-sync run to the scenario's own directory, exactly as
		// the legacy spawn's ScopePath=ScenarioPath did. Unlike plan-execution (whose
		// scope must be looked up on the owning backlog item, hence a resolver seam),
		// a scenario name fully determines its repo-relative directory
		// (scenarios/<name>/), so the adapter projects the acceptance scope directly
		// with no external resolver — and it is never zero, so the spawn can never
		// fall back to repo-wide. That fail-safe is load-bearing: the archive flow
		// RemoveAll's the scenario directory on completion, so an unconstrained agent
		// editing outside the scenario would be unsafe.
		Containment: ContainmentScope{AcceptanceAllow: []string{scenarioContainmentGlob(name)}},
	}, nil
}

// scenarioContainmentGlob is the repo-relative acceptance glob that scopes a
// scenario run to its own directory. It mirrors the convention the rest of the
// system uses for scenario acceptance scope (pathutil.ScenariosFromGlobs reads
// exactly this shape back out).
func scenarioContainmentGlob(name string) string {
	return "scenarios/" + name + "/**"
}

func (scenarioTargetAdapter) Values(t TargetInstance) map[string]any {
	return map[string]any{
		"target.scenario_name": t.Title,
	}
}

// OwnershipKey gives scenario targets a distinct lock namespace so a spec-sync
// run never collides with a backlog item or initiative of the same name.
func (scenarioTargetAdapter) OwnershipKey(id string) string {
	return "scenario--" + sanitizeOwnershipToken(id)
}
