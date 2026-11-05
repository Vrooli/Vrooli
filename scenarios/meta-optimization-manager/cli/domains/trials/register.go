// Package trials is the CLI's trials-domain command surface. It mirrors the
// API's TrialsService. The manifest (cli/manifest.json) carries the declarative
// command shape; handlers.go builds each typed request and renders the response.
// `trials run` is a write/non-run-eligible command — the empirical gate is
// explicit-invocation only.
package trials

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "trials"

// Register builds the trials subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"TrialsService.ListTrialTasks":  h.list,
		"TrialsService.RunTrials":       h.run,
		"TrialsService.GetTrialHistory": h.history,
		"TrialsService.GetTrialRun":     h.show,
		"TrialsService.GetGateCoverage": h.coverage,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("trials: load from manifest: %w", err)
	}
	return group, nil
}
