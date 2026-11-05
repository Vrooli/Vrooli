package maintenance

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Maintenance",
		Commands: []cliapp.Command{
			support.Command("maintenance", "Maintenance operations", deps.Maintenance),
		},
	}
}
