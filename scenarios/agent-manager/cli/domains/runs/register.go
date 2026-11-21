package runs

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Runs",
		Commands: []cliapp.Command{
			support.Command("scenario-smoke", "Run the live tracking/provenance smoke check", deps.ScenarioSmoke),
		},
	}
}

// SubcommandGroup publishes every run operation to cliapp. The handlers retain
// their established flag parsing while the command tree becomes discoverable.
func SubcommandGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "run",
		Description: "Manage and inspect run executions",
		NeedsAPI:    true,
		Subcommands: deps.RunCommands,
	}
}
