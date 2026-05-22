package callerheader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

type recordingHandler struct {
	worktreeconnect.UnimplementedWorktreeServiceHandler
	caller     string
	authorized string
}

func (r *recordingHandler) ListWorktrees(ctx context.Context, req *connect.Request[worktreev1.ListWorktreesRequest]) (*connect.Response[worktreev1.ListWorktreesResponse], error) {
	r.caller = req.Header().Get(HeaderCaller)
	r.authorized = req.Header().Get(HeaderAuthorized)
	return connect.NewResponse(&worktreev1.ListWorktreesResponse{}), nil
}

func TestInterceptor_SendsCallerHeader(t *testing.T) {
	clearAuthEnv(t)
	h := &recordingHandler{}
	path, handler := worktreeconnect.NewWorktreeServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := worktreeconnect.NewWorktreeServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(New()))
	if _, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: "/x"})); err != nil {
		t.Fatalf("call: %v", err)
	}
	if h.caller == "" {
		t.Fatal("X-Vrooli-Caller should be set on outbound request")
	}
	if h.authorized != "" {
		t.Errorf("X-Vrooli-Authorized should be empty by default; got %q", h.authorized)
	}
}

func TestInterceptor_SendsAuthorizedHeader(t *testing.T) {
	t.Setenv(EnvAuthorized, "true")
	h := &recordingHandler{}
	path, handler := worktreeconnect.NewWorktreeServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := worktreeconnect.NewWorktreeServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(New()))
	if _, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: "/x"})); err != nil {
		t.Fatalf("call: %v", err)
	}
	if h.authorized != "true" {
		t.Errorf("X-Vrooli-Authorized = %q, want true", h.authorized)
	}
}

func clearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvAuthorized, "")
}
