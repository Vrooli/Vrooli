// Package worktree owns the `gct worktree ...` command group. It is
// the first proto+Connect-RPC domain in the GCT CLI; every method goes
// through the generated WorktreeServiceClient — never raw APIClient.
// Command surface loads from cli/manifest.json via cliapp.LoadFromManifest.
//
// Testing rule: the CLI never invokes real git. Handler tests in this
// package stand up an httptest.Server with a fake Service from the
// internal/worktree/mocks package.
package worktree

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "worktree"

// Register builds the worktree subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"WorktreeService.ListWorktrees":  h.list,
		"WorktreeService.GetWorktree":    h.get,
		"WorktreeService.CreateWorktree": h.create,
		"WorktreeService.RemoveWorktree": h.remove,
		"WorktreeService.LockWorktree":   h.lock,
		"WorktreeService.UnlockWorktree": h.unlock,
		"WorktreeService.MoveWorktree":   h.move,
		"WorktreeService.PruneWorktrees": h.prune,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("worktree: load from manifest: %w", err)
	}
	return group, nil
}
