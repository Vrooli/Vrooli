package drills

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "drills"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RecoveryDrillsService.PreviewDrill": h.preview,
		"RecoveryDrillsService.RunDrill":     h.run,
		"RecoveryDrillsService.GetDrill":     h.get,
		"RecoveryDrillsService.ListDrills":   h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("drills: load from manifest: %w", err)
	}
	return group, nil
}
