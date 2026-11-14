package swaps

import (
	swapcmd "deployment-manager/cli/swaps"

	"github.com/vrooli/cli-core/cliapp"
	swapsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/swaps/swapsv1connect"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	commands := swapcmd.NewWithConnectClient(app.APIClient, swapsconnect.NewSwapsServiceClient(httpClient, baseURL))
	return cliapp.CommandGroup{
		Title: "Swaps",
		Commands: []cliapp.Command{
			{Name: "swaps", NeedsAPI: true, Description: "Dependency swap tools", Run: commands.Run},
		},
	}
}
