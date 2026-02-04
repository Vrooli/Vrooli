package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAdminRemoteProfiles_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`DELETE FROM remote_profiles`); err != nil {
		t.Fatalf("failed to clear remote_profiles: %v", err)
	}

	svc := newRemoteProfileServiceForTest(db, nil)
	sessionMgr := initSessionManager()
	server := &Server{db: db, sessionManager: sessionMgr, remoteProfileService: svc}

	createReq := []byte(`{"tag":"prod","label":"Production","api_base":"https://example.com/api/v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/remote-profiles", bytes.NewBuffer(createReq))
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(handleAdminCreateRemoteProfile(server))(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/remote-profiles", nil)
	attachAdminSession(t, listReq, defaultAdminEmail)
	listResp := httptest.NewRecorder()

	server.requireAdmin(handleAdminListRemoteProfiles(server.remoteProfileService))(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	var payload struct {
		Profiles []RemoteProfile `json:"profiles"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(payload.Profiles))
	}
	if payload.Profiles[0].Tag != "prod" {
		t.Fatalf("expected tag prod, got %s", payload.Profiles[0].Tag)
	}
}
