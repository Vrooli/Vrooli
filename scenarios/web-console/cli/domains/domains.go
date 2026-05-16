package domains

import (
	"web-console/cli/domains/ai"
	"web-console/cli/domains/capabilities"
	"web-console/cli/domains/conversation"
	"web-console/cli/domains/events"
	"web-console/cli/domains/metrics"
	"web-console/cli/domains/session"
	"web-console/cli/domains/settings"
	"web-console/cli/domains/shortcuts"
	"web-console/cli/domains/terminal"
	"web-console/cli/domains/workspace"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `events`, `metrics`, and `capabilities` live here so the invocation stays
// `web-console events` instead of `web-console events list`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		events.Register(core),
		metrics.Register(core),
		capabilities.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		session.Register(core),
		terminal.Register(core),
		workspace.Register(core),
		settings.Register(core),
		shortcuts.Register(core),
		ai.Register(core),
		conversation.Register(core),
	}
}
