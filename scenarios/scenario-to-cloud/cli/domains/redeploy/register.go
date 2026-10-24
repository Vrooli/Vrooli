package redeploy

import (
	"scenario-to-cloud/cli/internal/appctx"
	redeploycmd "scenario-to-cloud/cli/redeploy"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Redeploy",
		Commands: []cliapp.Command{
			{
				Name:        "redeploy",
				NeedsAPI:    true,
				Description: "Create/update and execute deployment in one command",
				Run: func(args []string) error {
					return redeploycmd.Run(deps.DeploymentClient, args)
				},
			},
		},
	}
}
