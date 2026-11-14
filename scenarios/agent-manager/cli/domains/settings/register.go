package settings

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Settings",
		Commands: []cliapp.Command{
			support.Command("settings", "Manage settings", deps.Settings),
		},
	}
}
