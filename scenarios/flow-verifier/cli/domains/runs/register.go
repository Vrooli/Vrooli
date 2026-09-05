// Package runs is the CLI's verification-run history command surface,
// a thin wrapper over the Connect-RPC RunsService. Command surface loads
// from cli/manifest.json via cliapp.LoadFromManifest.
package runs

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "runs"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RunsService.ListRuns": h.list,
		"RunsService.GetRun":   h.show,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("runs: load from manifest: %w", err)
	}
	return group, nil
}
