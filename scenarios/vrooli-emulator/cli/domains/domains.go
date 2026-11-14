package domains

import (
	"vrooli-emulator/cli/domains/metrics"
	"vrooli-emulator/cli/domains/sessions"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups currently exposes no flat commands; growth happens via SubcommandGroups.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups registers the session and metrics subcommand trees.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		sessions.Register(core),
		metrics.Register(core),
	}
}
