// Package validate is the CLI's measures coverage-validation command surface.
// It mirrors the API's Connect-RPC validation surface: `measures-health validate
// scenario <name>` calls the shared ScenarioValidationService, and
// `measures-health validate coverage` rolls the fleet up through measures-health's
// native ValidationService.
//
// The manifest (cli/manifest.json, group "validate") is the single source of
// truth for the command-line shape (flags, positionals, governance, RPC
// binding); handlers live in handlers.go and are wired via the bindings map.
package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "validate"

// Register builds the validate subcommand group from the embedded manifest and
// wires its Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validateScenario,
		"ValidationService.ListFleetCoverage":        h.coverage,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	return group, nil
}
