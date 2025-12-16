package main

import (
	"context"
	"errors"
	"testing"
)

// [REQ:GCT-OT-P0-006] Push/pull status tests

func TestGetSyncStatus_BasicBranchInfo(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("feature/test", "origin/feature/test", 2, 1).
		WithRemoteURL("https://github.com/example/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Branch != "feature/test" {
		t.Errorf("expected branch 'feature/test', got %q", resp.Branch)
	}
	if resp.Upstream != "origin/feature/test" {
		t.Errorf("expected upstream 'origin/feature/test', got %q", resp.Upstream)
	}
	if resp.Ahead != 2 {
		t.Errorf("expected ahead=2, got %d", resp.Ahead)
	}
	if resp.Behind != 1 {
		t.Errorf("expected behind=1, got %d", resp.Behind)
	}
	if !resp.HasUpstream {
		t.Error("expected HasUpstream=true")
	}
	if resp.RemoteURL != "https://github.com/example/repo.git" {
		t.Errorf("expected remote URL, got %q", resp.RemoteURL)
	}
}

func TestGetSyncStatus_NeedsFlags(t *testing.T) {
	tests := []struct {
		name     string
		ahead    int
		behind   int
		wantPush bool
		wantPull bool
	}{
		{"ahead only", 3, 0, true, false},
		{"behind only", 0, 2, false, true},
		{"both", 1, 1, true, true},
		{"in sync", 0, 0, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeGit := NewFakeGitRunner().
				WithBranch("main", "origin/main", tc.ahead, tc.behind).
				WithRemoteURL("https://github.com/example/repo.git")

			resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
				Git:     fakeGit,
				RepoDir: "/fake/repo",
			}, SyncStatusRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.NeedsPush != tc.wantPush {
				t.Errorf("NeedsPush: got %v, want %v", resp.NeedsPush, tc.wantPush)
			}
			if resp.NeedsPull != tc.wantPull {
				t.Errorf("NeedsPull: got %v, want %v", resp.NeedsPull, tc.wantPull)
			}
		})
	}
}

func TestGetSyncStatus_CanPush(t *testing.T) {
	tests := []struct {
		name     string
		ahead    int
		behind   int
		upstream string
		wantCan  bool
	}{
		{"ahead only - can push", 2, 0, "origin/main", true},
		{"behind blocks push", 2, 1, "origin/main", false},
		{"nothing to push", 0, 0, "origin/main", false},
		{"no upstream", 2, 0, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeGit := NewFakeGitRunner().
				WithBranch("main", tc.upstream, tc.ahead, tc.behind).
				WithRemoteURL("https://github.com/example/repo.git")

			resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
				Git:     fakeGit,
				RepoDir: "/fake/repo",
			}, SyncStatusRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.CanPush != tc.wantCan {
				t.Errorf("CanPush: got %v, want %v", resp.CanPush, tc.wantCan)
			}
		})
	}
}

func TestGetSyncStatus_CanPull(t *testing.T) {
	tests := []struct {
		name     string
		behind   int
		upstream string
		wantCan  bool
	}{
		{"behind - can pull", 2, "origin/main", true},
		{"nothing to pull", 0, "origin/main", false},
		{"no upstream", 2, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeGit := NewFakeGitRunner().
				WithBranch("main", tc.upstream, 0, tc.behind).
				WithRemoteURL("https://github.com/example/repo.git")

			resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
				Git:     fakeGit,
				RepoDir: "/fake/repo",
			}, SyncStatusRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.CanPull != tc.wantCan {
				t.Errorf("CanPull: got %v, want %v", resp.CanPull, tc.wantCan)
			}
		})
	}
}

func TestGetSyncStatus_UncommittedChanges(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 0, 0).
		WithRemoteURL("https://github.com/example/repo.git").
		AddStagedFile("staged.go").
		AddUnstagedFile("modified.go")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HasUncommittedChanges {
		t.Error("expected HasUncommittedChanges=true")
	}
}

func TestGetSyncStatus_SafetyWarnings_Diverged(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 2, 3).
		WithRemoteURL("https://github.com/example/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range resp.SafetyWarnings {
		if w == "Branch has diverged from remote - pull before push to avoid force-push" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected diverged warning, got: %v", resp.SafetyWarnings)
	}
}

func TestGetSyncStatus_SafetyWarnings_NoUpstream(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("feature/new", "", 0, 0).
		WithRemoteURL("https://github.com/example/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, w := range resp.SafetyWarnings {
		if w == "No upstream branch configured - push requires explicit remote/branch" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected no-upstream warning, got: %v", resp.SafetyWarnings)
	}
}

func TestGetSyncStatus_Recommendations_InSync(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 0, 0).
		WithRemoteURL("https://github.com/example/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range resp.Recommendations {
		if r == "Branch is in sync with remote" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'in sync' recommendation, got: %v", resp.Recommendations)
	}
}

func TestGetSyncStatus_WithFetch(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 0, 0).
		WithRemoteURL("https://github.com/example/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{Fetch: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Fetched {
		t.Error("expected Fetched=true")
	}
	if !fakeGit.AssertCalled("FetchRemote") {
		t.Error("expected FetchRemote to be called")
	}
}

func TestGetSyncStatus_FetchError_NonFatal(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 1, 0).
		WithRemoteURL("https://github.com/example/repo.git")
	fakeGit.FetchError = errors.New("network error")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{Fetch: true})
	if err != nil {
		t.Fatalf("fetch error should be non-fatal, got: %v", err)
	}

	if resp.FetchError == "" {
		t.Error("expected FetchError to be set")
	}
	if resp.Fetched {
		t.Error("expected Fetched=false when fetch fails")
	}
	// Should still return valid status
	if resp.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", resp.Branch)
	}
}

func TestGetSyncStatus_CustomRemote(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "upstream/main", 0, 0).
		WithRemoteURL("https://github.com/upstream/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{Remote: "upstream"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify custom remote was used
	if !fakeGit.AssertCalled("GetRemoteURL") {
		t.Error("expected GetRemoteURL to be called")
	}
	// Check call args include custom remote
	for _, call := range fakeGit.Calls {
		if call.Method == "GetRemoteURL" {
			if len(call.Args) < 2 || call.Args[1] != "upstream" {
				t.Errorf("expected remote 'upstream', got args: %v", call.Args)
			}
			break
		}
	}

	if resp.RemoteURL != "https://github.com/upstream/repo.git" {
		t.Errorf("expected upstream URL, got %q", resp.RemoteURL)
	}
}

func TestGetSyncStatus_MissingDeps(t *testing.T) {
	_, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     nil,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})

	if err == nil {
		t.Error("expected error for nil GitRunner")
	}

	_, err = GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     NewFakeGitRunner(),
		RepoDir: "",
	}, SyncStatusRequest{})

	if err == nil {
		t.Error("expected error for empty RepoDir")
	}
}

func TestGetSyncStatus_Conflicts(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 0, 0).
		WithRemoteURL("https://github.com/example/repo.git").
		AddConflictFile("conflict.go")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have conflict warning
	found := false
	for _, w := range resp.SafetyWarnings {
		if w == "1 file(s) have unresolved merge conflicts" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected conflict warning, got: %v", resp.SafetyWarnings)
	}

	// Should recommend resolving conflicts
	foundRec := false
	for _, r := range resp.Recommendations {
		if r == "Resolve merge conflicts before continuing" {
			foundRec = true
			break
		}
	}
	if !foundRec {
		t.Errorf("expected conflict resolution recommendation, got: %v", resp.Recommendations)
	}
}

func TestGetSyncStatus_FarBehind(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("main", "origin/main", 0, 15).
		WithRemoteURL("https://github.com/example/repo.git")

	resp, err := GetSyncStatus(context.Background(), SyncStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, SyncStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should warn about being far behind
	found := false
	for _, w := range resp.SafetyWarnings {
		if w == "Branch is 15 commits behind - consider rebasing" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected far-behind warning, got: %v", resp.SafetyWarnings)
	}
}
