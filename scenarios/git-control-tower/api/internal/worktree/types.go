// Package worktree owns git-worktree-aware identity and management for
// git-control-tower. It is the first proto+Connect-RPC domain in this
// scenario; future domains follow the same shape during incremental
// template migration.
//
// Layering:
//
//	handlers/worktree           — Connect-RPC handler, proto<->domain translation
//	internal/worktree           — domain types, service, seams (this package)
//	internal/worktree/mocks     — FakeInspector / FakeMutator for tests
//
// Testing rule: callers must NEVER invoke real git in tests. All paths
// exercise Inspector / Mutator with the fakes in mocks/.
package worktree

import "errors"

// Worktree is the canonical in-process representation of one git
// worktree (main or linked). Mirrors the proto Worktree message but
// stays free of proto imports so the domain layer can be reused by
// callers that don't want to depend on generated code.
type Worktree struct {
	Path           string
	Name           string
	HeadCommit     string
	Branch         string
	Detached       bool
	Locked         bool
	LockReason     string
	Prunable       bool
	PrunableReason string
	IsMain         bool
}

// Identity captures the Tier-1 passive-awareness facts about a given
// on-disk path: whether it is the main worktree or a linked one, the
// common repo root, and convenience fields used by the status header.
type Identity struct {
	IsLinkedWorktree    bool
	CommonRepoRoot      string
	WorktreeName        string
	WorktreeHead        string
	LinkedWorktreeCount int
	// Branch is the branch currently checked out at this path. Empty
	// when HEAD is detached.
	Branch string
	// Detached is true when HEAD is detached at this path.
	Detached bool
}

// CreateInput is the domain shape for CreateWorktree, sourced from the
// proto request after Connect-handler validation but before the seam
// boundary.
type CreateInput struct {
	RepoPath        string
	NewWorktreePath string
	// Exactly one of ExistingBranch / NewBranchName / Commit is set.
	ExistingBranch string
	NewBranchName  string
	NewBranchStart string
	Commit         string
	Force          bool
	Track          bool
}

// CreateMode classifies the create variant. Used both by Service
// validation and by tests asserting which Mutator method was called.
type CreateMode int

const (
	CreateModeUnspecified CreateMode = iota
	CreateModeExistingBranch
	CreateModeNewBranch
	CreateModeDetachedCommit
)

// Mode returns the resolved CreateMode for this input, or
// CreateModeUnspecified when none / multiple sources are set.
func (i CreateInput) Mode() CreateMode {
	var modes int
	mode := CreateModeUnspecified
	if i.ExistingBranch != "" {
		mode = CreateModeExistingBranch
		modes++
	}
	if i.NewBranchName != "" {
		mode = CreateModeNewBranch
		modes++
	}
	if i.Commit != "" {
		mode = CreateModeDetachedCommit
		modes++
	}
	if modes != 1 {
		return CreateModeUnspecified
	}
	return mode
}

// MoveInput is the domain shape for MoveWorktree.
type MoveInput struct {
	RepoPath        string
	WorktreePath    string
	NewWorktreePath string
}

// LockInput is the domain shape for LockWorktree.
type LockInput struct {
	RepoPath     string
	WorktreePath string
	Reason       string
}

// PruneInput is the domain shape for PruneWorktrees.
type PruneInput struct {
	RepoPath   string
	Reason     string
	ReportOnly bool
}

// PruneResult is the domain shape for the result of PruneWorktrees.
type PruneResult struct {
	PrunedPaths []string
}

// Sentinel domain errors. Connect handlers translate these to typed
// *connect.Error values; see service.go ErrorCodeFor / handlers/worktree
// for the mapping.
var (
	// ErrNotFound indicates the worktree path was not present in the
	// repo's worktree list.
	ErrNotFound = errors.New("worktree: not found")

	// ErrInvalid indicates a precondition violation that is the caller's
	// fault (path collision, attempt to remove main, invalid branch
	// spec, missing source).
	ErrInvalid = errors.New("worktree: invalid request")

	// ErrLocked indicates the operation cannot proceed because the
	// worktree is locked and the caller did not pass Force.
	ErrLocked = errors.New("worktree: locked")

	// ErrDirty indicates uncommitted changes block the operation and
	// the caller did not pass Force.
	ErrDirty = errors.New("worktree: dirty")
)
