package publication

import (
	deploymentcmd "scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Publication",
		Commands: []cliapp.Command{
			{
				Name:        "publication",
				NeedsAPI:    true,
				Description: "Governed publication bound to an approved review (request, apply, status)",
				Run: func(args []string) error {
					return deploymentcmd.RunPublication(deps.DeploymentClient, args)
				},
			},
		},
	}
}
