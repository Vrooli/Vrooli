// Package components is the CLI's component-registry surface. Mirrors
// the API's Connect-RPC ComponentsService (proto schema at
// packages/proto/schemas/react-component-library/v1/components). Handlers
// call the generated Connect-Go client; --json output is the proto wire
// shape, identical to what `curl /…/ComponentsService/ListComponents`
// returns.
package components

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `components` subcommand group. Handlers close
// over `core` for API request + output rendering.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "components",
		Description: "Browse and re-index the local React component registry",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "index",
				Description: "Walk the source root and refresh the registry",
				RunCtx:      h.index,
			},
			{
				Name:        "list",
				Description: "List indexed components",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "match", Description: "Case-insensitive substring filter"},
						{Name: "tag", Description: "Exact tag filter"},
						{Name: "limit", Description: "Maximum number of rows (default 200)"},
					},
				},
				RunCtx: h.list,
			},
			{
				Name:        "get",
				Description: "Get a component by id",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Component id"},
					},
				},
				RunCtx: h.get,
			},
			{
				Name:        "get-by-library-id",
				Description: "Get a component by its @libraryId header value",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "library-id", Required: true, Description: "@libraryId header value"},
					},
				},
				RunCtx: h.getByLibraryID,
			},
			{
				Name:        "content-get",
				Description: "Read a component's source file",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Component id"},
					},
				},
				RunCtx: h.contentGet,
			},
			{
				Name:        "content-set",
				Description: "Write a component's source file (use - to read from stdin)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Component id"},
						{Name: "file", Required: true, Description: "Path to new file body, or - for stdin"},
					},
					Flags: []cliapp.Flag{
						{Name: "expected-sha256", Description: "Optimistic-concurrency guard; must match the current on-disk digest"},
					},
				},
				RunCtx: h.contentSet,
			},
		},
	}
}
