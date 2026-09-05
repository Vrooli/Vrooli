package operations

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "operations",
		Description: "Operations center status and briefings",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List active operations [--window PT3H] [--status S] [--lane L] [--mode M] [--owner-type T] [--q TEXT] [--json]", deps.OperationsList),
			support.APICommand("brief", "Get the bounded current operations briefing [--window PT3H] [--json]", deps.OperationsBrief),
		},
	}
}
