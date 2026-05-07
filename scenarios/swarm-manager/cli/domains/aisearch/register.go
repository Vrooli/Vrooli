// Package aisearch registers CLI subcommands for AI search status,
// ad-hoc queries, and reconcile lifecycle. The domain-specific shortcuts
// `backlog search-ai` and `initiatives search-ai` are registered by the
// respective domain packages.
package aisearch

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "ai-search",
		Description: "AI (semantic) search over backlog items and initiatives",
		Subcommands: []cliapp.Command{
			support.APICommand("status", "Show AI search availability and index coverage", deps.AISearchStatus),
			support.APICommand("query", "Search both collections (use backlog/initiatives search-ai to scope)", deps.AISearchQuery),
			support.APICommand("reconcile", "Diff disk against the index and apply minimum-change upserts/deletes [--wait]", deps.AISearchReconcile),
			support.APICommand("reconcile-status", "Show the current reconcile job status", deps.AISearchReconcileStat),
			support.APICommand("reconcile-cancel", "Cancel the running reconcile job", deps.AISearchReconcileCan),
		},
	}
}
