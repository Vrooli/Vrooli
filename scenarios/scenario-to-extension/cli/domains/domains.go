package domains

import (
	"scenario-to-extension/cli/domains/extension"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The scenario-to-extension CLI
// exposes no flat commands today; all wrapped endpoints live under the `extension`
// subcommand group for consistency with the API surface (/api/v1/extension/*).
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		extension.Register(core),
	}
}
