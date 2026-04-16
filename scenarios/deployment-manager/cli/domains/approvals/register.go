package approvals

import (
	approvalcmd "deployment-manager/cli/approvals"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := approvalcmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Approvals",
		Commands: []cliapp.Command{
			{Name: "approvals", NeedsAPI: true, Description: "Deployment approval gate (list, get, create, decide, gate, platforms)", Run: commands.Run},
		},
	}
}
