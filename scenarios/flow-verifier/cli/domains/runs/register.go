// Package runs is the CLI's verification-run history command surface,
// a thin wrapper over the Connect-RPC RunsService.
package runs

import "github.com/vrooli/cli-core/cliapp"

// Register returns the `runs` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "runs",
		Description: "Browse persisted verification run history",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List recent verification runs",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "flow", Description: "Restrict to a single flow id"},
						{Name: "limit", Description: "Maximum rows to return (default 50)"},
					},
				},
				RunCtx: h.list,
			},
			{
				Name:        "show",
				Description: "Show one verification run (with counterexample on failure)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Run id"}},
				},
				RunCtx: h.show,
			},
		},
	}
}
