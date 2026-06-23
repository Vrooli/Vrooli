package main

import (
	"testing"
	"web-console/internal/policy"

	"connectrpc.com/connect"
)

// --- HTTP handler tests ---
// Sweeper tests that touch private fields (running, interval, stopCh, etc.)
// live alongside the implementation in session/session_policy_test.go.

// [REQ:P1-001a] Expiration Policy Engine — Get RPC
func TestHandleGetPolicy(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	view, err := callGetPolicy(t, srv, sess.ID)
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if view.GetPolicy().GetMode() != string(policy.Never) {
		t.Errorf("default should be never, got %s", view.GetPolicy().GetMode())
	}
	if view.GetHasExpiry() {
		t.Error("never policy should not have expiry")
	}
}

func TestHandleGetPolicy_NotFound(t *testing.T) {
	srv := newFakeTestServer()
	_, err := callGetPolicy(t, srv, "nonexistent")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s", got)
	}
}

// [REQ:P1-001a] Expiration Policy Engine — Update RPC
func TestHandleUpdatePolicy(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	view, err := callUpdatePolicy(t, srv, sess.ID, "preset", "8h")
	if err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}
	if view.GetPolicy().GetMode() != string(policy.Preset) || view.GetPolicy().GetDuration() != "8h" {
		t.Errorf("expected preset/8h, got %s/%s", view.GetPolicy().GetMode(), view.GetPolicy().GetDuration())
	}
	if !view.GetHasExpiry() {
		t.Error("preset policy should have expiry")
	}
	if view.GetExpiresAt() == "" || view.GetTtlSeconds() <= 0 {
		t.Errorf("preset policy should have expires_at and positive ttl, got %q / %v",
			view.GetExpiresAt(), view.GetTtlSeconds())
	}
}

func TestHandleUpdatePolicy_InvalidMode(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(sess.ID) }()

	_, err = callUpdatePolicy(t, srv, sess.ID, "bad", "")
	if got := connectCode(err); got != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %s (err=%v)", got, err)
	}
}

func TestHandleUpdatePolicy_NotFound(t *testing.T) {
	srv := newFakeTestServer()
	_, err := callUpdatePolicy(t, srv, "nonexistent", "never", "")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %s", got)
	}
}

// Verify policy is included in session response
func TestSessionResponse_IncludesPolicy(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := callCreate(t, srv, 80, 24, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.GetPolicy().GetMode() != string(policy.Never) {
		t.Errorf("new session should have never policy, got %s", sess.GetPolicy().GetMode())
	}
	_ = srv.sessions.Delete(sess.GetId())
}

// Verify error catalog includes invalid_policy
func TestErrorCatalog_InvalidPolicy(t *testing.T) {
	ae, ok := errorCatalog["invalid_policy"]
	if !ok {
		t.Fatal("error catalog missing invalid_policy")
	}
	if ae.Category != "validation" {
		t.Errorf("expected category=validation, got %s", ae.Category)
	}
}

// Verify the default policy mode on new sessions
func TestDefaultPolicy(t *testing.T) {
	p := policy.Default()
	if p.Mode != policy.Never {
		t.Errorf("default policy should be never, got %s", p.Mode)
	}
	if p.Duration != "" {
		t.Errorf("default policy should have empty duration, got %s", p.Duration)
	}
}
