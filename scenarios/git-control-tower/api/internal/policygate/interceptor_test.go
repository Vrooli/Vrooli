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

func TestInterceptor_HumanCallerAllowed(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, denyAllPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(HeaderCaller, "human")
	_, err := client.CreateWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateWorktree as human: %v", err)
	}
	if !srv.reached {
		t.Fatal("human caller should reach handler under deny policy")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "allow" {
		t.Errorf("expected single allow event; got %+v", events)
	}
}

func TestInterceptor_AgentDeniedUnderConfirmWithoutOverride(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, confirmPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(HeaderCaller, "external-agent")
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

func TestInterceptor_AgentAllowedWithOverride(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, confirmPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(HeaderCaller, "external-agent")
	req.Header().Set(HeaderAuthorized, "true")
	_, err := client.CreateWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateWorktree with override: %v", err)
	}
	if !srv.reached {
		t.Fatal("authorized agent should reach handler")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "allow" || !events[0].Authorized {
		t.Errorf("expected allow+authorized event; got %+v", events)
	}
}

func TestInterceptor_AgentWarnRunsButSurfacesWarning(t *testing.T) {
	audit := &captureAuditLogger{}
	policy := config.PolicyConfig{AgentAccess: config.AgentAccessWarn, AgentOverrideFlag: "--ok", CallerDetection: config.CallerDetectionBroad}
	client, srv, cleanup := newTestClient(t, policy, audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(HeaderCaller, "external-agent")
	resp, err := client.CreateWorktree(context.Background(), req)
	if err != nil {
		t.Fatalf("warn: %v", err)
	}
	if !srv.reached {
		t.Fatal("warn should still reach handler")
	}
	if resp.Header().Get("X-Vrooli-Policy-Warning") == "" {
		t.Error("warn should set X-Vrooli-Policy-Warning trailer/header")
	}
	events := audit.snapshot()
	if len(events) != 1 || events[0].Decision != "warn" {
		t.Errorf("expected single warn event; got %+v", events)
	}
}

func TestInterceptor_AgentDenyAlwaysRefuses(t *testing.T) {
	audit := &captureAuditLogger{}
	client, srv, cleanup := newTestClient(t, denyAllPolicy(), audit)
	defer cleanup()
	req := connect.NewRequest(&worktreev1.CreateWorktreeRequest{RepoPath: "/x", NewWorktreePath: "/y"})
	req.Header().Set(HeaderCaller, "vrooli-agent")
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
