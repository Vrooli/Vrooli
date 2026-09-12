// DOC: docs/reference/cli-commands.md
// Package domains aggregates the CLI subcommand groups exposed by
// git-control-tower. When adding or removing a group here, update
// docs/reference/cli-commands.md in the same change.
//
// The manifest catalogs the complete observable command tree. The `worktree`
// domain is also constructed from it via cliapp.LoadFromManifest; the remaining
// domains are explicitly local, REST-backed registrations until their Connect
// migrations can make the manifest drive their handlers as well.
package domains

import (
	"git-control-tower/cli/domains/audit"
	"git-control-tower/cli/domains/baseline"
	"git-control-tower/cli/domains/branch"
	"git-control-tower/cli/domains/repo"
	"git-control-tower/cli/domains/review"
	"git-control-tower/cli/domains/worktree"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp, manifestBytes []byte) ([]cliapp.SubcommandGroup, error) {
	wt, err := worktree.Register(core, manifestBytes)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{
		repo.Register(core),
		branch.Register(core),
		wt,
		review.Register(core),
		audit.Register(core),
		baseline.Register(core),
	}, nil
}
