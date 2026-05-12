// Package settings is the CLI's UI/CLI-preferences command surface,
// a thin wrapper over the Connect-RPC SettingsService.
package settings

import "github.com/vrooli/cli-core/cliapp"

// Register returns the `settings` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Get or update UI/CLI preferences (theme, font scale, density, …)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "get",
				Description: "Print the local principal's UI/CLI preferences",
				RunCtx:      h.get,
			},
			{
				Name:        "set",
				Description: "Update one or more preferences via <key>=<value> pairs",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "pair", Required: true, Repeated: true, Description: "<key>=<value> pair (repeatable)"},
					},
				},
				RunCtx: h.set,
			},
		},
	}
}
