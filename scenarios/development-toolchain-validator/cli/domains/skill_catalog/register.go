// Package skill_catalog is the CLI's skill-catalog command surface.
// Mirrors the API's Connect-RPC SkillCatalogService.
package skill_catalog

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `skill-catalog` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "skill-catalog",
		Description: "Mirror prompt-manager's skill catalog locally for manifest pinning and validation",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "sync",
				Description: "Pull the current skill set from prompt-manager and reconcile the local mirror",
				RunCtx:      h.sync,
			},
			{
				Name:        "list",
				Description: "List mirrored skills (id, version, content_hash)",
				RunCtx:      h.list,
			},
			{
				Name:        "get",
				Description: "Show one mirrored skill by id",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Skill id"},
					},
				},
				RunCtx: h.get,
			},
		},
	}
}
