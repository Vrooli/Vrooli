package domains

import (
	"fall-foliage-explorer/cli/domains/foliage"
	"fall-foliage-explorer/cli/domains/regions"
	"fall-foliage-explorer/cli/domains/reports"
	"fall-foliage-explorer/cli/domains/trips"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. `regions` is a single read-only
// surface (`GET /api/regions`) so it stays flat for the shortest invocation.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		regions.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		foliage.Register(core),
		reports.Register(core),
		trips.Register(core),
	}
}
