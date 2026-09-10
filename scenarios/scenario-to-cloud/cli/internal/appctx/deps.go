package appctx

import (
	"scenario-to-cloud/cli/bundle"
	"scenario-to-cloud/cli/credential"
	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/edge"
	"scenario-to-cloud/cli/inspect"
	"scenario-to-cloud/cli/internal/transport"
	"scenario-to-cloud/cli/manifest"
	"scenario-to-cloud/cli/operation"
	"scenario-to-cloud/cli/preflight"
	"scenario-to-cloud/cli/process"
	"scenario-to-cloud/cli/scenario"
	"scenario-to-cloud/cli/secrets"
	"scenario-to-cloud/cli/task"
	"scenario-to-cloud/cli/vps"

	"github.com/vrooli/cli-core/cliapp"
)

// Dependencies wires every command group's client over one transport.
type Dependencies struct {
	Core             *cliapp.ScenarioApp
	Transport        transport.Transport
	ManifestClient   *manifest.Client
	BundleClient     *bundle.Client
	DeploymentClient *deployment.Client
	PreflightClient  *preflight.Client
	VPSClient        *vps.Client
	InspectClient    *inspect.Client
	ProcessClient    *process.Client
	EdgeClient       *edge.Client
	SecretsClient    *secrets.Client
	ScenarioClient   *scenario.Client
	TaskClient       *task.Client
	Operations       operation.Commands
	Credentials      credential.Commands
}

// New builds the dependencies over a transport. Production passes
// transport.FromCore(core); tests pass transport.ForBaseURL(fakeServer).
func New(core *cliapp.ScenarioApp, tr transport.Transport) Dependencies {
	deploymentClient := deployment.NewClient(tr)
	return Dependencies{
		Core:             core,
		Transport:        tr,
		ManifestClient:   manifest.NewClient(tr.API),
		BundleClient:     bundle.NewClient(tr),
		DeploymentClient: deploymentClient,
		PreflightClient:  preflight.NewClient(tr),
		VPSClient:        vps.NewClient(tr.API),
		InspectClient:    inspect.NewClient(tr.API),
		ProcessClient:    process.NewClient(tr.API),
		EdgeClient:       edge.NewClient(tr),
		SecretsClient:    secrets.NewClient(tr.API),
		ScenarioClient:   scenario.NewClient(tr.API),
		TaskClient:       task.NewClient(tr.API),
		Operations:       operation.Commands{Client: deploymentClient.Operations, Deployments: deploymentClient.Deployments},
		Credentials:      credential.New(tr),
	}
}
