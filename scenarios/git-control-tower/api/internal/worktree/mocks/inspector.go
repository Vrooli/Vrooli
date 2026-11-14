// Package mocks provides programmable test doubles for the worktree
// seams. These fakes carry zero git logic — tests construct them with
// canned returns and assert against the recorded call history.
//
// Why mocks/ and not _test.go: tests in handlers/worktree need to
// import these fakes, which cross-package test files cannot do.
package mocks

import (
	"context"

	"git-control-tower/internal/worktree"
)

// FakeInspector is a programmable worktree.Inspector. Set the Result*
// fields before invoking the system under test; reading the Calls* /
// Last* fields after returns the recorded call.
type FakeInspector struct {
	ListResult            []worktree.Worktree
	ListErr               error
	IdentifyResult        worktree.Identity
	IdentifyErr           error
	ClaimedBranchesResult map[string]string
	ClaimedBranchesErr    error

	// Recorded calls; appended to in invocation order.
	ListCalls            []string
	IdentifyCalls        []string
	ClaimedBranchesCalls []string
}

// Compile-time assertion that this fake satisfies the seam.
var _ worktree.Inspector = (*FakeInspector)(nil)

// List returns the canned ListResult / ListErr after recording repoPath.
func (f *FakeInspector) List(_ context.Context, repoPath string) ([]worktree.Worktree, error) {
	f.ListCalls = append(f.ListCalls, repoPath)
	return f.ListResult, f.ListErr
}

// IdentifyPath returns the canned IdentifyResult / IdentifyErr.
func (f *FakeInspector) IdentifyPath(_ context.Context, repoPath string) (worktree.Identity, error) {
	f.IdentifyCalls = append(f.IdentifyCalls, repoPath)
	return f.IdentifyResult, f.IdentifyErr
}

// ClaimedBranches returns the canned ClaimedBranchesResult.
func (f *FakeInspector) ClaimedBranches(_ context.Context, repoPath string) (map[string]string, error) {
	f.ClaimedBranchesCalls = append(f.ClaimedBranchesCalls, repoPath)
	if f.ClaimedBranchesResult == nil && f.ClaimedBranchesErr == nil {
		return map[string]string{}, nil
	}
	return f.ClaimedBranchesResult, f.ClaimedBranchesErr
}
