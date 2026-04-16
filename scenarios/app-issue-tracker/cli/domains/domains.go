package domains

import (
	"app-issue-tracker/cli/domains/agent"
	"app-issue-tracker/cli/domains/app"
	"app-issue-tracker/cli/domains/issue"
	"app-issue-tracker/cli/domains/stats"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `stats` live here so invocation stays `app-issue-tracker stats`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		stats.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		issue.Register(core),
		agent.Register(core),
		app.Register(core),
	}
}
