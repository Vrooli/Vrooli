package domains

import (
	"scenario-to-cloud/cli/bundle"
	"scenario-to-cloud/cli/deployment"
	bundledomain "scenario-to-cloud/cli/domains/bundle"
	deploymentdomain "scenario-to-cloud/cli/domains/deployment"
	edgedomain "scenario-to-cloud/cli/domains/edge"
	inspectdomain "scenario-to-cloud/cli/domains/inspect"
	manifestdomain "scenario-to-cloud/cli/domains/manifest"
	preflightdomain "scenario-to-cloud/cli/domains/preflight"
	processdomain "scenario-to-cloud/cli/domains/process"
	redeploydomain "scenario-to-cloud/cli/domains/redeploy"
	scenariodomain "scenario-to-cloud/cli/domains/scenario"
	secretsdomain "scenario-to-cloud/cli/domains/secrets"
	sshdomain "scenario-to-cloud/cli/domains/ssh"
	taskdomain "scenario-to-cloud/cli/domains/task"
	vpsdomain "scenario-to-cloud/cli/domains/vps"
	"scenario-to-cloud/cli/edge"
	"scenario-to-cloud/cli/inspect"
	"scenario-to-cloud/cli/internal/appctx"
	"scenario-to-cloud/cli/manifest"
	"scenario-to-cloud/cli/preflight"
	"scenario-to-cloud/cli/process"
	"scenario-to-cloud/cli/scenario"
	"scenario-to-cloud/cli/secrets"
	"scenario-to-cloud/cli/ssh"
	"scenario-to-cloud/cli/task"
	"scenario-to-cloud/cli/vps"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	deps := appctx.Dependencies{
		Core:             core,
		ManifestClient:   manifest.NewClient(core.APIClient),
		BundleClient:     bundle.NewClient(core.APIClient),
		DeploymentClient: deployment.NewClient(core.APIClient),
		PreflightClient:  preflight.NewClient(core.APIClient),
		VPSClient:        vps.NewClient(core.APIClient),
		InspectClient:    inspect.NewClient(core.APIClient),
		ProcessClient:    process.NewClient(core.APIClient),
		EdgeClient:       edge.NewClient(core.APIClient),
		SSHClient:        ssh.NewClient(core.APIClient),
		SecretsClient:    secrets.NewClient(core.APIClient),
		ScenarioClient:   scenario.NewClient(core.APIClient),
		TaskClient:       task.NewClient(core.APIClient),
	}

	return []cliapp.CommandGroup{
		manifestdomain.Register(deps),
		bundledomain.Register(deps),
		deploymentdomain.Register(deps),
		redeploydomain.Register(deps),
		preflightdomain.Register(deps),
		vpsdomain.Register(deps),
		inspectdomain.Register(deps),
		processdomain.Register(deps),
		edgedomain.Register(deps),
		sshdomain.Register(deps),
		secretsdomain.Register(deps),
		scenariodomain.Register(deps),
		taskdomain.Register(deps),
	}
}
