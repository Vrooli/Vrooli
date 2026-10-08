package domains

import (
	"data-tools/cli/domains/data"
	"data-tools/cli/domains/docs"
	"data-tools/cli/domains/execution"
	"data-tools/cli/domains/resource"
	"data-tools/cli/domains/stream"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb surfaces like
// `docs` live here so invocation stays `data-tools docs` instead of
// `data-tools docs show`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		docs.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		resource.Register(core),
		execution.Register(core),
		data.Register(core),
		stream.Register(core),
	}
}
