package scenario

import (
	"scenario-to-cloud/cli/internal/appctx"
	scenariocmd "scenario-to-cloud/cli/scenario"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps appctx.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Scenarios",
		Commands: []cliapp.Command{
			{
				Name:        "scenario",
				NeedsAPI:    true,
				Description: "Scenario discovery (list, ports, deps)",
				Run: func(args []string) error {
					return scenariocmd.Run(deps.ScenarioClient, args)
				},
			},
		},
	}
}
