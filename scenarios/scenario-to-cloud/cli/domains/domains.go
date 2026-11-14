package domains

import (
	bundledomain "scenario-to-cloud/cli/domains/bundle"
	credentialdomain "scenario-to-cloud/cli/domains/credential"
	deploymentdomain "scenario-to-cloud/cli/domains/deployment"
	edgedomain "scenario-to-cloud/cli/domains/edge"
	inspectdomain "scenario-to-cloud/cli/domains/inspect"
	manifestdomain "scenario-to-cloud/cli/domains/manifest"
	operationdomain "scenario-to-cloud/cli/domains/operation"
	preflightdomain "scenario-to-cloud/cli/domains/preflight"
	processdomain "scenario-to-cloud/cli/domains/process"
	publicationdomain "scenario-to-cloud/cli/domains/publication"
	redeploydomain "scenario-to-cloud/cli/domains/redeploy"
	scenariodomain "scenario-to-cloud/cli/domains/scenario"
	secretsdomain "scenario-to-cloud/cli/domains/secrets"
	taskdomain "scenario-to-cloud/cli/domains/task"
	vpsdomain "scenario-to-cloud/cli/domains/vps"
	"scenario-to-cloud/cli/internal/appctx"
	"scenario-to-cloud/cli/internal/transport"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups registers every command group over the scenario app's
// transport.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return Groups(appctx.New(core, transport.FromCore(core)))
}

// Groups registers the command groups over already-built dependencies.
func Groups(deps appctx.Dependencies) []cliapp.CommandGroup {
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
		secretsdomain.Register(deps),
		scenariodomain.Register(deps),
		taskdomain.Register(deps),
		operationdomain.Register(deps),
		credentialdomain.Register(deps),
		publicationdomain.Register(deps),
	}
}
