// Package scenarios is the CLI's scenario-index command surface, a
// thin wrapper over the Connect-RPC ScenariosService. Command surface
// loads from cli/manifest.json via cliapp.LoadFromManifest.
package scenarios

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "scenarios"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ScenariosService.ListScenarios": h.list,
		"ScenariosService.GetScenario":   h.show,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("scenarios: load from manifest: %w", err)
	}
	return group, nil
}
