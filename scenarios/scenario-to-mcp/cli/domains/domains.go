package domains

import (
	"scenario-to-mcp/cli/domains/docs"
	"scenario-to-mcp/cli/domains/mcp"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. There are no flat commands in
// this CLI today; domains each expose their own subcommand group.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		mcp.Register(core),
		docs.Register(core),
	}
}
