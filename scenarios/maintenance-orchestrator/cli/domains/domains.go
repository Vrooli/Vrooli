package domains

import (
	"maintenance-orchestrator/cli/domains/orchestrator"
	"maintenance-orchestrator/cli/domains/presets"
	"maintenance-orchestrator/cli/domains/scenarios"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The orchestrator only needs
// hierarchical domains for now, so this returns an empty slice.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		scenarios.Register(core),
		presets.Register(core),
		orchestrator.Register(core),
	}
}
