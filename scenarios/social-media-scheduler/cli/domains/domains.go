package domains

import (
	"social-media-scheduler/cli/domains/accounts"
	"social-media-scheduler/cli/domains/analytics"
	"social-media-scheduler/cli/domains/auth"
	"social-media-scheduler/cli/domains/bulk"
	"social-media-scheduler/cli/domains/campaigns"
	"social-media-scheduler/cli/domains/media"
	"social-media-scheduler/cli/domains/oauth"
	"social-media-scheduler/cli/domains/platforms"
	"social-media-scheduler/cli/domains/posts"
	"social-media-scheduler/cli/domains/queue"
	"social-media-scheduler/cli/domains/user"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups for single-verb surfaces
// (e.g. `platforms`, `accounts`, `queue`).
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		platforms.Register(core),
		accounts.Register(core),
		queue.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		auth.Register(core),
		posts.Register(core),
		bulk.Register(core),
		campaigns.Register(core),
		analytics.Register(core),
		media.Register(core),
		oauth.Register(core),
		user.Register(core),
	}
}
