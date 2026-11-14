package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	credentials "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials/credentials_v1connect"
)

func TestCredentialBrokerConnectContractUsesAuthorityAndRedactsResults(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("received Authorization: Bearer broker-secret"))
	}))
	defer target.Close()

	handlers := newPasswordManagerHandlers(nil)
	item, err := handlers.manager.createItem(context.Background(), "local", "personal", "local-owner", "typed-idem", createItemInput{
		Type: "api_credential", Name: "Typed broker fixture", Fields: map[string]string{"token": "broker-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := "agent-run:typed-fixture"
	grant, err := handlers.manager.createGrant(context.Background(), "local", "local-owner", createGrantInput{
		VaultID: "personal", ItemID: item.ID, PrincipalType: "agent", PrincipalID: principal, SelectorMode: "current_snapshot",
		Operations: []string{"use"}, ExpiresAt: time.Now().UTC().Add(time.Hour), Target: target.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	path, transport := credentialsconnect.NewCredentialBrokerServiceHandler(newCredentialBrokerConnectHandler(handlers))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, path) {
			transport.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := credentialsconnect.NewCredentialBrokerServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&credentials.RequestAccessRequest{GrantId: grant.ID, ItemId: item.ID, Operation: "use"})
	setTypedIdentity(request, principal)
	access, err := client.RequestAccess(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if access.Msg.GetRequestedBy() != principal || access.Msg.GetStatus() != "pending" {
		t.Fatalf("access request = %+v", access.Msg)
	}
	statusRequest := connect.NewRequest(&credentials.GetAccessRequestRequest{RequestId: access.Msg.GetId()})
	setTypedIdentity(statusRequest, principal)
	status, err := client.GetAccessRequest(context.Background(), statusRequest)
	if err != nil {
		t.Fatal(err)
	}
	if status.Msg.GetId() != access.Msg.GetId() || status.Msg.GetStatus() != "pending" {
		t.Fatalf("typed access status = %+v", status.Msg)
	}

	sessionRequest := connect.NewRequest(&credentials.CreateBrokerSessionRequest{
		GrantId: grant.ID, ItemId: item.ID, TargetOrigin: target.URL, AllowInternal: true, DurationSeconds: 60,
	})
	setTypedIdentity(sessionRequest, principal)
	session, err := client.CreateBrokerSession(context.Background(), sessionRequest)
	if err != nil {
		t.Fatal(err)
	}
	if session.Msg.GetSessionToken() == "" || session.Msg.GetSecretExposure() != "brokered" {
		t.Fatalf("session = %+v", session.Msg)
	}

	operation := connect.NewRequest(&credentials.ExecuteBrokerOperationRequest{
		SessionId: session.Msg.GetSessionId(), SessionToken: session.Msg.GetSessionToken(), Method: http.MethodGet, Path: "/health",
	})
	setTypedIdentity(operation, principal)
	result, err := client.ExecuteBrokerOperation(context.Background(), operation)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Msg.GetBody(), "broker-secret") || !strings.Contains(result.Msg.GetBody(), "[REDACTED]") {
		t.Fatalf("broker result leaked credential: %q", result.Msg.GetBody())
	}

	revoke := connect.NewRequest(&credentials.RevokeBrokerSessionRequest{SessionId: session.Msg.GetSessionId()})
	setTypedIdentity(revoke, principal)
	if _, err := client.RevokeBrokerSession(context.Background(), revoke); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialBrokerConnectUsesVerifiedAgentIdentityBoundary(t *testing.T) {
	runID := uuid.NewString()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["token"] != "agent-capability" {
			t.Fatalf("identity verification body = %+v, err=%v", body, err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "claims": map[string]string{"run_id": runID, "workspace_id": "local"}})
	}))
	defer identityServer.Close()

	handlers := newPasswordManagerHandlers(nil)
	item, err := handlers.manager.createItem(context.Background(), "local", "personal", "local-owner", "agent-boundary-idem", createItemInput{
		Type: "api_credential", Name: "Agent boundary fixture", Fields: map[string]string{"token": "boundary-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := "agent-run:" + runID
	grant, err := handlers.manager.createGrant(context.Background(), "local", "local-owner", createGrantInput{
		VaultID: "personal", ItemID: item.ID, PrincipalType: "agent", PrincipalID: principal, SelectorMode: "current_snapshot",
		Operations: []string{"use"}, ExpiresAt: time.Now().UTC().Add(time.Hour), Target: target.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	path, transport := credentialsconnect.NewCredentialBrokerServiceHandler(newCredentialBrokerConnectHandler(handlers))
	wrapped := ownerAuthBoundary(transport, nil, &agentIdentityVerifier{baseURL: identityServer.URL, client: identityServer.Client()})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, path) {
			wrapped.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := credentialsconnect.NewCredentialBrokerServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&credentials.CreateBrokerSessionRequest{
		GrantId: grant.ID, ItemId: item.ID, TargetOrigin: target.URL, AllowInternal: true, DurationSeconds: 60,
	})
	request.Header().Set("X-Agent-Identity-Token", "agent-capability")
	request.Header().Set("X-Workspace-ID", "local")
	session, err := client.CreateBrokerSession(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if session.Msg.GetSessionId() == "" || session.Msg.GetSecretExposure() != "brokered" {
		t.Fatalf("verified agent session = %+v", session.Msg)
	}
}

func setTypedIdentity[T any](request *connect.Request[T], principal string) {
	request.Header().Set("X-Workspace-ID", "local")
	request.Header().Set("X-Actor-ID", principal)
}
