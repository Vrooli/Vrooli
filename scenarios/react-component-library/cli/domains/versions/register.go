// Package versions is the CLI's version-history surface. Mirrors the
// API's Connect-RPC VersionsService (proto schema at
// packages/proto/schemas/react-component-library/v1/versions).
package versions

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `versions` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "versions",
		Description: "List, fetch, and diff recorded versions of a component",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List recorded versions for a component",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "component-id", Required: true, Description: "Library component id"},
					},
					Flags: []cliapp.Flag{
						{Name: "limit", Description: "Maximum number of rows (default 100)"},
					},
				},
				RunCtx: h.list,
			},
			{
				Name:        "show",
				Description: "Show a specific recorded version (with optional content)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "component-id", Required: true, Description: "Library component id"},
						{Name: "version", Required: true, Description: "Recorded @version value"},
					},
					Flags: []cliapp.Flag{
						{Name: "with-content", Description: "Include the full content body in the output"},
					},
				},
				RunCtx: h.show,
			},
			{
				Name:        "diff",
				Description: "Diff two versions (or a version and an adoption:<id>)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "component-id", Required: true, Description: "Library component id"},
						{Name: "from", Required: true, Description: "Source — @version value or adoption:<id>"},
						{Name: "to", Required: true, Description: "Target — @version value or adoption:<id>"},
					},
				},
				RunCtx: h.diff,
			},
		},
	}
}
