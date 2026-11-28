package profiles

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Profiles",
		Commands: []cliapp.Command{
			support.Command("profile", "Manage agent profiles", deps.Profile),
		},
	}
}
