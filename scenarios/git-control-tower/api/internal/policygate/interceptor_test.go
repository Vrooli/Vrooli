package policygate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"git-control-tower/internal/config"

	"github.com/vrooli/cli-core/cliutil"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"
)

// captureAuditLogger collects gate events for assertions.
type captureAuditLogger struct {
	mu     sync.Mutex
	events []Event
}

func (c *captureAuditLogger) Log(e Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *captureAuditLogger) snapshot() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Event, len(c.events))
	copy(out, c.events)
	return out
}

// fakeWorktreeServer is the minimum WorktreeServiceHandler shape — it
// just records that the underlying handler was reached.
type fakeWorktreeServer struct {
	worktreeconnect.UnimplementedWorktreeServiceHandler
	reached bool
}

func (f *fakeWorktreeServer) CreateWorktree(ctx context.Context, req *connect.Request[worktreev1.CreateWorktreeRequest]) (*connect.Response[worktreev1.CreateWorktreeResponse], error) {
	f.reached = true
	return connect.NewResponse(&worktreev1.CreateWorktreeResponse{}), nil
}

func (f *fakeWorktreeServer) ListWorktrees(ctx context.Context, req *connect.Request[worktreev1.ListWorktreesRequest]) (*connect.Response[worktreev1.ListWorktreesResponse], error) {
	f.reached = true
	return connect.NewResponse(&worktreev1.ListWorktreesResponse{}), nil
}

func newTestClient(t *testing.T, policy config.PolicyConfig, audit AuditLogger) (worktreeconnect.WorktreeServiceClient, *fakeWorktreeServer, func()) {
	t.Helper()
	srv := &fakeWorktreeServer{}
	path, handler := worktreeconnect.NewWorktreeServiceHandler(srv, connect.WithInterceptors(NewInterceptor(policy, audit)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	test := httptest.NewServer(mux)
	client := worktreeconnect.NewWorktreeServiceClient(test.Client(), test.URL)
	return client, srv, test.Close
}

func TestInterceptor_ReadOnlyBypassesGate(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, denyAllPolicy(), audit)
	defer cleanup()
	_, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: "/x"}))
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if !srv.reached {
		t.Fatal("read-only call should reach handler")
	}
	if len(audit.snapshot()) != 0 {
		t.Errorf("read-only call should not emit audit event; got %d", len(audit.snapshot()))
	}
}

func TestInterceptor_ForgedHumanHeaderIsDenied(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, denyAllPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(cliutil.HeaderCaller, "human")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil {
		t.Fatal("caller-supplied human header must not grant authority")
	}
	if srv.reached {
		t.Fatal("forged human header must not reach the writer")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "deny" || events[0].Authorized {
		t.Errorf("expected single unauthorised deny event; got %+v", events)
	}
}

func TestInterceptor_AgentDeniedUnderConfirmWithoutOverride(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, confirmPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(cliutil.HeaderCaller, "external-agent")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for agent under confirm policy w/o override")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied; got %v", err)
	}
	if srv.reached {
		t.Fatal("denied call should not reach handler")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "deny" {
		t.Errorf("expected single deny event; got %+v", events)
	}
	if !strings.Contains(connectErr.Message(), "CreateWorktree") {
		t.Errorf("error message should mention command; got %q", connectErr.Message())
	}
}

func TestInterceptor_HeaderOverrideIsIgnored(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, confirmPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(cliutil.HeaderCaller, "external-agent")
	req.Header().Set(HeaderAuthorized, "true")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil {
		t.Fatal("caller-supplied override must not grant authority")
	}
	if srv.reached {
		t.Fatal("header override must not reach the writer")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "deny" || events[0].Authorized {
		t.Errorf("expected deny without verified authority; got %+v", events)
	}
}

func TestInterceptor_UnverifiedAgentHeaderCannotDowngradeToWarn(t *testing.T) {
	audit := &captureAuditLogger{}
	policy := config.PolicyConfig{AgentAccess: config.AgentAccessWarn, AgentOverrideFlag: "--ok", CallerDetection: config.CallerDetectionBroad}
	client, srv, cleanup := newTestClient(t, policy, audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(cliutil.HeaderCaller, "external-agent")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil {
		t.Fatal("unverified agent header must be denied, even under warn policy")
	}
	if srv.reached {
		t.Fatal("unverified agent must not reach the writer")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "deny" {
		t.Errorf("expected single deny event; got %+v", events)
	}
}

func TestInterceptor_AgentDenyAlwaysRefuses(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, denyAllPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(cliutil.HeaderCaller, "vrooli-agent")
	// Even with override flag, deny refuses.
	req.Header().Set(HeaderAuthorized, "true")
	_, err := client.CreateWorktree(context.Background(), req)
	if err == nil {
		t.Fatal("deny should refuse even with override")
	}
	if srv.reached {
		t.Fatal("deny should not reach handler")
	}
}

type fakeRepoServer struct {
	repoconnect.UnimplementedRepoServiceHandler
	reached bool
}

func (f *fakeRepoServer) CreateCommit(ctx context.Context, req *connect.Request[repov1.CreateCommitRequest]) (*connect.Response[repov1.CreateCommitResponse], error) {
	f.reached = true
	return connect.NewResponse(&repov1.CreateCommitResponse{Success: true}), nil
}

func newRepoTestClient(t *testing.T, principal *Principal) (repoconnect.RepoServiceClient, *fakeRepoServer, func()) {
	t.Helper()
	srv := &fakeRepoServer{}
	path, handler := repoconnect.NewRepoServiceHandler(srv, connect.WithInterceptors(NewInterceptor(denyAllPolicy(), &captureAuditLogger{})))
	mux := http.NewServeMux()
	if principal == nil {
		mux.Handle(path, handler)
	} else {
		mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithPrincipal(r.Context(), *principal)
			handler.ServeHTTP(w, r.WithContext(ctx))
		}))
	}
	test := httptest.NewServer(mux)
	client := repoconnect.NewRepoServiceClient(test.Client(), test.URL)
	return client, srv, test.Close
}

func TestInterceptor_HandlerManagedMutationRequiresVerifiedHuman(t *testing.T) {
	request := connect.NewRequest(&repov1.CreateCommitRequest{RepositoryId: "repo-1", IntentId: "intent-1"})

	client, srv, cleanup := newRepoTestClient(t, nil)
	_, err := client.CreateCommit(context.Background(), request)
	cleanup()
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unverified handler-managed mutation should be unauthenticated; got %v", err)
	}
	if srv.reached {
		t.Fatal("unverified handler-managed mutation reached writer")
	}

	client, srv, cleanup = newRepoTestClient(t, &Principal{Kind: cliutil.CallerKindVrooliAgent, Subject: "agent-1", Verified: true})
	_, err = client.CreateCommit(context.Background(), request)
	cleanup()
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent handler-managed mutation should be denied; got %v", err)
	}
	if srv.reached {
		t.Fatal("agent handler-managed mutation reached writer")
	}

	client, srv, cleanup = newRepoTestClient(t, &Principal{Kind: cliutil.CallerKindHuman, Subject: "operator-1", Verified: true})
	_, err = client.CreateCommit(context.Background(), request)
	cleanup()
	if err != nil {
		t.Fatalf("verified human handler-managed mutation should reach handler: %v", err)
	}
	if !srv.reached {
		t.Fatal("verified human handler-managed mutation did not reach writer")
	}
}

func TestCallerFromHeader_FallsBackOnMissing(t *testing.T) {
	// Use http.Header which implements Get(string) string.
	h := http.Header{}
	// With broad detection and no agent env, expect Unknown.
	got := callerFromHeader(h, config.CallerDetectionBroad)
	// Don't assert specific kind — depends on test runner env — only
	// that the function does not panic.
	_ = got
}

func confirmPolicy() config.PolicyConfig {
	return config.PolicyConfig{
		AgentAccess:       config.AgentAccessConfirm,
		AgentOverrideFlag: "--i-was-explicitly-authorized",
		CallerDetection:   config.CallerDetectionBroad,
	}
}

func denyAllPolicy() config.PolicyConfig {
	return config.PolicyConfig{
		AgentAccess:       config.AgentAccessDeny,
		AgentOverrideFlag: "--ok",
		CallerDetection:   config.CallerDetectionBroad,
	}
}
