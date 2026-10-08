package domains

import (
	"file-tools/cli/domains/analyze"
	"file-tools/cli/domains/archive"
	"file-tools/cli/domains/docs"
	"file-tools/cli/domains/files"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `docs` live here so the invocation stays `file-tools docs` instead of
// `file-tools docs show`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		docs.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		archive.Register(core),
		files.Register(core),
		analyze.Register(core),
	}
}
