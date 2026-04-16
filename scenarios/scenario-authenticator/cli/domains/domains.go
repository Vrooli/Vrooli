package domains

import (
	"scenario-authenticator/cli/domains/apikey"
	"scenario-authenticator/cli/domains/application"
	"scenario-authenticator/cli/domains/oauth"
	"scenario-authenticator/cli/domains/session"
	"scenario-authenticator/cli/domains/token"
	"scenario-authenticator/cli/domains/twofa"
	"scenario-authenticator/cli/domains/user"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Scenario-authenticator has no
// single-verb flat commands beyond what cli-core already registers (status,
// configure, --version, --help).
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		user.Register(core),
		session.Register(core),
		token.Register(core),
		apikey.Register(core),
		application.Register(core),
		oauth.Register(core),
		twofa.Register(core),
	}
}
