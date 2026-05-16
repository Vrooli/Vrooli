package worktree

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// CLI tests substitute clientFactory with a fakeClient. NO real git
// is ever invoked. NO real network is hit. The clientFactory seam
// keeps the substitution explicit.

type fakeClient struct {
	listResp   *worktreev1.ListWorktreesResponse
	listErr    error
	getResp    *worktreev1.GetWorktreeResponse
	getErr     error
	createResp *worktreev1.CreateWorktreeResponse
	createErr  error
	removeResp *worktreev1.RemoveWorktreeResponse
	removeErr  error
	lockResp   *worktreev1.LockWorktreeResponse
	unlockResp *worktreev1.UnlockWorktreeResponse
	moveResp   *worktreev1.MoveWorktreeResponse
	pruneResp  *worktreev1.PruneWorktreesResponse

	lastCreate *worktreev1.CreateWorktreeRequest
	lastRemove *worktreev1.RemoveWorktreeRequest
	lastList   *worktreev1.ListWorktreesRequest
}

var _ worktreeconnect.WorktreeServiceClient = (*fakeClient)(nil)

func (f *fakeClient) ListWorktrees(_ context.Context, req *connect.Request[worktreev1.ListWorktreesRequest]) (*connect.Response[worktreev1.ListWorktreesResponse], error) {
	f.lastList = req.Msg
	if f.listErr != nil {
		return nil, f.listErr
	}
	return connect.NewResponse(f.listResp), nil
}

func (f *fakeClient) GetWorktree(_ context.Context, req *connect.Request[worktreev1.GetWorktreeRequest]) (*connect.Response[worktreev1.GetWorktreeResponse], error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	_ = req
	return connect.NewResponse(f.getResp), nil
}

func (f *fakeClient) CreateWorktree(_ context.Context, req *connect.Request[worktreev1.CreateWorktreeRequest]) (*connect.Response[worktreev1.CreateWorktreeResponse], error) {
	f.lastCreate = req.Msg
	if f.createErr != nil {
		return nil, f.createErr
	}
	return connect.NewResponse(f.createResp), nil
}

func (f *fakeClient) RemoveWorktree(_ context.Context, req *connect.Request[worktreev1.RemoveWorktreeRequest]) (*connect.Response[worktreev1.RemoveWorktreeResponse], error) {
	f.lastRemove = req.Msg
	if f.removeErr != nil {
		return nil, f.removeErr
	}
	return connect.NewResponse(f.removeResp), nil
}

func (f *fakeClient) LockWorktree(_ context.Context, _ *connect.Request[worktreev1.LockWorktreeRequest]) (*connect.Response[worktreev1.LockWorktreeResponse], error) {
	return connect.NewResponse(f.lockResp), nil
}

func (f *fakeClient) UnlockWorktree(_ context.Context, _ *connect.Request[worktreev1.UnlockWorktreeRequest]) (*connect.Response[worktreev1.UnlockWorktreeResponse], error) {
	return connect.NewResponse(f.unlockResp), nil
}

func (f *fakeClient) MoveWorktree(_ context.Context, _ *connect.Request[worktreev1.MoveWorktreeRequest]) (*connect.Response[worktreev1.MoveWorktreeResponse], error) {
	return connect.NewResponse(f.moveResp), nil
}

func (f *fakeClient) PruneWorktrees(_ context.Context, _ *connect.Request[worktreev1.PruneWorktreesRequest]) (*connect.Response[worktreev1.PruneWorktreesResponse], error) {
	return connect.NewResponse(f.pruneResp), nil
}

func swapFactory(fake worktreeconnect.WorktreeServiceClient) func() {
	prev := clientFactory
	clientFactory = func(_ *cliapp.ScenarioApp) worktreeconnect.WorktreeServiceClient { return fake }
	return func() { clientFactory = prev }
}

func TestParse(t *testing.T) {
	flags := parse([]string{"--repo=/r", "--force", "--reason=because"})
	if flags.Get("repo") != "/r" {
		t.Fatalf("expected /r, got %q", flags.Get("repo"))
	}
	if !flags.Flag("force") {
		t.Fatalf("expected force=true")
	}
	if flags.Get("reason") != "because" {
		t.Fatalf("expected reason=because, got %q", flags.Get("reason"))
	}
}

func TestListRequiresRepoFlag(t *testing.T) {
	defer swapFactory(&fakeClient{})()
	h := &handlers{}
	err := h.list(nil)
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestListCallsClient(t *testing.T) {
	fc := &fakeClient{listResp: &worktreev1.ListWorktreesResponse{
		Worktrees: []*worktreev1.Worktree{
			{Path: "/repo", Name: "repo", IsMain: true, Branch: "main"},
			{Path: "/tmp/feature", Name: "feature", Branch: "feature"},
		},
	}}
	defer swapFactory(fc)()
	h := &handlers{}
	if err := h.list([]string{"--repo=/repo"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if fc.lastList.RepoPath != "/repo" {
		t.Fatalf("expected repo /repo, got %q", fc.lastList.RepoPath)
	}
}

func TestCreateRequiresSource(t *testing.T) {
	defer swapFactory(&fakeClient{})()
	h := &handlers{}
	err := h.create([]string{"--repo=/r", "--path=/p"})
	if err == nil || !strings.Contains(err.Error(), "requires one of") {
		t.Fatalf("expected source error, got %v", err)
	}
}

func TestCreateRequiresRepoAndPath(t *testing.T) {
	defer swapFactory(&fakeClient{})()
	h := &handlers{}
	if err := h.create([]string{"--branch=main"}); err == nil {
		t.Fatalf("expected usage error")
	}
}

func TestCreateExistingBranch(t *testing.T) {
	fc := &fakeClient{createResp: &worktreev1.CreateWorktreeResponse{
		Worktree: &worktreev1.Worktree{Path: "/tmp/feature", Branch: "feature"},
	}}
	defer swapFactory(fc)()
	h := &handlers{}
	if err := h.create([]string{"--repo=/r", "--path=/tmp/feature", "--branch=feature"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if fc.lastCreate.NewWorktreePath != "/tmp/feature" {
		t.Fatalf("path: %q", fc.lastCreate.NewWorktreePath)
	}
	src, ok := fc.lastCreate.Source.(*worktreev1.CreateWorktreeRequest_ExistingBranch)
	if !ok || src.ExistingBranch != "feature" {
		t.Fatalf("unexpected source: %+v", fc.lastCreate.Source)
	}
}

func TestCreateNewBranchWithStartAndTrack(t *testing.T) {
	fc := &fakeClient{createResp: &worktreev1.CreateWorktreeResponse{
		Worktree: &worktreev1.Worktree{Path: "/tmp/x", Branch: "topic"},
	}}
	defer swapFactory(fc)()
	h := &handlers{}
	if err := h.create([]string{"--repo=/r", "--path=/tmp/x", "--new-branch=topic", "--start=main", "--track"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	src, ok := fc.lastCreate.Source.(*worktreev1.CreateWorktreeRequest_NewBranch)
	if !ok {
		t.Fatalf("expected new-branch source, got %T", fc.lastCreate.Source)
	}
	if src.NewBranch.Name != "topic" || src.NewBranch.StartPoint != "main" {
		t.Fatalf("unexpected: %+v", src.NewBranch)
	}
	if !fc.lastCreate.Track {
		t.Fatalf("expected Track=true")
	}
}

func TestCreateDetachedCommit(t *testing.T) {
	fc := &fakeClient{createResp: &worktreev1.CreateWorktreeResponse{
		Worktree: &worktreev1.Worktree{Path: "/tmp/d", Detached: true},
	}}
	defer swapFactory(fc)()
	h := &handlers{}
	if err := h.create([]string{"--repo=/r", "--path=/tmp/d", "--commit=abcdef"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	src, ok := fc.lastCreate.Source.(*worktreev1.CreateWorktreeRequest_Commit)
	if !ok || src.Commit != "abcdef" {
		t.Fatalf("unexpected: %+v", fc.lastCreate.Source)
	}
}

func TestRemoveForwardsForce(t *testing.T) {
	fc := &fakeClient{removeResp: &worktreev1.RemoveWorktreeResponse{}}
	defer swapFactory(fc)()
	h := &handlers{}
	if err := h.remove([]string{"--repo=/r", "--path=/p", "--force"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !fc.lastRemove.Force {
		t.Fatalf("expected Force=true")
	}
}

func TestRemoveSurfaceConnectErrors(t *testing.T) {
	fc := &fakeClient{removeErr: connect.NewError(connect.CodeInvalidArgument, errors.New("cannot remove main"))}
	defer swapFactory(fc)()
	h := &handlers{}
	err := h.remove([]string{"--repo=/r", "--path=/p"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLockRequiresFlags(t *testing.T) {
	defer swapFactory(&fakeClient{})()
	h := &handlers{}
	if err := h.lock(nil); err == nil {
		t.Fatalf("expected usage error")
	}
}

func TestMoveRequiresNewPath(t *testing.T) {
	defer swapFactory(&fakeClient{})()
	h := &handlers{}
	if err := h.move([]string{"--repo=/r", "--path=/p"}); err == nil {
		t.Fatalf("expected error for missing --new-path")
	}
}

func TestPrune(t *testing.T) {
	fc := &fakeClient{pruneResp: &worktreev1.PruneWorktreesResponse{PrunedPaths: []string{"old1"}}}
	defer swapFactory(fc)()
	h := &handlers{}
	if err := h.prune([]string{"--repo=/r"}); err != nil {
		t.Fatalf("prune: %v", err)
	}
}

func TestShortSHA(t *testing.T) {
	if shortSHA("abcdef1234567") != "abcdef1" {
		t.Fatalf("got %q", shortSHA("abcdef1234567"))
	}
	if shortSHA("abc") != "abc" {
		t.Fatalf("short shouldn't truncate")
	}
}
