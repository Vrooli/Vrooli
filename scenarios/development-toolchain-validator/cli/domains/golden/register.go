// Package golden is the CLI's golden-domain command surface. Mirrors
// the API's Connect-RPC GoldenService.
package golden

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `goldens` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "goldens",
		Description: "Manage template-pristine golden scenarios used by skill validation",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List all registered goldens",
				RunCtx:      h.list,
			},
			{
				Name:        "get",
				Description: "Show one golden by slug",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "slug", Required: true, Description: "Golden slug"},
					},
				},
				RunCtx: h.get,
			},
			{
				Name:        "register",
				Description: "Register a template-pristine golden scenario",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "slug", Required: true, Description: "Unique golden slug (kebab-case)"},
						{Name: "template", Required: true, Description: "Template id (e.g. react-vite)"},
						{Name: "version", Required: true, Description: "Template version to pin (e.g. 1.0.1)"},
						{Name: "path", Required: true, Description: "Repo-relative path to the golden directory"},
					},
				},
				RunCtx: h.register,
			},
			{
				Name:        "update",
				Description: "Patch a golden's path and/or template version",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "slug", Required: true, Description: "Golden slug"},
					},
					Flags: []cliapp.Flag{
						{Name: "path", Description: "New repo-relative path"},
						{Name: "version", Description: "New template version"},
					},
				},
				RunCtx: h.update,
			},
			{
				Name:        "delete",
				Description: "Delete a golden record (does not remove files on disk)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "slug", Required: true, Description: "Golden slug"},
					},
					Flags: []cliapp.Flag{
						{Name: "yes", Bool: true, Description: "Confirm deletion"},
					},
				},
				RunCtx: h.delete,
			},
			{
				Name:        "regenerate",
				Description: "Regenerate a golden's on-disk tree from its pinned template",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "slug", Required: true, Description: "Golden slug"},
					},
					Flags: []cliapp.Flag{
						{Name: "yes", Bool: true, Description: "Confirm regeneration"},
					},
				},
				RunCtx: h.regenerate,
			},
		},
	}
}
