// Package scenarios is the CLI's scenario-index command surface, a
// thin wrapper over the Connect-RPC ScenariosService.
package scenarios

import "github.com/vrooli/cli-core/cliapp"

// Register returns the `scenarios` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "Browse the scenario index discovered under the Vrooli root",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List every scenario discovered under the Vrooli root",
				RunCtx:      h.list,
			},
			{
				Name:        "show",
				Description: "Show one scenario with its discovered flows",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Scenario id"}},
				},
				RunCtx: h.show,
			},
		},
	}
}
