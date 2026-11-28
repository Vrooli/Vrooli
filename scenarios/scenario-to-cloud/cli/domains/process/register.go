package process

import (
	"scenario-to-cloud/cli/internal/appctx"
	processcmd "scenario-to-cloud/cli/process"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Process",
		Commands: []cliapp.Command{
			{
				Name:        "process",
				NeedsAPI:    true,
				Description: "Process control (kill, restart, control, vps-action)",
				Run: func(args []string) error {
					return processcmd.Run(deps.ProcessClient, args)
				},
			},
		},
	}
}
