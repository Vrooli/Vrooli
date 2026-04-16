package preflight

import (
	"scenario-to-cloud/cli/internal/appctx"
	preflightcmd "scenario-to-cloud/cli/preflight"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Preflight",
		Commands: []cliapp.Command{
			{
				Name:        "preflight",
				NeedsAPI:    true,
				Description: "Preflight checks and remediation actions (run, requirements, fix, disk tools)",
				Run: func(args []string) error {
					return preflightcmd.Run(deps.PreflightClient, args)
				},
			},
		},
	}
}
