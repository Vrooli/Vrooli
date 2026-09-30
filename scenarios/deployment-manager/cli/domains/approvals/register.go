package approvals

import (
	approvalcmd "deployment-manager/cli/approvals"

	"github.com/vrooli/cli-core/cliapp"
	approvalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/approvals/approvalsv1connect"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	commands := approvalcmd.NewWithConnectClient(app.APIClient, approvalconnect.NewApprovalsServiceClient(httpClient, baseURL))
	return cliapp.CommandGroup{
		Title: "Approvals",
		Commands: []cliapp.Command{
			{Name: "approvals", NeedsAPI: true, Description: "Deployment approval gate (list, get, create, decide, gate, platforms)", Run: commands.Run},
		},
	}
}
