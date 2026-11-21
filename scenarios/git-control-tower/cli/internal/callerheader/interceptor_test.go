package callerheader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"
	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

type recordingHandler struct {
	worktreeconnect.UnimplementedWorktreeServiceHandler
	caller     string
	authorized string
}

func (r *recordingHandler) ListWorktrees(ctx context.Context, req *connect.Request[worktreev1.ListWorktreesRequest]) (*connect.Response[worktreev1.ListWorktreesResponse], error) {
	r.caller = req.Header().Get(cliutil.HeaderCaller)
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
	h := &recordingHandler{}
	path, handler := worktreeconnect.NewWorktreeServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := worktreeconnect.NewWorktreeServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(NewWithEnvLookup(func(key string) string {
		if key != EnvAuthorized {
			t.Fatalf("unexpected environment lookup %q", key)
		}
		return "true"
	})))
	if _, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: "/x"})); err != nil {
		t.Fatalf("call: %v", err)
	}
	if h.authorized != "true" {
		t.Errorf("X-Vrooli-Authorized = %q, want true", h.authorized)
	}
}

func TestInterceptor_AuthorizedHeaderUsesInjectedEnvironment(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "case insensitive affirmative", value: " YES ", want: true},
		{name: "numeric affirmative", value: "1", want: true},
		{name: "false remains absent", value: "false", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAuthorized(func(key string) string {
				if key != EnvAuthorized {
					t.Fatalf("environment lookup key = %q, want %q", key, EnvAuthorized)
				}
				return tc.value
			}); got != tc.want {
				t.Errorf("isAuthorized() = %t, want %t", got, tc.want)
			}
		})
	}
}

func clearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvAuthorized, "")
}
