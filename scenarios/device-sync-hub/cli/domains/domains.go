package domains

import (
	"device-sync-hub/cli/domains/devices"
	"device-sync-hub/cli/domains/settings"
	"device-sync-hub/cli/domains/sync"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. `settings` is a read-only,
// single-verb surface exposed as `device-sync-hub settings`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		settings.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		devices.Register(core),
		sync.Register(core),
	}
}
