package settings

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

// GroupName is the manifest group name this package owns.
const GroupName = "settings"

// Register builds the `settings` subcommand group for scenario-wide
// configuration (currently: session defaults) from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SettingsService.GetSessionDefaults":    h.get,
		"SettingsService.UpdateSessionDefaults": h.set,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("settings: load from manifest: %w", err)
	}
	// Preserve the pre-manifest subcommand alias (cli-manifest/v1 has no
	// per-command alias field).
	support.ApplyAliases(group.Subcommands, map[string][]string{
		"session-defaults-get": {"session-defaults"},
	})
	return group, nil
}
