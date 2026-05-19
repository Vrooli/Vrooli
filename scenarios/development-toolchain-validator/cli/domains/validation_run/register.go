// Package validation_run is the CLI's validation command surface.
package validation_run

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `validation` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "validation",
		Description: "Start and inspect async (skill|tool, golden) validation runs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "start",
				Description: "Queue a new validation run (returns immediately; use --wait to poll for terminal)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "skill", Description: "Skill id (mutually exclusive with --tool)"},
						{Name: "tool", Description: "Tool name (mutually exclusive with --skill)"},
						{Name: "golden", Required: true, Description: "Golden slug"},
						{Name: "force", Bool: true, Description: "Force a new run even if a recent one exists"},
						{Name: "wait", Bool: true, Description: "Poll until the run reaches terminal status"},
						{Name: "wait-timeout", Default: "300", Description: "Seconds to poll when --wait is set"},
					},
				},
				RunCtx: h.start,
			},
			{
				Name:        "get",
				Description: "Show one validation run by id",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Run id (UUID)"},
					},
				},
				RunCtx: h.get,
			},
			{
				Name:        "list-active",
				Description: "List runs whose status is not terminal",
				RunCtx:      h.listActive,
			},
		},
	}
}
