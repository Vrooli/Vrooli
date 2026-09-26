package operation

import (
	"scenario-to-cloud/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Operations",
		Commands: []cliapp.Command{
			{
				Name:        "operation",
				NeedsAPI:    true,
				Description: "Durable cloud operations (get, wait, resume, cancel, list)",
				Run:         deps.Operations.Run,
			},
		},
	}
}
