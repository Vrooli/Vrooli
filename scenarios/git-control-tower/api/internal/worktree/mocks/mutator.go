package mocks

import (
	"context"

	"git-control-tower/internal/worktree"
)

// FakeMutator is a programmable worktree.Mutator. Set the Result* or
// Err fields per method before invocation; the Calls slice records
// every call in invocation order.
type FakeMutator struct {
	AddResult       worktree.Worktree
	AddErr          error
	RemoveErr       error
	LockResult      worktree.Worktree
	LockErr         error
	UnlockResult    worktree.Worktree
	UnlockErr       error
	MoveResult      worktree.Worktree
	MoveErr         error
	PruneResult     worktree.PruneResult
	PruneErr        error

	AddCalls    []worktree.CreateInput
	RemoveCalls []RemoveCall
	LockCalls   []worktree.LockInput
	UnlockCalls []UnlockCall
	MoveCalls   []worktree.MoveInput
	PruneCalls  []worktree.PruneInput
}

// RemoveCall captures the args of one Remove invocation.
type RemoveCall struct {
	RepoPath     string
	WorktreePath string
	Force        bool
}

// UnlockCall captures the args of one Unlock invocation.
type UnlockCall struct {
	RepoPath     string
	WorktreePath string
}

var _ worktree.Mutator = (*FakeMutator)(nil)

func (f *FakeMutator) Add(_ context.Context, in worktree.CreateInput) (worktree.Worktree, error) {
	f.AddCalls = append(f.AddCalls, in)
	if f.AddErr != nil {
		return worktree.Worktree{}, f.AddErr
	}
	if f.AddResult.Path == "" {
		// Synthesize a reasonable default based on input so the caller
		// sees a non-empty Worktree without having to set AddResult
		// in every test.
		return worktree.Worktree{
			Path:       in.NewWorktreePath,
			Name:       worktreeNameFor(in.NewWorktreePath),
			Branch:     resolveBranch(in),
			HeadCommit: in.Commit,
			Detached:   in.Mode() == worktree.CreateModeDetachedCommit,
		}, nil
	}
	return f.AddResult, nil
}

func (f *FakeMutator) Remove(_ context.Context, repoPath, worktreePath string, force bool) error {
	f.RemoveCalls = append(f.RemoveCalls, RemoveCall{
		RepoPath: repoPath, WorktreePath: worktreePath, Force: force,
	})
	return f.RemoveErr
}

func (f *FakeMutator) Lock(_ context.Context, in worktree.LockInput) (worktree.Worktree, error) {
	f.LockCalls = append(f.LockCalls, in)
	if f.LockErr != nil {
		return worktree.Worktree{}, f.LockErr
	}
	if f.LockResult.Path == "" {
		return worktree.Worktree{
			Path:       in.WorktreePath,
			Name:       worktreeNameFor(in.WorktreePath),
			Locked:     true,
			LockReason: in.Reason,
		}, nil
	}
	return f.LockResult, nil
}

func (f *FakeMutator) Unlock(_ context.Context, repoPath, worktreePath string) (worktree.Worktree, error) {
	f.UnlockCalls = append(f.UnlockCalls, UnlockCall{RepoPath: repoPath, WorktreePath: worktreePath})
	if f.UnlockErr != nil {
		return worktree.Worktree{}, f.UnlockErr
	}
	if f.UnlockResult.Path == "" {
		return worktree.Worktree{
			Path: worktreePath,
			Name: worktreeNameFor(worktreePath),
		}, nil
	}
	return f.UnlockResult, nil
}

func (f *FakeMutator) Move(_ context.Context, in worktree.MoveInput) (worktree.Worktree, error) {
	f.MoveCalls = append(f.MoveCalls, in)
	if f.MoveErr != nil {
		return worktree.Worktree{}, f.MoveErr
	}
	if f.MoveResult.Path == "" {
		return worktree.Worktree{
			Path: in.NewWorktreePath,
			Name: worktreeNameFor(in.NewWorktreePath),
		}, nil
	}
	return f.MoveResult, nil
}

func (f *FakeMutator) Prune(_ context.Context, in worktree.PruneInput) (worktree.PruneResult, error) {
	f.PruneCalls = append(f.PruneCalls, in)
	return f.PruneResult, f.PruneErr
}

func worktreeNameFor(path string) string {
	if path == "" {
		return ""
	}
	// last path segment via simple manual split to avoid importing
	// filepath in the fake (keeps fakes lightweight).
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

func resolveBranch(in worktree.CreateInput) string {
	switch in.Mode() {
	case worktree.CreateModeExistingBranch:
		return in.ExistingBranch
	case worktree.CreateModeNewBranch:
		return in.NewBranchName
	}
	return ""
}
