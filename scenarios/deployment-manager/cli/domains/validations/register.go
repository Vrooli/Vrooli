package validations

import (
	validationcmd "deployment-manager/cli/validations"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := validationcmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Validations",
		Commands: []cliapp.Command{
			{Name: "validations", NeedsAPI: true, Description: "Visual validation quality gate (run, status, video, review, list)", Run: commands.Run},
		},
	}
}
