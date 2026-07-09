package autofiler

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "autofiler",
		Description: "Backlog auto-filer operations",
		Subcommands: []cliapp.Command{
			support.APICommand("status", "Show backlog auto-filer policy and latest cycle status [--json]", deps.AutoFilerStatus),
			support.APICommand("run-now", "Run one governed auto-filer cycle immediately [--json]", deps.AutoFilerRunNow),
		},
	}
}
