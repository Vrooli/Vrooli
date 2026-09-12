package worktree

import "context"

// seam: Inspector reads worktree state from git. Production wires
// gitInspector (worktree_git.go); tests wire FakeInspector from
// internal/worktree/mocks/inspector.go.
//
// Inspector is intentionally narrow: only the read-side operations
// the worktree.Service and repo.Service consume. Adding a worktree-
// shaped read here means adding it to FakeInspector too — the
// compile-time `var _ Inspector` assertion on both implementations
// catches any drift.
type Inspector interface {
	// List returns every worktree (main + linked) for the repository
	// containing repoPath. The first element is conventionally the
	// main worktree (IsMain == true). repoPath may be any directory
	// inside the repository.
	List(ctx context.Context, repoPath string) ([]Worktree, error)

	// IdentifyPath returns identity facts for repoPath. Resolves the
	// common repo root and figures out which worktree (if any) the
	// path resolves to, along with branch / HEAD info.
	IdentifyPath(ctx context.Context, repoPath string) (Identity, error)

	// ClaimedBranches returns a map of branch name -> worktree path
	// for every branch checked out in a non-main worktree of the repo
	// containing repoPath. Used to enrich the existing branch list
	// with the `checked_out_in_worktree` field.
	ClaimedBranches(ctx context.Context, repoPath string) (map[string]string, error)
}
