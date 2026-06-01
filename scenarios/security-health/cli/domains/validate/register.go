// Package validate is the CLI's validate-domain command surface. It mirrors
// the API's Connect-RPC ValidationService and is the seam test-genie's
// `security` phase shells: `security-health validate scenario <name> --json`.
package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "validate"

// Register builds the validate subcommand group from the embedded manifest and
// wires the Connect-RPC binding to the handler in handlers.go.
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
