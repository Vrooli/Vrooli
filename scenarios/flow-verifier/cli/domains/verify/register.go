// Package verify is the CLI's verification command surface, a thin
// wrapper over the Connect-RPC VerificationsService.
package verify

import "github.com/vrooli/cli-core/cliapp"

// Register returns the `verify` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	flowFlag := cliapp.Flag{Name: "flow", Description: "Restrict to a single flow id"}
	return cliapp.SubcommandGroup{
		Name:        "verify",
		Description: "Generate and check formal-temporal-model artifacts via Quint",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "run",
				Description: "Regenerate artifacts (model.qnt, runtime, replay helper) for every flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag}},
				RunCtx:      h.run,
			},
			{
				Name:        "check",
				Description: "Verify every flow: lint + freshness + Quint check",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag}},
				RunCtx:      h.check,
			},
			{
				Name:        "show",
				Description: "Show one verification's recorded run row",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Run id"}},
				},
				RunCtx: h.show,
			},
		},
	}
}
