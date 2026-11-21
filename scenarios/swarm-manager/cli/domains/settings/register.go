package settings

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Scenario settings",
		Subcommands: []cliapp.Command{
			support.APICommand("get", "Get current settings", deps.SettingsGet),
			support.APICommand("update", "Update settings (--data JSON)", deps.SettingsUpdate),
		},
	}
}
