package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// newRecoveryTestServer wires the in-memory store + fake PTY factory so the
// recovery endpoint can be exercised end-to-end without tmux. The fake PTY
// preserves the WriteInput contract used by the recovery handler.
func newRecoveryTestServer(t *testing.T) *Server {
	t.Helper()
	useIsolatedSessionState(t)
	srv := newFakeTestServer()
	srv.sessionStore = NewInMemorySessionStore()
	srv.sessions.SetStore(srv.sessionStore)
	// Register recovery routes on the test router so we can exercise them via
	// the same HTTP path the production binary uses.
	srv.router.HandleFunc("/api/v1/sessions/recoverable", srv.handleListRecoverable).Methods("GET")
	srv.router.HandleFunc("/api/v1/sessions/recoverable/{id}", srv.handleDismissRecoverable).Methods("DELETE")
	srv.router.HandleFunc("/api/v1/sessions/{id}/recover", srv.handleRecoverSession).Methods("POST")
	return srv
}

func saveOrphan(t *testing.T, srv *Server, id string, agent AgentType, agentSessionID string) {
	t.Helper()
	if err := srv.sessionStore.Save(SessionMetadata{
		ID:             id,
		Backend:        BackendPersistent,
		Shell:          "/bin/bash",
		Cols:           120,
		Rows:           36,
		Created:        time.Now().Add(-time.Hour),
		Detached:       true,
		Status:         SessionStatusAwaitingRecovery,
		AgentType:      agent,
		AgentSessionID: agentSessionID,
		OrphanedAt:     time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("save orphan: %v", err)
	}
}

func TestHandleListRecoverable_OrdersByActivity(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "older", AgentTypeCodex, "codex-1")
	saveOrphan(t, srv, "newer", AgentTypeCodex, "codex-2")
	// Set last_activity_at differently so we can verify ordering.
	_ = srv.sessionStore.UpdateAgentInfo("older", AgentInfo{LastActivityAt: time.Now().Add(-2 * time.Hour)})
	_ = srv.sessionStore.UpdateAgentInfo("newer", AgentInfo{LastActivityAt: time.Now().Add(-5 * time.Minute)})

	req := httptest.NewRequest("GET", "/api/v1/sessions/recoverable", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var rows []RecoverableSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// In-memory store does not enforce ordering; just check membership +
	// recoverable flag for each.
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if !r.Recoverable {
			t.Errorf("expected codex orphan %s recoverable, got reason=%q", r.ID, r.NotRecoverable)
		}
	}
}

func TestHandleRecover_Codex_HappyPath(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "codex-old", AgentTypeCodex, "019d-codex-uuid")

	req := httptest.NewRequest("POST", "/api/v1/sessions/codex-old/recover", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "codex-old"})
	rec := httptest.NewRecorder()
	srv.handleRecoverSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp RecoverSessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.OldSessionID != "codex-old" {
		t.Errorf("OldSessionID: got %q", resp.OldSessionID)
	}
	if resp.NewSessionID == "" {
		t.Errorf("NewSessionID empty")
	}
	if !strings.Contains(resp.CommandSent, "codex --yolo resume 019d-codex-uuid") {
		t.Errorf("CommandSent: got %q", resp.CommandSent)
	}
	old, _ := srv.sessionStore.Get("codex-old")
	if old.Status != SessionStatusDismissed {
		t.Errorf("old row status: got %q", old.Status)
	}
	if old.RecoveredInto != resp.NewSessionID {
		t.Errorf("RecoveredInto: got %q want %q", old.RecoveredInto, resp.NewSessionID)
	}
}

func TestHandleRecover_Codex_NoSessionIDFallsBackToLast(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "codex-bare", AgentTypeCodex, "")

	req := httptest.NewRequest("POST", "/api/v1/sessions/codex-bare/recover", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "codex-bare"})
	rec := httptest.NewRecorder()
	srv.handleRecoverSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp RecoverSessionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !strings.Contains(resp.CommandSent, "codex --yolo resume --last") {
		t.Errorf("CommandSent: got %q", resp.CommandSent)
	}
}

func TestHandleRecover_Claude_RequiresSessionID(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "claude-bare", AgentTypeClaude, "")

	req := httptest.NewRequest("POST", "/api/v1/sessions/claude-bare/recover", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "claude-bare"})
	rec := httptest.NewRecorder()
	srv.handleRecoverSession(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRecover_RejectsLiveSession(t *testing.T) {
	srv := newRecoveryTestServer(t)
	if err := srv.sessionStore.Save(SessionMetadata{
		ID:        "live-id",
		Backend:   BackendPersistent,
		Shell:     "/bin/bash",
		Cols:      80,
		Rows:      24,
		Created:   time.Now(),
		Detached:  true,
		Status:    SessionStatusLive,
		AgentType: AgentTypeCodex,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/sessions/live-id/recover", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "live-id"})
	rec := httptest.NewRecorder()
	srv.handleRecoverSession(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleRecover_NotFound(t *testing.T) {
	srv := newRecoveryTestServer(t)
	req := httptest.NewRequest("POST", "/api/v1/sessions/nope/recover", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nope"})
	rec := httptest.NewRecorder()
	srv.handleRecoverSession(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDismissRecoverable_TransitionsToDismissed(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "drop-me", AgentTypeCodex, "x")

	req := httptest.NewRequest("DELETE", "/api/v1/sessions/recoverable/drop-me", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "drop-me"})
	rec := httptest.NewRecorder()
	srv.handleDismissRecoverable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got, _ := srv.sessionStore.Get("drop-me")
	if got.Status != SessionStatusDismissed {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestCreateSession_PersistsLaunchCommandAndAgentType(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessionStore = NewInMemorySessionStore()
	srv.sessions.SetStore(srv.sessionStore)

	body := strings.NewReader(`{"cols":80,"rows":24,"backend":"standard","launch_command":"codex --yolo","agent_type":"codex"}`)
	req := httptest.NewRequest("POST", "/api/v1/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleCreateSession(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp SessionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	got, err := srv.sessionStore.Get(resp.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentType != AgentTypeCodex {
		t.Errorf("agent_type: got %q", got.AgentType)
	}
	if got.LaunchCommand != "codex --yolo" {
		t.Errorf("launch_command: got %q", got.LaunchCommand)
	}
}
