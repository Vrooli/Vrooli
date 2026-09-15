package overview

import (
	overviewcmd "deployment-manager/cli/overview"

	"github.com/vrooli/cli-core/cliapp"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/dependencies/dependenciesv1connect"
	fitnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/fitness/fitnessv1connect"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	commands := overviewcmd.NewWithConnectClients(app.APIClient, dependenciesconnect.NewDependenciesServiceClient(httpClient, baseURL), fitnessconnect.NewFitnessServiceClient(httpClient, baseURL))
	return cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			{Name: "analyze", NeedsAPI: true, Description: "Analyze scenario dependencies", Run: commands.Analyze},
			{Name: "fitness", NeedsAPI: true, Description: "Calculate platform fitness scores", Run: commands.Fitness},
		},
	}
}
