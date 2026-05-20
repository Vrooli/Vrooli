// Package validate is the CLI's validate-domain command surface. Mirrors
// the API's Connect-RPC ValidationService. Phase 1 wires the surface
// end-to-end and surfaces the server's Unimplemented response so the
// proto/Connect/manifest chain is verified.
package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "validate"

// Register builds the validate subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ValidationService.ValidateScenario": h.validateScenario,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	return group, nil
}
