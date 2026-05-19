// Package staleness is the CLI's staleness command surface. Mirrors the
// API's Connect-RPC StalenessService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package staleness

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "staleness"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"StalenessService.ListStale": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("staleness: load from manifest: %w", err)
	}
	return group, nil
}
