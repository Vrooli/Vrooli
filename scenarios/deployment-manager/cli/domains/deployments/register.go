package deployments

import (
	bundlecmd "deployment-manager/cli/bundles"
	deploycmd "deployment-manager/cli/deployments"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	deployments := deploycmd.New(app.APIClient)
	bundles := bundlecmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Deployments",
		Commands: []cliapp.Command{
			{Name: "deploy", NeedsAPI: true, Description: "Deploy a profile", Run: deployments.Deploy},
			{Name: "deploy-desktop", NeedsAPI: true, Description: "Orchestrate complete bundled desktop deployment", Run: deployments.DeployDesktop},
			{Name: "deployment", NeedsAPI: true, Description: "Manage deployment records", Run: deployments.Deployment},
			{Name: "build", NeedsAPI: true, Description: "Cross-compile service binaries", Run: deployments.Build},
			{Name: "logs", NeedsAPI: true, Description: "Fetch deployment logs", Run: deployments.Logs},
			{Name: "validate", NeedsAPI: true, Description: "Validate deployment profile", Run: deployments.Validate},
			{Name: "estimate-cost", NeedsAPI: true, Description: "Estimate deployment costs", Run: deployments.EstimateCost},
			{Name: "bundle", NeedsAPI: true, Description: "Bundle manifest operations (assemble, export, validate)", Run: bundles.Run},
		},
	}
}
