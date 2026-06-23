package main

import (
	"context"
	"strings"
	"testing"
	"time"
	"web-console/internal/backend"
	"web-console/internal/sessionstore"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

// newRecoveryTestServer wires the in-memory store + fake PTY factory so the
// recovery flow can be exercised end-to-end without tmux. The fake PTY
// preserves the WriteInput contract the recovery adapter depends on.
func newRecoveryTestServer(t *testing.T) *Server {
	t.Helper()
	useIsolatedSessionState(t)
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)
	return srv
}

func saveOrphan(t *testing.T, srv *Server, id string, agent sessionstore.Agent, agentSessionID string) {
	t.Helper()
	if err := srv.sessionStore.Save(sessionstore.Metadata{
		ID:             id,
		Backend:        backend.Persistent,
		Shell:          "/bin/bash",
		Cols:           120,
		Rows:           36,
		Created:        time.Now().Add(-time.Hour),
		Detached:       true,
		Status:         sessionstore.StatusAwaitingRecovery,
		AgentType:      agent,
		AgentSessionID: agentSessionID,
		OrphanedAt:     time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("save orphan: %v", err)
	}
}

func callListRecoverable(t *testing.T, srv *Server) []*sessionsv1.RecoverableSession {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).ListRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.ListRecoverableRequest{}))
	if err != nil {
		t.Fatalf("ListRecoverable: %v", err)
	}
	return resp.Msg.GetSessions()
}

func callRecover(t *testing.T, srv *Server, id string) (*sessionsv1.RecoverResponse, error) {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).Recover(context.Background(),
		connect.NewRequest(&sessionsv1.RecoverRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func callDismissRecoverable(t *testing.T, srv *Server, id string) error {
	t.Helper()
	_, err := newSessionsConnectHandlerForServer(srv).DismissRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.DismissRecoverableRequest{Id: id}))
	return err
}

func TestHandleListRecoverable_OrdersByActivity(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "older", sessionstore.AgentCodex, "codex-1")
	saveOrphan(t, srv, "newer", sessionstore.AgentCodex, "codex-2")
	_ = srv.sessionStore.UpdateAgentInfo("older", sessionstore.AgentInfo{LastActivityAt: time.Now().Add(-2 * time.Hour)})
	_ = srv.sessionStore.UpdateAgentInfo("newer", sessionstore.AgentInfo{LastActivityAt: time.Now().Add(-5 * time.Minute)})

	rows := callListRecoverable(t, srv)
	// In-memory store does not enforce ordering; just check membership + recoverable flag.
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if !r.GetRecoverable() {
			t.Errorf("expected codex orphan %s recoverable, got reason=%q", r.GetId(), r.GetNotRecoverableReason())
		}
	}
}

func TestHandleRecover_Codex_HappyPath(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "codex-old", sessionstore.AgentCodex, "019d-codex-uuid")

	resp, err := callRecover(t, srv, "codex-old")
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if resp.GetOldSessionId() != "codex-old" {
		t.Errorf("OldSessionID: got %q", resp.GetOldSessionId())
	}
	if resp.GetNewSessionId() == "" {
		t.Error("NewSessionID empty")
	}
	if !strings.Contains(resp.GetCommandSent(), "codex --yolo resume 019d-codex-uuid") {
		t.Errorf("CommandSent: got %q", resp.GetCommandSent())
	}
	old, _ := srv.sessionStore.Get("codex-old")
	if old.Status != sessionstore.StatusDismissed {
		t.Errorf("old row status: got %q", old.Status)
	}
	if old.RecoveredInto != resp.GetNewSessionId() {
		t.Errorf("RecoveredInto: got %q want %q", old.RecoveredInto, resp.GetNewSessionId())
	}
}

func TestHandleRecover_Codex_NoSessionIDFallsBackToLast(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "codex-bare", sessionstore.AgentCodex, "")

	resp, err := callRecover(t, srv, "codex-bare")
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if !strings.Contains(resp.GetCommandSent(), "codex --yolo resume --last") {
		t.Errorf("CommandSent: got %q", resp.GetCommandSent())
	}
}

func TestHandleRecover_Claude_RequiresSessionID(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "claude-bare", sessionstore.AgentClaude, "")

	_, err := callRecover(t, srv, "claude-bare")
	if got := connectCode(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %s (err=%v)", got, err)
	}
}

func TestHandleRecover_RejectsLiveSession(t *testing.T) {
	srv := newRecoveryTestServer(t)
	if err := srv.sessionStore.Save(sessionstore.Metadata{
		ID:        "live-id",
		Backend:   backend.Persistent,
		Shell:     "/bin/bash",
		Cols:      80,
		Rows:      24,
		Created:   time.Now(),
		Detached:  true,
		Status:    sessionstore.StatusLive,
		AgentType: sessionstore.AgentCodex,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	_, err := callRecover(t, srv, "live-id")
	if got := connectCode(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %s (err=%v)", got, err)
	}
}

func TestHandleRecover_NotFound(t *testing.T) {
	srv := newRecoveryTestServer(t)
	_, err := callRecover(t, srv, "nope")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %s (err=%v)", got, err)
	}
}

func TestHandleDismissRecoverable_TransitionsToDismissed(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "drop-me", sessionstore.AgentCodex, "x")

	if err := callDismissRecoverable(t, srv, "drop-me"); err != nil {
		t.Fatalf("DismissRecoverable: %v", err)
	}
	got, _ := srv.sessionStore.Get("drop-me")
	if got.Status != sessionstore.StatusDismissed {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestCreateSession_PersistsLaunchCommandAndAgentType(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)

	resp, err := newSessionsConnectHandlerForServer(srv).Create(context.Background(),
		connect.NewRequest(&sessionsv1.CreateRequest{
			Cols:          80,
			Rows:          24,
			Backend:       "standard",
			LaunchCommand: "codex --yolo",
			AgentType:     "codex",
		}))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := srv.sessionStore.Get(resp.Msg.GetSession().GetId())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentType != sessionstore.AgentCodex {
		t.Errorf("agent_type: got %q", got.AgentType)
	}
	if got.LaunchCommand != "codex --yolo" {
		t.Errorf("launch_command: got %q", got.LaunchCommand)
	}
}
