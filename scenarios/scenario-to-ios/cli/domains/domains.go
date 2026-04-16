package domains

import (
	"scenario-to-ios/cli/domains/build"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The iOS API surface has a
// single real endpoint (POST /api/v1/ios/build), so the `build` command lives
// at the top level rather than under a subcommand group.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		build.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups. The scenario-to-ios
// API currently exposes no multi-verb domains, so this list is empty.
func SubcommandGroups(_ *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return nil
}
