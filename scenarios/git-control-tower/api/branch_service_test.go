package main

import (
	"context"
	"strings"
	"testing"
)

// [REQ:GCT-OT-P1-001] Branch operations

func TestListBranches_UsesCurrentBranchMetadata(t *testing.T) {
	fake := NewFakeGitRunner().
		WithBranch("feature/test", "origin/feature/test", 2, 1).
		WithLocalBranch("feature/test", "origin/feature/test", "deadbeef")

	resp, err := ListBranches(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Current != "feature/test" {
		t.Fatalf("expected current branch feature/test, got %s", resp.Current)
	}
	found := false
	for _, branch := range resp.Locals {
		if branch.Name == "feature/test" {
			found = true
			if !branch.IsCurrent {
				t.Fatalf("expected branch to be current")
			}
			if branch.Ahead != 2 || branch.Behind != 1 {
				t.Fatalf("expected ahead/behind 2/1, got %d/%d", branch.Ahead, branch.Behind)
			}
		}
	}
	if !found {
		t.Fatalf("expected feature/test branch in locals")
	}
}

func TestSwitchBranch_BlocksDirtyByDefault(t *testing.T) {
	fake := NewFakeGitRunner().AddUnstagedFile("dirty.txt")

	resp, err := SwitchBranch(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"}, SwitchBranchRequest{
		Name: "main",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected switch to be blocked for dirty tree")
	}
	if resp.Warning == nil || !resp.Warning.RequiresConfirmation {
		t.Fatalf("expected confirmation warning")
	}
}

func TestSwitchBranch_RemoteRequiresTracking(t *testing.T) {
	fake := NewFakeGitRunner().WithRemoteBranch("origin/feature/remote", "abc123")

	resp, err := SwitchBranch(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"}, SwitchBranchRequest{
		Name: "origin/feature/remote",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected switch to be blocked for remote-only branch")
	}
	if resp.Warning == nil || !resp.Warning.RequiresTracking {
		t.Fatalf("expected tracking warning")
	}
}

func TestSwitchBranch_TracksRemoteWhenRequested(t *testing.T) {
	fake := NewFakeGitRunner().WithRemoteBranch("origin/feature/remote", "abc123")

	resp, err := SwitchBranch(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"}, SwitchBranchRequest{
		Name:        "origin/feature/remote",
		TrackRemote: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected switch success, got warning: %v", resp.Warning)
	}
	if fake.Branch.Head != "feature/remote" {
		t.Fatalf("expected head to be feature/remote, got %s", fake.Branch.Head)
	}
}

func TestCreateBranch_InvalidName(t *testing.T) {
	fake := NewFakeGitRunner()
	fake.CheckRefFormatError = errTest("invalid branch")

	resp, err := CreateBranch(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"}, CreateBranchRequest{
		Name: "invalid name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected create to fail")
	}
	if len(resp.ValidationErrors) == 0 {
		t.Fatalf("expected validation error")
	}
}

func TestCreateBranch_CheckoutBlockedByDirty(t *testing.T) {
	fake := NewFakeGitRunner().AddUnstagedFile("dirty.txt")

	resp, err := CreateBranch(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"}, CreateBranchRequest{
		Name:     "feature/one",
		Checkout: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected create to be blocked for dirty checkout")
	}
	if resp.Warning == nil || !resp.Warning.RequiresConfirmation {
		t.Fatalf("expected confirmation warning")
	}
}

func TestPublishBranch_SetsUpstreamWhenMissing(t *testing.T) {
	fake := NewFakeGitRunner().WithBranch("feature/test", "", 1, 0)

	resp, err := PublishBranch(context.Background(), BranchDeps{Git: fake, RepoDir: "/fake/repo"}, PublishBranchRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected publish to succeed, got error: %s", resp.Error)
	}
	if !fake.AssertCalled("Push") {
		t.Fatalf("expected push to be called")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if !strings.Contains(strings.Join(last.Args, " "), "setUpstream=true") {
		t.Fatalf("expected setUpstream=true in push call")
	}
}

func errTest(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
