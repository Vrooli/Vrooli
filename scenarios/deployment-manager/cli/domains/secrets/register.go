package secrets

import (
	profilecmd "deployment-manager/cli/profiles"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := profilecmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Secrets",
		Commands: []cliapp.Command{
			{Name: "secrets", NeedsAPI: true, Description: "Secret discovery and templates", Run: commands.Secrets},
		},
	}
}
