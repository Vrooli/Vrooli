package repo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	hrepo "git-control-tower/handlers/repo"
	drepo "git-control-tower/internal/repo"
	"git-control-tower/internal/worktree"
	"git-control-tower/internal/worktree/mocks"

	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
)

func newTestRepoClient(t *testing.T, insp worktree.Inspector) (repoconnect.RepoServiceClient, func()) {
	t.Helper()
	svc := drepo.NewService(insp)
	path, handler := hrepo.NewHandler(hrepo.Deps{Service: svc})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	return repoconnect.NewRepoServiceClient(srv.Client(), srv.URL), srv.Close
}

func TestRepoHandler_GetRepoStatus_LinkedWorktree(t *testing.T) {
	insp := &mocks.FakeInspector{
		IdentifyResult: worktree.Identity{
			IsLinkedWorktree:    true,
			CommonRepoRoot:      "/home/user/repo",
			WorktreeName:        "feature",
			WorktreeHead:        "abc",
			LinkedWorktreeCount: 3,
			Branch:              "feature",
		},
	}
	client, cleanup := newTestRepoClient(t, insp)
	defer cleanup()

	resp, err := client.GetRepoStatus(context.Background(), connect.NewRequest(&repov1.GetRepoStatusRequest{RepoPath: "/home/user/wt/feature"}))
	if err != nil {
		t.Fatalf("GetRepoStatus: %v", err)
	}
	if !resp.Msg.Worktree.IsLinkedWorktree {
		t.Fatalf("expected IsLinkedWorktree=true")
	}
	if resp.Msg.Worktree.WorktreeName != "feature" || resp.Msg.Branch != "feature" {
		t.Fatalf("unexpected: %+v", resp.Msg)
	}
	if resp.Msg.Worktree.LinkedWorktreeCount != 3 {
		t.Fatalf("expected count=3, got %d", resp.Msg.Worktree.LinkedWorktreeCount)
	}
}

func TestRepoHandler_GetRepoStatus_RequiresRepoPath(t *testing.T) {
	client, cleanup := newTestRepoClient(t, &mocks.FakeInspector{})
	defer cleanup()
	_, err := client.GetRepoStatus(context.Background(), connect.NewRequest(&repov1.GetRepoStatusRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}
