// Package validation_record is the CLI's record command surface.
package validation_record

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `record` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "record",
		Description: "List and inspect terminal validation records (append-only history)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List validation records (paginated)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "golden", Description: "Filter by golden slug"},
						{Name: "subject", Description: "Filter by subject id (skill id or tool name)"},
						{Name: "kind", Description: "Filter by tuple kind: skill | tool"},
						{Name: "page-size", Default: "50", Description: "Records per page"},
						{Name: "page-token", Description: "Opaque cursor from the previous page"},
					},
				},
				RunCtx: h.list,
			},
			{
				Name:        "get",
				Description: "Show one record by id",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Record id (UUID)"},
					},
				},
				RunCtx: h.get,
			},
		},
	}
}
