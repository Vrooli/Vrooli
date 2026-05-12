// Package adoptions is the CLI's adoption-registry surface. Mirrors
// the API's Connect-RPC AdoptionsService (proto schema at
// packages/proto/schemas/react-component-library/v1/adoptions).
package adoptions

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `adoptions` subcommand group. Handlers close
// over `core` for API request + output rendering.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "adoptions",
		Description: "Track which scenarios have adopted library components and detect drift",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List adoption records",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "component-id", Description: "Only adoptions for this component id"},
						{Name: "scenario", Description: "Only adoptions targeting this scenario"},
						{Name: "limit", Description: "Maximum number of rows (default 200)"},
					},
				},
				RunCtx: h.list,
			},
			{
				Name:        "create",
				Description: "Create an adoption record",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "component-id", Required: true, Description: "Library component id"},
						{Name: "scenario", Required: true, Description: "Target scenario name (e.g. swarm-manager)"},
						{Name: "adopted-path", Required: true, Description: "Path within the target scenario"},
					},
					Flags: []cliapp.Flag{
						{Name: "adopted-version", Description: "Library version stamped on the adopted copy"},
					},
				},
				RunCtx: h.create,
			},
			{
				Name:        "delete",
				Description: "Delete an adoption record by id",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Adoption record id"},
					},
				},
				RunCtx: h.delete,
			},
			{
				Name:        "refresh",
				Description: "Recompute drift status for every adoption (or one component's)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "component-id", Description: "Limit refresh to one component's adoptions"},
					},
				},
				RunCtx: h.refresh,
			},
		},
	}
}
