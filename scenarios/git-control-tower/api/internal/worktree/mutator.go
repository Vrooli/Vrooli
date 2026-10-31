package worktree

import "context"

// seam: Mutator performs git-worktree write operations. Production
// wires gitMutator (worktree_git.go); tests wire FakeMutator from
// internal/worktree/mocks/mutator.go.
//
// Mutator is intentionally narrow: one method per WorktreeService
// mutating RPC. Adding a method here means updating FakeMutator too
// (compile-time assertion enforced).
type Mutator interface {
	// Add creates a worktree. The Mode of input determines which
	// underlying invocation is used; callers are expected to have
	// validated input.Mode() before calling.
	Add(ctx context.Context, input CreateInput) (Worktree, error)

	// Remove deletes a worktree directory and unregisters it. force
	// forwards --force to git (override safety checks).
	Remove(ctx context.Context, repoPath, worktreePath string, force bool) error

	// Lock marks a worktree as locked with an optional human-readable
	// reason.
	Lock(ctx context.Context, input LockInput) (Worktree, error)

	// Unlock clears a lock from a worktree and returns the updated
	// representation.
	Unlock(ctx context.Context, repoPath, worktreePath string) (Worktree, error)

	// Move relocates a worktree to a new absolute path.
	Move(ctx context.Context, input MoveInput) (Worktree, error)

	// Prune removes worktree administrative records git considers
	// prunable. reportOnly forwards --dry-run.
	Prune(ctx context.Context, input PruneInput) (PruneResult, error)
}
