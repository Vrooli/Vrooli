package domains

import (
	"campaign-content-studio/cli/domains/campaigns"
	"campaign-content-studio/cli/domains/content"
	"campaign-content-studio/cli/domains/documents"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. `generate` is a single-verb
// content-generation surface (`POST /generate`) that stays flat for the
// shortest invocation.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		content.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		campaigns.Register(core),
		documents.Register(core),
	}
}
