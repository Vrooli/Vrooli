package overview

import (
	overviewcmd "deployment-manager/cli/overview"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := overviewcmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			{Name: "analyze", NeedsAPI: true, Description: "Analyze scenario dependencies", Run: commands.Analyze},
			{Name: "fitness", NeedsAPI: true, Description: "Calculate platform fitness scores", Run: commands.Fitness},
		},
	}
}
