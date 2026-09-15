package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestAgentIdentityVerifierUsesLiveRunVerification(t *testing.T) {
	runID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/identity/verify" || r.Method != http.MethodPost {
			t.Fatalf("verification request = %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["token"] != "opaque-run-token" {
			t.Fatalf("verification body = %+v, err=%v", body, err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": map[string]string{"run_id": runID, "workspace_id": "workspace-a"}})
	}))
	defer server.Close()

	identity, err := (&agentIdentityVerifier{baseURL: server.URL, client: server.Client()}).Verify(t.Context(), "opaque-run-token")
	if err != nil {
		t.Fatal(err)
	}
	if identity.PrincipalID != "agent-run:"+runID || identity.RunID != runID {
		t.Fatalf("identity = %+v", identity)
	}
	if identity.WorkspaceID != "workspace-a" {
		t.Fatalf("workspace = %q, want workspace-a", identity.WorkspaceID)
	}
}

func TestOwnerAuthBoundaryLimitsVerifiedAgentsToAgentRoutes(t *testing.T) {
	runID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": map[string]string{"run_id": runID, "workspace_id": "workspace-a"}})
	}))
	defer server.Close()
	t.Setenv("SECRETS_MANAGER_OWNER_TOKEN", "")

	verifier := &agentIdentityVerifier{baseURL: server.URL, client: server.Client()}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspace, actor := requestIdentity(r)
		if workspace != "workspace-a" || actor != "agent-run:"+runID {
			t.Errorf("identity = %q/%q", workspace, actor)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := ownerAuthBoundary(next, nil, verifier)

	allowed := httptest.NewRequest(http.MethodPost, "/api/v1/broker/sessions", nil)
	allowed.Header.Set("X-Agent-Identity-Token", "opaque-run-token")
	allowed.Header.Set("X-Workspace-ID", "workspace-a")
	allowedResponse := httptest.NewRecorder()
	handler.ServeHTTP(allowedResponse, allowed)
	if allowedResponse.Code != http.StatusNoContent {
		t.Fatalf("allowed status = %d, want %d", allowedResponse.Code, http.StatusNoContent)
	}

	denied := httptest.NewRequest(http.MethodPost, "/api/v1/grants", nil)
	denied.Header.Set("X-Agent-Identity-Token", "opaque-run-token")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("management status = %d, want %d", deniedResponse.Code, http.StatusForbidden)
	}
}

func TestAgentIdentityVerifierRejectsWorkspaceMismatch(t *testing.T) {
	runID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": map[string]string{"run_id": runID, "workspace_id": "workspace-a"}})
	}))
	defer server.Close()
	verifier := &agentIdentityVerifier{baseURL: server.URL, client: server.Client()}
	if _, err := verifier.VerifyForWorkspace(t.Context(), "opaque-run-token", "workspace-b"); !errors.Is(err, errAgentIdentityInvalid) {
		t.Fatalf("mismatched workspace error = %v, want invalid identity", err)
	}
	if _, err := verifier.VerifyForWorkspace(t.Context(), "opaque-run-token", "workspace-a"); err != nil {
		t.Fatalf("matching workspace rejected: %v", err)
	}
}

func TestOwnerAuthBoundaryAllowsDevRoutingControlPlane(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_OWNER_TOKEN", "")
	called := false
	handler := ownerAuthBoundary(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}), nil, nil)

	request := httptest.NewRequest(http.MethodPost, "/vrooli.dev_routing.v1.routing.RoutingService/InstallTestPool", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || !called {
		t.Fatalf("dev-routing status/call = %d/%v, want %d/true", response.Code, called, http.StatusNoContent)
	}
}
