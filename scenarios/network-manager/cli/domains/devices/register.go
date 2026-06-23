package devices

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "devices"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := handlers{core: core}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"InventoryService.RefreshInventory":      h.refresh,
		"InventoryService.ListDevices":           h.list,
		"InventoryService.UpdateDeviceGroup":     h.group,
		"InventoryService.ExplainDeviceIdentity": h.explain,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("devices: load from manifest: %w", err)
	}
	return group, nil
}
