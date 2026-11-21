package domains

import (
	"agent-inbox/cli/domains/agent"
	"agent-inbox/cli/domains/chat"
	"agent-inbox/cli/domains/label"
	"agent-inbox/cli/domains/model"
	"agent-inbox/cli/domains/settings"
	"agent-inbox/cli/domains/usage"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Shortcuts",
			Commands: []cliapp.Command{
				{
					Name:        "list",
					NeedsAPI:    true,
					Description: "Shortcut for chat list",
					Run:         func(args []string) error { return chat.RunList(core, args) },
				},
				{
					Name:        "new",
					NeedsAPI:    true,
					Description: "Shortcut for chat create",
					Run:         func(args []string) error { return chat.RunCreate(core, args) },
				},
				{
					Name:        "open",
					NeedsAPI:    true,
					Description: "Shortcut for chat get",
					Run:         func(args []string) error { return chat.RunGet(core, args) },
				},
				{
					Name:        "labels",
					NeedsAPI:    true,
					Description: "Shortcut for label list",
					Run:         func(args []string) error { return label.RunList(core, args) },
				},
				{
					Name:        "models",
					NeedsAPI:    true,
					Description: "Shortcut for model list",
					Run:         func(args []string) error { return model.RunList(core, args) },
				},
				{
					Name:        "runs",
					NeedsAPI:    true,
					Description: "Shortcut for agent runs",
					Run:         func(args []string) error { return agent.RunRuns(core, args) },
				},
			},
		},
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		chat.Register(core),
		label.Register(core),
		model.Register(core),
		agent.Register(core),
		settings.Register(core),
		usage.Register(core),
	}
}
