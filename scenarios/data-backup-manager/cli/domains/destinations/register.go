// Package destinations is the CLI's destinations-domain command surface. Mirrors
// the API's Connect-RPC DestinationsService. Operators use create/get/list/update/delete
// to manage backup destinations (kopia repositories), and usage to inspect storage.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, positionals, governance, RPC bindings); this package only wires
// bindings to handlers in handlers.go.
package destinations

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "destinations"

// Register builds the destinations subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DestinationsService.CreateDestination":             h.create,
		"DestinationsService.GetDestination":                h.get,
		"DestinationsService.ListDestinations":              h.list,
		"DestinationsService.UpdateDestination":             h.update,
		"DestinationsService.DeleteDestination":             h.delete,
		"DestinationsService.GetDestinationUsage":           h.usage,
		"DestinationsService.AnalyzeDestination":            h.readiness,
		"DestinationsService.PlanDestinationPreparation":    h.preparePlan,
		"DestinationsService.ExecuteDestinationPreparation": h.prepareExecute,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("destinations: load from manifest: %w", err)
	}
	return group, nil
}
