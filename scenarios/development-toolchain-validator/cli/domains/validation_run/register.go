// Package validation_run is the CLI's validation command surface. Mirrors
// the API's Connect-RPC ValidationRunService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package validation_run

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validation"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ValidationRunService.Start":      h.start,
		"ValidationRunService.Get":        h.get,
		"ValidationRunService.ListActive": h.listActive,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validation_run: load from manifest: %w", err)
	}
	return group, nil
}
