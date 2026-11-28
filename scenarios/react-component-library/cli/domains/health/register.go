// Package health is the CLI's typed library-health read composition. Mirrors
// the API's Connect-RPC LibraryHealthService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifestPrimitives.
package health

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "health"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"LibraryHealthService.GetLibraryHealth": cliapp.ProtoList(h.showCall, h.showReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("health: load from manifest: %w", err)
	}
	return group, nil
}
