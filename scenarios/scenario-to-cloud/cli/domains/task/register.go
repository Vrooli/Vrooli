package task

import (
	"scenario-to-cloud/cli/internal/appctx"
	taskcmd "scenario-to-cloud/cli/task"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Tasks",
		Commands: []cliapp.Command{
			{
				Name:        "task",
				NeedsAPI:    true,
				Description: "AI task management (create, list, get, stop, agent-status)",
				Run: func(args []string) error {
					return taskcmd.Run(deps.TaskClient, args)
				},
			},
		},
	}
}
