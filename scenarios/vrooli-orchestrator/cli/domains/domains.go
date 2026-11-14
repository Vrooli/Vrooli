package domains

import (
	"vrooli-orchestrator/cli/domains/profiles"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The orchestrator has no flat
// top-level commands today; built-in `status` (from cli-core) handles root
// /health. Scenario-specific status lives under `profiles active`.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		profiles.Register(core),
	}
}
