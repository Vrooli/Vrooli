package domains

import (
	"agent-metareasoning-manager/cli/domains/analyze"
	"agent-metareasoning-manager/cli/domains/execute"
	"agent-metareasoning-manager/cli/domains/reasoning"
	"agent-metareasoning-manager/cli/domains/workflows"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups (single-verb domains).
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		analyze.Register(core),
		execute.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		workflows.Register(core),
		reasoning.Register(core),
	}
}
