package domains

import (
	"deployment-manager/cli/domains/approvals"
	"deployment-manager/cli/domains/deployments"
	"deployment-manager/cli/domains/overview"
	"deployment-manager/cli/domains/profiles"
	"deployment-manager/cli/domains/releases"
	"deployment-manager/cli/domains/secrets"
	"deployment-manager/cli/domains/signing"
	"deployment-manager/cli/domains/swaps"
	"deployment-manager/cli/domains/validations"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(app *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		overview.Register(app),
		profiles.Register(app),
		swaps.Register(app),
		deployments.Register(app),
		secrets.Register(app),
		signing.Register(app),
		validations.Register(app),
		approvals.Register(app),
		releases.Register(app),
	}
}
