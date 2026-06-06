package domains

import (
	"scenario-dependency-analyzer/cli/domains/analyze"
	"scenario-dependency-analyzer/cli/domains/bundle"
	"scenario-dependency-analyzer/cli/domains/coreset"
	"scenario-dependency-analyzer/cli/domains/cycles"
	"scenario-dependency-analyzer/cli/domains/dag"
	"scenario-dependency-analyzer/cli/domains/dependencies"
	"scenario-dependency-analyzer/cli/domains/deployment"
	"scenario-dependency-analyzer/cli/domains/graph"
	"scenario-dependency-analyzer/cli/domains/impact"
	"scenario-dependency-analyzer/cli/domains/optimize"
	"scenario-dependency-analyzer/cli/domains/proposals"
	"scenario-dependency-analyzer/cli/domains/scan"
	"scenario-dependency-analyzer/cli/domains/scenarios"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The root /health probe is
// served by cli-core's built-in `status` command, so no status/health/system
// domain is registered here.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		analyze.Register(core),
		scan.Register(core),
		coreset.Register(core),
		graph.Register(core),
		cycles.Register(core),
		impact.Register(core),
		proposals.Register(core),
		optimize.Register(core),
		dependencies.Register(core),
		deployment.Register(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		dag.Register(core),
		scenarios.Register(core),
		bundle.Register(core),
	}
}
