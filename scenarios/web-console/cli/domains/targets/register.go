// Package targets exposes the safe, server-owned target catalog used to route
// new terminal sessions to local or Bridge-backed nodes.
package targets

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "target"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"TargetCatalogService.List":   h.list,
		"TargetCatalogService.Get":    h.get,
		"TargetCatalogService.Doctor": h.doctor,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("target: load from manifest: %w", err)
	}
	return group, nil
}
