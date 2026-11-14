package credential

import (
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Credentials",
		Commands: []cliapp.Command{
			{
				Name:        "credential",
				NeedsAPI:    true,
				Description: "Deployment credential bindings (list, rotate, revoke, recover, rotation get/resume)",
				Run:         deps.Credentials.Run,
			},
		},
	}
}
