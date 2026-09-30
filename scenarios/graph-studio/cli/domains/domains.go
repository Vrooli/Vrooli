package domains

import (
	"graph-studio/cli/domains/conversions"
	"graph-studio/cli/domains/graphs"
	"graph-studio/cli/domains/metrics"
	"graph-studio/cli/domains/plugins"
	"graph-studio/cli/domains/stats"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb surfaces like
// `plugins`, `metrics`, and observability commands sit at the top level so
// invocations stay short (`graph-studio stats` rather than
// `graph-studio stats show`). The root /health probe is served by cli-core's
// built-in `status` command; `stats` wraps the scenario-specific /stats and
// health-detailed endpoints.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		stats.Register(core),
		plugins.Register(core),
		metrics.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		graphs.Register(core),
		conversions.Register(core),
	}
}
