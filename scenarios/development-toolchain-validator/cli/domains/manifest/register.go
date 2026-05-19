// Package manifest is the CLI's manifest command surface. Mirrors the
// API's Connect-RPC ManifestService.
package manifest

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `manifest` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "manifest",
		Description: "Manage per-(skill, golden) expected-diff manifests",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List all stored manifests",
				RunCtx:      h.list,
			},
			{
				Name:        "get",
				Description: "Show one manifest by (skill_id, golden_slug)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "skill_id", Required: true, Description: "Skill id"},
						{Name: "golden_slug", Required: true, Description: "Golden slug"},
					},
				},
				RunCtx: h.get,
			},
			{
				Name:        "upsert",
				Description: "Create or replace a manifest",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "skill", Required: true, Description: "Skill id"},
						{Name: "golden", Required: true, Description: "Golden slug"},
						{Name: "allow", Description: "Comma-separated list of allowed path globs"},
						{Name: "wildcard-allowed", Bool: true, Description: "Allow any path (allowed_paths becomes advisory)"},
						{Name: "convergence", Default: "none", Description: "Convergence target: none | empty-diff"},
						{Name: "template-version", Description: "Pinned template version"},
						{Name: "skill-version", Description: "Pinned skill version"},
					},
				},
				RunCtx: h.upsert,
			},
			{
				Name:        "clear-stale",
				Description: "Clear staleness override for a (skill, golden) manifest",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "skill", Required: true, Description: "Skill id"},
						{Name: "golden", Required: true, Description: "Golden slug"},
					},
				},
				RunCtx: h.clearStale,
			},
		},
	}
}
