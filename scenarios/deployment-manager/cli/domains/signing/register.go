package signing

import (
	signingcmd "deployment-manager/cli/signing"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := signingcmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Code Signing",
		Commands: []cliapp.Command{
			{Name: "signing", NeedsAPI: true, Description: "Configure code signing for deployments", Run: commands.Run},
		},
	}
}
