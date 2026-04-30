package domains

import (
	"prompt-manager/cli/actions"
	"prompt-manager/cli/agents"
	"prompt-manager/cli/discover"
	"prompt-manager/cli/experiments"
	"prompt-manager/cli/graph"
	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/members"
	"prompt-manager/cli/metadata"
	"prompt-manager/cli/search"
	"prompt-manager/cli/skills"
	"prompt-manager/cli/tags"
	"prompt-manager/cli/teams"
	"prompt-manager/cli/testing"
	"prompt-manager/cli/topics"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates prompt-manager's domain registrations behind one
// canonical bootstrap surface.
func CommandGroups(ctx appctx.Context) []cliapp.CommandGroup {
	groups := make([]cliapp.CommandGroup, 0, 14)
	groups = append(groups, skills.Commands(ctx)...)
	groups = append(groups,
		actions.Commands(ctx),
		experiments.Commands(ctx),
		tags.Commands(ctx),
		members.Commands(ctx),
		agents.Commands(ctx),
		teams.Commands(ctx),
		topics.Commands(ctx),
		testing.Commands(ctx),
		metadata.Commands(ctx),
		search.Commands(ctx),
		discover.Commands(ctx),
		graph.Commands(ctx),
	)
	return groups
}
