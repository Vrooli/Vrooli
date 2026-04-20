package releases

import (
	releasecmd "deployment-manager/cli/releases"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := releasecmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Releases",
		Commands: []cliapp.Command{
			{Name: "releases", NeedsAPI: true, Description: "LPBS desktop release lifecycle (list, get, start, verify)", Run: commands.Run},
		},
	}
}
