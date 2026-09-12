package portfolio

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "portfolio",
		Description: "Portfolio planning briefs",
		Subcommands: []cliapp.Command{
			support.APICommand("brief", "Get the bounded portfolio startup brief [--json]", deps.PortfolioBrief),
		},
	}
}
