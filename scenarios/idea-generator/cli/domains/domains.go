package domains

import (
	"idea-generator/cli/domains/campaign"
	"idea-generator/cli/domains/document"
	"idea-generator/cli/domains/idea"
	"idea-generator/cli/domains/workflow"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Read-only single-verb surfaces
// like `workflows` live here so invocations stay short.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		workflow.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		campaign.Register(core),
		idea.Register(core),
		document.Register(core),
	}
}
