package inventory

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "inventory"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"InventoryService.ListProviderRoles": h.roles,
		"InventoryService.SmokeProvider":     h.smoke,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("inventory: load from manifest: %w", err)
	}
	return group, nil
}
