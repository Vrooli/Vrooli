package releases

import (
	releasecmd "deployment-manager/cli/releases"

	"github.com/vrooli/cli-core/cliapp"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	commands := releasecmd.NewWithOperationClient(app.APIClient, releasesconnect.NewReleasesServiceClient(httpClient, baseURL))
	return cliapp.CommandGroup{
		Title: "Releases",
		Commands: []cliapp.Command{
			{Name: "releases", NeedsAPI: true, Description: "Governed release lifecycle (list, get, operation, dossier, health, start, verify, reconcile, recover)", Run: commands.Run},
		},
	}
}
