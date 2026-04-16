package runners

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Runners",
		Commands: []cliapp.Command{
			support.Command("runner", "Manage agent runners", deps.Runner),
		},
	}
}
