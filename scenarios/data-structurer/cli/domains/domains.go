package domains

import (
	"data-structurer/cli/domains/data"
	"data-structurer/cli/domains/jobs"
	"data-structurer/cli/domains/process"
	"data-structurer/cli/domains/schemas"
	"data-structurer/cli/domains/templates"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like `data`
// live here so the invocation stays `data-structurer data <schema-id>` instead of
// `data-structurer data list <schema-id>`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		data.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		schemas.Register(core),
		templates.Register(core),
		process.Register(core),
		jobs.Register(core),
	}
}
