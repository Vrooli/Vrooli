package domains

import (
	"notes/cli/domains/folder"
	"notes/cli/domains/note"
	"notes/cli/domains/search"
	"notes/cli/domains/tag"
	"notes/cli/domains/template"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Notes has no single-verb flat
// domains — every domain has multiple verbs, so this returns nil.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		note.Register(core),
		folder.Register(core),
		tag.Register(core),
		template.Register(core),
		search.Register(core),
	}
}
