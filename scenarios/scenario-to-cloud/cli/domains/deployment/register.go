package deployment

import (
	deploymentcmd "scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Deployments",
		Commands: []cliapp.Command{
			{
				Name:        "deployment",
				NeedsAPI:    true,
				Description: "Deployment lifecycle (plan, create, execute, start/stop, health, history)",
				Run: func(args []string) error {
					return deploymentcmd.Run(deps.DeploymentClient, args)
				},
			},
		},
	}
}
