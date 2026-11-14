package main

import (
	"context"
	"errors"
	"testing"
)

// TestEnrichBranchesWithWorktreeClaims_PopulatesField runs the
// REST-side enrichment with a fake claimedBranchesFn — NO real git.
func TestEnrichBranchesWithWorktreeClaims_PopulatesField(t *testing.T) {
	original := claimedBranchesFn
	t.Cleanup(func() { claimedBranchesFn = original })
	claimedBranchesFn = func(_ context.Context, _ string) (map[string]string, error) {
		return map[string]string{
			"feature":  "/tmp/feature",
			"hotfix":   "/tmp/hotfix",
			"detached": "",
		}, nil
	}
	result := &RepoBranchesResponse{
		Locals: []BranchInfo{
			{Name: "main"},
			{Name: "feature"},
			{Name: "hotfix"},
			{Name: "unclaimed"},
		},
	}
	enrichBranchesWithWorktreeClaims(context.Background(), result, "/repo")
	if result.Locals[0].CheckedOutInWorktree != "" {
		t.Errorf("main should not be claimed: %+v", result.Locals[0])
	}
	if result.Locals[1].CheckedOutInWorktree != "/tmp/feature" {
		t.Errorf("feature should be claimed: %+v", result.Locals[1])
	}
	if result.Locals[2].CheckedOutInWorktree != "/tmp/hotfix" {
		t.Errorf("hotfix should be claimed: %+v", result.Locals[2])
	}
	if result.Locals[3].CheckedOutInWorktree != "" {
		t.Errorf("unclaimed should be empty: %+v", result.Locals[3])
	}
}

// TestEnrichBranchesWithWorktreeClaims_SwallowsErrors confirms the
// enrichment is best-effort: failures must not corrupt the branch list.
func TestEnrichBranchesWithWorktreeClaims_SwallowsErrors(t *testing.T) {
	original := claimedBranchesFn
	t.Cleanup(func() { claimedBranchesFn = original })
	claimedBranchesFn = func(_ context.Context, _ string) (map[string]string, error) {
		return nil, errors.New("worktree inspector broken")
	}
	result := &RepoBranchesResponse{Locals: []BranchInfo{{Name: "main"}}}
	enrichBranchesWithWorktreeClaims(context.Background(), result, "/repo")
	if result.Locals[0].CheckedOutInWorktree != "" {
		t.Errorf("error must leave CheckedOutInWorktree empty: %+v", result.Locals[0])
	}
}

// TestEnrichBranchesWithWorktreeClaims_NoOpForEmpty handles the
// edge cases where there is nothing to enrich.
func TestEnrichBranchesWithWorktreeClaims_NoOpForEmpty(t *testing.T) {
	called := false
	original := claimedBranchesFn
	t.Cleanup(func() { claimedBranchesFn = original })
	claimedBranchesFn = func(_ context.Context, _ string) (map[string]string, error) {
		called = true
		return nil, nil
	}
	// nil result -> no-op
	enrichBranchesWithWorktreeClaims(context.Background(), nil, "/repo")
	// empty repoDir -> no-op
	enrichBranchesWithWorktreeClaims(context.Background(), &RepoBranchesResponse{Locals: []BranchInfo{{Name: "main"}}}, "")
	// empty locals -> no-op
	enrichBranchesWithWorktreeClaims(context.Background(), &RepoBranchesResponse{}, "/repo")
	if called {
		t.Errorf("claimedBranchesFn must not be called for empty inputs")
	}
}
