package domains

import (
	"vrooli-assistant/cli/domains/assistant"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
// vrooli-assistant currently has no single-verb command surfaces; cli-core
// already provides the built-in `status` command that hits the root /health
// endpoint.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		assistant.Register(core),
	}
}
