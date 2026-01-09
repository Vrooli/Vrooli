package main

import (
	"context"
	"testing"
)

func TestPushToRemote_UpToDate(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.Branch.Head = "agi"
	fake.Branch.OID = "local123"
	fake.Branch.Upstream = "origin/agi"
	fake.RemoteBranches["origin/agi"] = FakeBranchRef{Name: "origin/agi", OID: "local123"}

	resp, err := PushToRemote(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, PushRequest{})
	if err != nil {
		t.Fatalf("PushToRemote returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if !resp.Verified {
		t.Fatalf("expected verified push")
	}
	if resp.Pushed {
		t.Fatalf("expected pushed=false when already up to date")
	}
	if !resp.UpToDate {
		t.Fatalf("expected up_to_date=true when already up to date")
	}
}

func TestPushToRemote_Pushed(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.Branch.Head = "agi"
	fake.Branch.OID = "local123"
	fake.Branch.Upstream = "origin/agi"
	fake.RemoteBranches["origin/agi"] = FakeBranchRef{Name: "origin/agi", OID: "old999"}

	resp, err := PushToRemote(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, PushRequest{})
	if err != nil {
		t.Fatalf("PushToRemote returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if !resp.Verified {
		t.Fatalf("expected verified push")
	}
	if !resp.Pushed {
		t.Fatalf("expected pushed=true when remote updated")
	}
	if resp.UpToDate {
		t.Fatalf("expected up_to_date=false when remote updated")
	}
}

func TestPushToRemote_DetectsRemoteNotUpdated(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.PushUpdatesRemote = false
	fake.Branch.Head = "agi"
	fake.Branch.OID = "local123"
	fake.Branch.Upstream = "origin/agi"
	fake.RemoteBranches["origin/agi"] = FakeBranchRef{Name: "origin/agi", OID: "old999"}

	resp, err := PushToRemote(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, PushRequest{})
	if err != nil {
		t.Fatalf("PushToRemote returned error: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false when remote did not update")
	}
	if !resp.Verified {
		t.Fatalf("expected verified=true when remote ref resolved")
	}
	if resp.Error == "" {
		t.Fatalf("expected error message when remote did not update")
	}
}

func TestPushToRemote_UsesUpstreamWhenAvailable(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.Branch.Head = "agi"
	fake.Branch.OID = "local123"
	fake.Branch.Upstream = "origin/master"
	fake.RemoteBranches["origin/master"] = FakeBranchRef{Name: "origin/master", OID: "old999"}

	resp, err := PushToRemote(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, PushRequest{})
	if err != nil {
		t.Fatalf("PushToRemote returned error: %v", err)
	}
	if resp.Remote != "origin" || resp.Branch != "master" {
		t.Fatalf("expected upstream origin/master, got %s/%s", resp.Remote, resp.Branch)
	}
	if !resp.Pushed {
		t.Fatalf("expected pushed=true when upstream updated")
	}
}

func TestRunUpstreamAction_Fetch(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	resp, err := RunUpstreamAction(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, UpstreamActionRequest{Action: "fetch", Remote: "origin"})
	if err != nil {
		t.Fatalf("RunUpstreamAction returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if fake.FetchCount != 1 {
		t.Fatalf("expected fetch count 1, got %d", fake.FetchCount)
	}
}

func TestRunUpstreamAction_SetUpstream(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.Branch.Head = "agi"
	resp, err := RunUpstreamAction(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, UpstreamActionRequest{
		Action:   "set_upstream",
		Branch:   "agi",
		Upstream: "origin/agi",
	})
	if err != nil {
		t.Fatalf("RunUpstreamAction returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if fake.Branch.Upstream != "origin/agi" {
		t.Fatalf("expected upstream origin/agi, got %s", fake.Branch.Upstream)
	}
}

func TestRunUpstreamAction_PushSetUpstream(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.Branch.Head = "agi"
	fake.Branch.OID = "local123"
	resp, err := RunUpstreamAction(context.Background(), PushPullDeps{
		Git:     fake,
		RepoDir: fake.RepoRoot,
	}, UpstreamActionRequest{
		Action: "push_set_upstream",
		Remote: "origin",
		Branch: "agi",
	})
	if err != nil {
		t.Fatalf("RunUpstreamAction returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if fake.CallCount("Push") != 1 {
		t.Fatalf("expected push to be called once")
	}
}
