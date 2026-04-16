package domains

import (
	"code-smell/cli/domains/smells"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Every code-smell command is a
// single verb against `/api/v1/code-smell/*`, so they all live in one group to
// keep invocations as `code-smell <verb>` rather than `code-smell smells <verb>`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		smells.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
// code-smell exposes only flat commands today; subcommand groups remain empty
// until a verb set emerges that warrants grouping.
func SubcommandGroups(_ *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return nil
}
