package tasks

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Tasks",
		Commands: []cliapp.Command{
			support.Command("task", "Manage tasks", deps.Task),
		},
	}
}
