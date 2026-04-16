package domains

import (
	"react-component-library/cli/domains/adoption"
	"react-component-library/cli/domains/ai"
	"react-component-library/cli/domains/component"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. This scenario has none; all
// domains use the hierarchical SubcommandGroup layout.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		component.Register(core),
		adoption.Register(core),
		ai.Register(core),
	}
}
