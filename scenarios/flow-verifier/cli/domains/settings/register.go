// Package settings is the CLI's UI/CLI-preferences command surface,
// a thin wrapper over the Connect-RPC SettingsService. Command surface
// loads from cli/manifest.json via cliapp.LoadFromManifest.
package settings

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "settings"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SettingsService.GetSettings":    h.get,
		"SettingsService.UpdateSettings": h.set,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("settings: load from manifest: %w", err)
	}
	return group, nil
}
