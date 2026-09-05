package domains

import (
	"secrets-manager/cli/domains/admin"
	"secrets-manager/cli/domains/backup"
	"secrets-manager/cli/domains/campaigns"
	"secrets-manager/cli/domains/credentials"
	"secrets-manager/cli/domains/deployment"
	"secrets-manager/cli/domains/descriptors"
	"secrets-manager/cli/domains/grants"
	"secrets-manager/cli/domains/keyring"
	"secrets-manager/cli/domains/overrides"
	"secrets-manager/cli/domains/overview"
	"secrets-manager/cli/domains/resources"
	"secrets-manager/cli/domains/scenarios"
	"secrets-manager/cli/domains/security"
	"secrets-manager/cli/domains/tiers"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		overview.Register(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		security.Register(core),
		deployment.Register(core),
		resources.Register(core),
		scenarios.Register(core),
		campaigns.Register(core),
		overrides.Register(core),
		admin.Register(core),
		credentials.Register(core),
		grants.Register(core),
		descriptors.Register(core),
		backup.Register(core),
		keyring.Register(core),
		tiers.Register(core),
	}
}
