package appctx

import (
	"scenario-to-cloud/cli/bundle"
	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/edge"
	"scenario-to-cloud/cli/inspect"
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

type Dependencies struct {
	Core             *cliapp.ScenarioApp
	ManifestClient   *manifest.Client
	BundleClient     *bundle.Client
	DeploymentClient *deployment.Client
	PreflightClient  *preflight.Client
	VPSClient        *vps.Client
	InspectClient    *inspect.Client
	ProcessClient    *process.Client
	EdgeClient       *edge.Client
	SSHClient        *ssh.Client
	SecretsClient    *secrets.Client
	ScenarioClient   *scenario.Client
	TaskClient       *task.Client
}
