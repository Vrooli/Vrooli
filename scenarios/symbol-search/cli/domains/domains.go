package domains

import (
	"symbol-search/cli/domains/catalog"
	"symbol-search/cli/domains/characters"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Symbol Search exposes only
// hierarchical domains, so this returns nil.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		characters.Register(core),
		catalog.Register(core),
	}
}
