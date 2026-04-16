package domains

import (
	"document-manager/cli/domains/agents"
	"document-manager/cli/domains/applications"
	"document-manager/cli/domains/index"
	"document-manager/cli/domains/queue"
	"document-manager/cli/domains/search"
	"document-manager/cli/domains/system"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like `search`
// and `index` live here so the invocation stays `document-manager search ...`
// instead of `document-manager search query ...`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		search.Register(core),
		index.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		applications.Register(core),
		agents.Register(core),
		queue.Register(core),
		system.Register(core),
	}
}
