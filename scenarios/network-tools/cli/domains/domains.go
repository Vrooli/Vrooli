package domains

import (
	"network-tools/cli/domains/apidefs"
	"network-tools/cli/domains/network"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. network-tools has no flat
// commands beyond the built-in operational set provided by cli-core.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		network.Register(core),
		apidefs.Register(core),
	}
}
