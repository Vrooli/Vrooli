package secrets

import (
	"scenario-to-cloud/cli/internal/appctx"
	secretscmd "scenario-to-cloud/cli/secrets"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Secrets",
		Commands: []cliapp.Command{
			{
				Name:        "secrets",
				NeedsAPI:    true,
				Description: "Secrets management (set, get, delete, verify)",
				Run: func(args []string) error {
					return secretscmd.Run(deps.SecretsClient, deps.DeploymentClient, args)
				},
			},
		},
	}
}
