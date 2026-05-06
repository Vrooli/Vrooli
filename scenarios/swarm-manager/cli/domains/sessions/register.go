package sessions

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "sessions",
		Description: "Agent session management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List agent sessions [--kind KIND] [--status STATUS] [--active-only] [--limit N] [--json]", deps.SessionsList),
			support.APICommand("get", "Get agent session details (--id ID) [--json]", deps.SessionsGet),
			support.APICommand("delete", "Delete an agent session (--id ID --yes) [--json]", deps.SessionsDelete),
		},
	}
}
