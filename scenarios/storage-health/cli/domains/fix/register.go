// Package fix is the CLI's fix-domain command surface. It calls the API's
// shared ScenarioValidationService Fix RPCs (PreviewFix/ApplyFix) to preview or
// apply storage-health's deterministic autofixes for a scenario.
package fix

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "fix"

// Register builds the fix subcommand group from the embedded manifest and wires
// the Connect-RPC bindings to the handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.PreviewFix": h.preview,
		"ScenarioValidationService.ApplyFix":   h.apply,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fix: load from manifest: %w", err)
	}
	return group, nil
}
