// Package runs is the CLI's runs-domain command surface. Mirrors the API's
// Connect-RPC RunsService. Operators use trigger/get/list/status/browse to manage
// and inspect backup run executions.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, positionals, governance, RPC bindings); this package only wires
// bindings to handlers in handlers.go.
package runs

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "runs"

// Register builds the runs subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RunsService.TriggerRun":       h.trigger,
		"RunsService.GetRun":           h.get,
		"RunsService.ListRuns":         h.list,
		"RunsService.ListTargetStatus": h.status,
		"RunsService.BrowseSnapshot":   h.browse,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("runs: load from manifest: %w", err)
	}
	return group, nil
}
