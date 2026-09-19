package inspect

import (
	inspectcmd "scenario-to-cloud/cli/inspect"
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Inspect",
		Commands: []cliapp.Command{
			{
				Name:        "inspect",
				NeedsAPI:    true,
				Description: "Inspection operations (plan, status, live, drift, logs, files, metrics)",
				Run: func(args []string) error {
					return inspectcmd.Run(deps.InspectClient, args)
				},
			},
		},
	}
}
