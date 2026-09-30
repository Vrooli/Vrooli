package domains

import (
	"test-data-generator/cli/domains/generate"
	"test-data-generator/cli/domains/types"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The single-verb `types` surface
// (`GET /api/types`) lives here so the invocation stays
// `test-data-generator types` rather than `test-data-generator types list`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		types.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
// `generate` has one subcommand per data type plus a custom-schema variant.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		generate.Register(core),
	}
}
