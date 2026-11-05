package deployments

import (
	"errors"

	bundlecmd "deployment-manager/cli/bundles"
	deploycmd "deployment-manager/cli/deployments"

	"github.com/vrooli/cli-core/cliapp"
	deploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/deployments/deploymentsv1connect"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	deployments := deploycmd.NewWithConnectClient(app.APIClient, deploymentsconnect.NewDeploymentsServiceClient(httpClient, baseURL))
	bundles := bundlecmd.New(app.APIClient)
	return cliapp.CommandGroup{
		Title: "Deployments",
		Commands: []cliapp.Command{
			{Name: "deploy", NeedsAPI: true, Description: "Deploy a profile", Run: deployments.Deploy},
			{Name: "deploy-desktop", NeedsAPI: true, Description: "Orchestrate complete bundled desktop deployment", Run: deployments.DeployDesktop},
			{Name: "deployment", NeedsAPI: true, Description: "Manage deployment records", Run: deployments.Deployment},
			{Name: "build", NeedsAPI: true, Description: "Cross-compile service binaries (retired)", Run: retired("build", "use the typed release preparation and candidate operations")},
			{Name: "logs", NeedsAPI: true, Description: "Fetch deployment logs (retired)", Run: retired("logs", "use deployment operation standing and owner receipts")},
			{Name: "validate", NeedsAPI: true, Description: "Validate deployment profile (retired)", Run: retired("validate", "use readiness review and evidence operations")},
			{Name: "estimate-cost", NeedsAPI: true, Description: "Estimate deployment costs (retired)", Run: retired("estimate-cost", "use the typed release and destination review surfaces")},
			{Name: "bundle", NeedsAPI: true, Description: "Bundle export compatibility operation", Run: bundleRun(bundles)},
		},
	}
}

func bundleRun(commands *bundlecmd.Commands) func([]string) error {
	return func(args []string) error {
		if len(args) > 0 {
			switch args[0] {
			case "assemble":
				return errors.New("deployment-manager bundle assemble is retired; use the typed release preparation and candidate operations")
			case "validate":
				return errors.New("deployment-manager bundle validate is retired; use readiness review and evidence operations")
			}
		}
		return commands.Run(args)
	}
}

func retired(command, replacement string) func([]string) error {
	return func([]string) error {
		return errors.New("deployment-manager " + command + " is retired; " + replacement)
	}
}
