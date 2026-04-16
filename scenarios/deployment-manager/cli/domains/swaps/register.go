package swaps

import (
	swapcmd "deployment-manager/cli/swaps"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := swapcmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Swaps",
		Commands: []cliapp.Command{
			{Name: "swaps", NeedsAPI: true, Description: "Dependency swap tools", Run: commands.Run},
		},
	}
}
