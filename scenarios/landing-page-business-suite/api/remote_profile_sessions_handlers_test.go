package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestHandleAdminListIncomingRemoteProfileSessions(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at, ip_address, user_agent, created_at, last_activity)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "remote-session-1", "admin@localhost", time.Now().Add(1*time.Hour), "127.0.0.1",
		buildRemoteProfileSessionUserAgent(RemoteProfileSessionMetadata{
			ConnectorID: "connector-1",
			ProfileTag:  "prod",
			Origin:      "local-dev",
		}),
		time.Now().Add(-5*time.Minute), time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("insert admin session: %v", err)
	}
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_sessions`).Scan(&total); err != nil {
		t.Fatalf("count admin_sessions: %v", err)
	}
	if total < 1 {
		t.Fatalf("expected inserted admin session row")
	}
	var storedUA string
	if err := db.QueryRow(`SELECT user_agent FROM admin_sessions WHERE id = $1`, "remote-session-1").Scan(&storedUA); err != nil {
		t.Fatalf("read user_agent: %v", err)
	}
	if _, ok := parseRemoteProfileSessionUserAgent(storedUA); !ok {
		t.Fatalf("stored user agent should parse, got %q", storedUA)
	}

	handler := handleAdminListIncomingRemoteProfileSessions(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/remote-profile-sessions?connector_id=connector-1", nil)
	resp := httptest.NewRecorder()
	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Sessions []IncomingRemoteProfileSessionResponse `json:"sessions"`
	}
	decodeJSONResponse(t, resp.Body.Bytes(), &payload)
	if len(payload.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d body=%s", len(payload.Sessions), resp.Body.String())
	}
	if payload.Sessions[0].ConnectorID != "connector-1" {
		t.Fatalf("unexpected connector id: %q", payload.Sessions[0].ConnectorID)
	}
}

func TestHandleAdminRevokeIncomingRemoteProfileSession(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at, ip_address, user_agent, created_at, last_activity)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "remote-session-2", "admin@localhost", time.Now().Add(1*time.Hour), "127.0.0.1",
		buildRemoteProfileSessionUserAgent(RemoteProfileSessionMetadata{ConnectorID: "connector-2"}),
		time.Now().Add(-5*time.Minute), time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("insert admin session: %v", err)
	}

	handler := handleAdminRevokeIncomingRemoteProfileSession(db)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/remote-profile-sessions/remote-session-2", nil)
	req = mux.SetURLVars(req, map[string]string{"session_id": "remote-session-2"})
	resp := httptest.NewRecorder()
	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestHandleAdminRevokeIncomingRemoteProfileSession_MissingID(t *testing.T) {
	db := setupTestDB(t)

	handler := handleAdminRevokeIncomingRemoteProfileSession(db)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/remote-profile-sessions/", nil)
	req = mux.SetURLVars(req, map[string]string{"session_id": ""})
	resp := httptest.NewRecorder()
	handler(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", resp.Code, resp.Body.String())
	}
}
