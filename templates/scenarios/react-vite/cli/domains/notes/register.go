// Package notes is the CLI's notes-domain command surface. Mirrors
// the API's /api/v1/notes endpoints and the UI's lib/notes.ts client
// — the three CLI commands (list/create/get) wrap the same wire
// contract the other surfaces decode through.
//
// New domain packages copy this shape: a Register(core) returning a
// cliapp.SubcommandGroup, and one handler per subcommand in
// handlers.go. Resist accreting helpers until a third domain repeats
// the pattern; an internal/support package is the canonical extraction
// target when that day comes.
package notes

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `notes` subcommand group. The handlers in
// handlers.go close over `core` so they can issue versioned-API
// requests via core.Get / core.Request.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "notes",
		Description: "Manage notes (CRUD reference)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List all notes",
				Run:         h.list,
			},
			{
				Name:        "create",
				Description: "Create a note (--title TITLE [--body BODY])",
				Run:         h.create,
			},
			{
				Name:        "get",
				Description: "Get a note by id (notes get ID)",
				Run:         h.get,
			},
		},
	}
}
