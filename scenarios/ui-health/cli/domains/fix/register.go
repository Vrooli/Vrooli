// Package fix is the CLI's fix-domain command surface. It drives the shared
// ScenarioValidationService PreviewFix/ApplyFix RPCs to preview or apply
// ui-health's deterministic auto-fixes for a scenario.
package fix

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this domain binds to.
const GroupName = "fix"

// Register wires the `fix` group's manifest commands to their handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.PreviewFix": h.run,
		"ScenarioValidationService.ApplyFix":   h.apply,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fix: load from manifest: %w", err)
	}
	return group, nil
}
