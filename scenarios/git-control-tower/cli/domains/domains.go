// DOC: docs/reference/cli-commands.md
// Package domains aggregates the CLI subcommand groups exposed by
// git-control-tower. When adding or removing a group here, update
// docs/reference/cli-commands.md in the same change.
package domains

import (
	"git-control-tower/cli/domains/audit"
	"git-control-tower/cli/domains/branch"
	"git-control-tower/cli/domains/repo"
	"git-control-tower/cli/domains/review"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		repo.Register(core),
		branch.Register(core),
		review.Register(core),
		audit.Register(core),
	}
}
