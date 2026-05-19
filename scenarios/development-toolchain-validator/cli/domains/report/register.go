// Package report is the CLI's report command surface.
package report

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `report` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "report",
		Description: "Read-only roll-ups composed from goldens, manifests, validation records, and staleness",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "golden-summary",
				Description: "Latest verdict per skill and tool for a golden",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "slug", Required: true, Description: "Golden slug"},
					},
				},
				RunCtx: h.goldenSummary,
			},
			{
				Name:        "tuple-history",
				Description: "Paginated record history for one (skill|tool, golden) tuple",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "skill", Description: "Skill id (mutually exclusive with --tool)"},
						{Name: "tool", Description: "Tool name (mutually exclusive with --skill)"},
						{Name: "golden", Required: true, Description: "Golden slug"},
						{Name: "page-size", Default: "50"},
						{Name: "page-token"},
					},
				},
				RunCtx: h.tupleHistory,
			},
			{
				Name:        "coverage",
				Description: "Per-skill coverage grid for one golden",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "slug", Required: true, Description: "Golden slug"},
					},
				},
				RunCtx: h.coverage,
			},
		},
	}
}
