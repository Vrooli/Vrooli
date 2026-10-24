// Package selfidentity is the single source of truth for the scenario name
// this orchestrator process embodies. agent-manager runs coding agents against
// other scenarios, but it can also be pointed at itself (a run whose task scope
// is under scenarios/agent-manager). When a promote tries to quiesce/drain the
// agent-manager scenario from inside an agent-manager run, draining would wait
// on the very run requesting the drain — a self-deadlock. The promote-quiesce
// self-guard recognizes that case by comparing against ScenarioName here rather
// than hard-coding an "agent-manager" literal, so a rename is a one-line change
// and the guard can never silently drift away from the real identity.
//
// This mirrors test-genie's selfidentity package (the runnability self-host
// guard); Baseline Modes P6 ports the same pattern into agent-manager because
// the promote drain is the second place the kernel can deadlock on itself.
package selfidentity

// ScenarioName is the canonical lifecycle/scenario slug for this binary. It
// matches main.go's preflight ScenarioName and the lifecycle registration
// under ~/.vrooli/processes/scenarios/<name>/.
const ScenarioName = "agent-manager"

// Is reports whether the provided scenario name refers to this orchestrator's
// own scenario. Comparison is exact against the canonical slug; callers that
// accept arbitrary user input should normalize (trim/lowercase) first.
func Is(scenario string) bool {
	return scenario == ScenarioName
}
