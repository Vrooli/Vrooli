// Package aisearch registers CLI subcommands for AI search status,
// ad-hoc queries, and reindex lifecycle. The domain-specific shortcuts
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
			support.APICommand("reindex", "Start a full reindex [--wait]", deps.AISearchReindex),
			support.APICommand("reindex-status", "Show the current reindex job status", deps.AISearchReindexStat),
			support.APICommand("reindex-cancel", "Cancel the running reindex job", deps.AISearchReindexCan),
		},
	}
}
