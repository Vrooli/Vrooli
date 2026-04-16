package domains

import (
	"palette-gen/cli/domains/analyze"
	"palette-gen/cli/domains/palette"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. palette-gen has no single-verb
// top-level domains; all commands are grouped under hierarchical subcommand
// groups, so this returns nil.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		palette.Register(core),
		analyze.Register(core),
	}
}
