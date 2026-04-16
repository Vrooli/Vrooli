package domains

import (
	"bookmark-intelligence-hub/cli/domains/actions"
	"bookmark-intelligence-hub/cli/domains/analytics"
	"bookmark-intelligence-hub/cli/domains/bookmarks"
	"bookmark-intelligence-hub/cli/domains/categories"
	"bookmark-intelligence-hub/cli/domains/platforms"
	"bookmark-intelligence-hub/cli/domains/profiles"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `categories` live here so the invocation stays
// `bookmark-intelligence-hub categories`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		categories.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		profiles.Register(core),
		bookmarks.Register(core),
		actions.Register(core),
		platforms.Register(core),
		analytics.Register(core),
	}
}
