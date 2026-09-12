package worktree_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	hworktree "git-control-tower/handlers/worktree"
	dworktree "git-control-tower/internal/worktree"
	"git-control-tower/internal/worktree/mocks"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// All tests in this file run against a httptest.Server wired to the
// Connect handler with FAKE inspector/mutator. NO real git is invoked.

func newTestServer(t *testing.T, insp dworktree.Inspector, mut dworktree.Mutator) (worktreeconnect.WorktreeServiceClient, func()) {
	t.Helper()
	svc := dworktree.NewService(insp, mut)
	path, handler := hworktree.NewHandler(hworktree.Deps{Service: svc})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	client := worktreeconnect.NewWorktreeServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

func TestHandlerList(t *testing.T) {
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{
		{Path: "/repo", Name: "repo", IsMain: true, Branch: "main"},
		{Path: "/tmp/feature", Name: "feature", Branch: "feature"},
	}}
	client, cleanup := newTestServer(t, insp, &mocks.FakeMutator{})
	defer cleanup()

	resp, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: "/repo"}))
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(resp.Msg.Worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(resp.Msg.Worktrees))
	}
	if resp.Msg.Worktrees[0].IsMain != true || resp.Msg.Worktrees[1].Branch != "feature" {
		t.Fatalf("unexpected payload: %+v", resp.Msg.Worktrees)
	}
}

func TestHandlerList_RequiresRepoPath(t *testing.T) {
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, &mocks.FakeMutator{})
	defer cleanup()

	_, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{}))
	if err == nil {
		t.Fatalf("expected error for empty repo_path")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (%v)", connect.CodeOf(err), err)
	}
}

func TestHandlerGet_NotFound(t *testing.T) {
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{{Path: "/repo", IsMain: true}}}
	client, cleanup := newTestServer(t, insp, &mocks.FakeMutator{})
	defer cleanup()

	_, err := client.GetWorktree(context.Background(), connect.NewRequest(&worktreev1.GetWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/missing",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestHandlerCreate_ExistingBranch(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.CreateWorktree(context.Background(), connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/feature",
		Source:          &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"},
	}))
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if resp.Msg.DryRun {
		t.Fatalf("expected DryRun=false")
	}
	if resp.Msg.Worktree.Branch != "feature" || resp.Msg.Worktree.Path != "/tmp/feature" {
		t.Fatalf("unexpected: %+v", resp.Msg.Worktree)
	}
	if len(mut.AddCalls) != 1 {
		t.Fatalf("expected one Add call, got %d", len(mut.AddCalls))
	}
}

func TestHandlerCreate_NewBranchWithStartPoint(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	_, err := client.CreateWorktree(context.Background(), connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
		Source: &worktreev1.CreateWorktreeRequest_NewBranch{NewBranch: &worktreev1.NewBranchSpec{
			Name:       "topic",
			StartPoint: "main",
		}},
		Track: true,
	}))
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if len(mut.AddCalls) != 1 {
		t.Fatalf("expected one Add call")
	}
	got := mut.AddCalls[0]
	if got.NewBranchName != "topic" || got.NewBranchStart != "main" || !got.Track {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestHandlerCreate_DryRun_DoesNotCallMutator(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/feature",
		Source:          &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"},
	})
	req.Header().Set(hworktree.DryRunHeader, "true")

	resp, err := client.CreateWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if !resp.Msg.DryRun {
		t.Fatalf("expected DryRun=true, got %+v", resp.Msg)
	}
	if len(mut.AddCalls) != 0 {
		t.Fatalf("Mutator must NOT be called during dry-run, got %d calls", len(mut.AddCalls))
	}
}

func TestHandlerCreate_DryRun_StillValidates(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	// Multiple sources is invalid; dry-run must surface InvalidArgument.
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
		Source:          &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: "feature"},
	})
	// Manually set Commit to violate Mode() (can't through oneof normally — but we can pass a different oneof).
	// Use empty source instead — the validation rejects it.
	req.Msg.Source = nil
	req.Header().Set(hworktree.DryRunHeader, "true")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestHandlerRemove_DryRun(t *testing.T) {
	mut := &mocks.FakeMutator{}
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{
		{Path: "/repo", IsMain: true}, {Path: "/tmp/feature", Name: "feature"},
	}}
	client, cleanup := newTestServer(t, insp, mut)
	defer cleanup()

	req := connect.NewRequest(&worktreev1.RemoveWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/feature",
	})
	req.Header().Set(hworktree.DryRunHeader, "true")
	resp, err := client.RemoveWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
	if !resp.Msg.DryRun {
		t.Fatalf("expected DryRun=true")
	}
	if len(mut.RemoveCalls) != 0 {
		t.Fatalf("Mutator must NOT be called during dry-run")
	}
}

func TestHandlerRemove_RefusesMain(t *testing.T) {
	insp := &mocks.FakeInspector{ListResult: []dworktree.Worktree{{Path: "/repo", Name: "repo", IsMain: true}}}
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, insp, mut)
	defer cleanup()

	_, err := client.RemoveWorktree(context.Background(), connect.NewRequest(&worktreev1.RemoveWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/repo", Force: true,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestHandlerLock(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.LockWorktree(context.Background(), connect.NewRequest(&worktreev1.LockWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/feature", Reason: "agents pinned",
	}))
	if err != nil {
		t.Fatalf("LockWorktree: %v", err)
	}
	if !resp.Msg.Worktree.Locked || resp.Msg.Worktree.LockReason != "agents pinned" {
		t.Fatalf("unexpected: %+v", resp.Msg.Worktree)
	}
}

func TestHandlerUnlock(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	_, err := client.UnlockWorktree(context.Background(), connect.NewRequest(&worktreev1.UnlockWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/feature",
	}))
	if err != nil {
		t.Fatalf("UnlockWorktree: %v", err)
	}
}

func TestHandlerMove(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.MoveWorktree(context.Background(), connect.NewRequest(&worktreev1.MoveWorktreeRequest{
		RepoPath: "/repo", WorktreePath: "/tmp/old", NewWorktreePath: "/tmp/new",
	}))
	if err != nil {
		t.Fatalf("MoveWorktree: %v", err)
	}
	if resp.Msg.Worktree.Path != "/tmp/new" {
		t.Fatalf("unexpected: %+v", resp.Msg.Worktree)
	}
}

func TestHandlerPrune(t *testing.T) {
	mut := &mocks.FakeMutator{PruneResult: dworktree.PruneResult{PrunedPaths: []string{"old1", "old2"}}}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	resp, err := client.PruneWorktrees(context.Background(), connect.NewRequest(&worktreev1.PruneWorktreesRequest{
		RepoPath: "/repo",
	}))
	if err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
	if len(resp.Msg.PrunedPaths) != 2 {
		t.Fatalf("expected 2 pruned, got %v", resp.Msg.PrunedPaths)
	}
}

func TestHandlerPrune_DryRun(t *testing.T) {
	mut := &mocks.FakeMutator{}
	client, cleanup := newTestServer(t, &mocks.FakeInspector{}, mut)
	defer cleanup()

	req := connect.NewRequest(&worktreev1.PruneWorktreesRequest{RepoPath: "/repo"})
	req.Header().Set(hworktree.DryRunHeader, "true")
	resp, err := client.PruneWorktrees(context.Background(), req)
	if err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
	if !resp.Msg.DryRun {
		t.Fatalf("expected DryRun")
	}
	if len(mut.PruneCalls) != 0 {
		t.Fatalf("Mutator must not be called during dry-run")
	}
}
