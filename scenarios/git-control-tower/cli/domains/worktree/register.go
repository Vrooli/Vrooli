// Package worktree owns the `gct worktree ...` command group. It is
// the first proto+Connect-RPC domain in the GCT CLI; every method goes
// through the generated WorktreeServiceClient — never raw APIClient.
//
// Testing rule: the CLI never invokes real git. Handler tests in this
// package stand up an httptest.Server with a fake Service from the
// internal/worktree/mocks package.
package worktree

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the worktree subcommand group. The handler module
// constructs a generated Connect client lazily per command so the
// shared base-URL resolution stays consistent with the rest of the
// CLI.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "worktree",
		Description: "Manage git worktrees via the WorktreeService Connect-RPC surface",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List worktrees: worktree list --repo=PATH", Run: h.list},
			{Name: "get", NeedsAPI: true, Description: "Show one worktree: worktree get --repo=PATH --path=PATH", Run: h.get},
			{Name: "create", NeedsAPI: true, Description: "Create a worktree: worktree create --repo=PATH --path=PATH (--branch=NAME | --new-branch=NAME [--start=REF] [--track] | --commit=SHA) [--force]", Run: h.create},
			{Name: "remove", NeedsAPI: true, Description: "Remove a worktree: worktree remove --repo=PATH --path=PATH [--force]", Run: h.remove},
			{Name: "lock", NeedsAPI: true, Description: "Lock a worktree: worktree lock --repo=PATH --path=PATH [--reason=TEXT]", Run: h.lock},
			{Name: "unlock", NeedsAPI: true, Description: "Unlock a worktree: worktree unlock --repo=PATH --path=PATH", Run: h.unlock},
			{Name: "move", NeedsAPI: true, Description: "Move a worktree: worktree move --repo=PATH --path=PATH --new-path=PATH", Run: h.move},
			{Name: "prune", NeedsAPI: true, Description: "Prune worktrees: worktree prune --repo=PATH [--report-only]", Run: h.prune},
		},
	}
}
