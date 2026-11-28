// Package selfidentity is the single source of truth for the scenario name
// this orchestrator process embodies. test-genie tests other scenarios, but it
// can also be pointed at itself; when that happens the runnability gate must
// recognize the target as "self" and refuse any lifecycle mutation that would
// terminate the live process running the suite.
//
// Every call site that needs to ask "is the scenario under test me?" compares
// against ScenarioName here rather than hard-coding a "test-genie" literal, so
// a rename is a one-line change and the self-host guard can never silently
// drift away from the real identity.
package selfidentity

// ScenarioName is the canonical lifecycle/scenario slug for this binary. It
// matches main.go's preflight ScenarioName and the lifecycle registration
// under ~/.vrooli/processes/scenarios/<name>/.
const ScenarioName = "test-genie"

// Is reports whether the provided scenario name refers to this orchestrator's
// own scenario. Comparison is exact against the canonical slug; callers that
// accept arbitrary user input should normalize (trim/lowercase) first.
func Is(scenario string) bool {
	return scenario == ScenarioName
}
