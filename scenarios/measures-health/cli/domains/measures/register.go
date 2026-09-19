// Package measures is the CLI's surface for measures-health's own declared
// analytical measures (the gold-star dogfood): `measures-health measures failed`
// and `measures-health measures coverage` count scenarios that failed/passed
// measures validation in a --window, by calling the Connect-RPC MeasuresService
// — the same compute path the /measures serve registry and the search-hub
// central index use, so all three resolve identical numbers.
//
// The manifest (cli/manifest.json, group "measures") is the single source of
// truth for the command-line shape (flags, governance, RPC binding + the measure
// block); handlers live in handlers.go and are wired via the bindings map.
package measures

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "measures"

// Register builds the measures subcommand group from the embedded manifest and
// wires its Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MeasuresService.CountFailedValidations":  h.failed,
		"MeasuresService.CountValidationCoverage": h.coverage,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("measures: load from manifest: %w", err)
	}
	return group, nil
}
