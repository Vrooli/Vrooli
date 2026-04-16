package domains

import (
	"reference-react-vite/cli/domains/notes"
	"reference-react-vite/cli/domains/projects"
	"reference-react-vite/cli/domains/tasks"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		tasks.Register(core),
		projects.Register(core),
		notes.Register(core),
	}
}
