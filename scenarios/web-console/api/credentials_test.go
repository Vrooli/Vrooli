package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"web-console/internal/events"

	_ "modernc.org/sqlite"
)

type credentialHandlerClient struct {
	refs        []credentialclient.CredentialRef
	provision   credentialclient.ProvisionRequest
	deleteID    string
	deleteField string
}

func (c *credentialHandlerClient) Provision(_ context.Context, request credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	c.provision = request
	return credentialclient.ProvisionResponse{Identity: request.Identity, Field: request.Field}, nil
}
func (c *credentialHandlerClient) Resolve(context.Context, string, string) (string, error) {
	return "", nil
}
func (c *credentialHandlerClient) Delete(_ context.Context, identity, field string) error {
	c.deleteID, c.deleteField = identity, field
	return nil
}
func (c *credentialHandlerClient) Status(context.Context, string, string) (credentialclient.CredentialStatus, error) {
	return credentialclient.CredentialStatus{}, nil
}
func (c *credentialHandlerClient) List(context.Context) ([]credentialclient.CredentialRef, error) {
	return c.refs, nil
}
func (c *credentialHandlerClient) Doctor(context.Context) (credentialclient.DoctorResponse, error) {
	return credentialclient.DoctorResponse{}, nil
}
func (c *credentialHandlerClient) KeyringInspect(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}
func (c *credentialHandlerClient) KeyringRepair(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}
func (c *credentialHandlerClient) RecoveryExport(context.Context, credentialclient.RecoveryExportRequest) (credentialclient.RecoveryExportResponse, error) {
	return credentialclient.RecoveryExportResponse{}, nil
}
func (c *credentialHandlerClient) RecoveryVerify(context.Context, credentialclient.RecoveryVerifyRequest) (credentialclient.RecoveryVerifyResponse, error) {
	return credentialclient.RecoveryVerifyResponse{}, nil
}
func (c *credentialHandlerClient) RecoveryRestore(context.Context, credentialclient.RecoveryRestoreRequest) error {
	return nil
}
func (c *credentialHandlerClient) StoreStatus(context.Context) (credentialclient.StoreStatus, error) {
	return credentialclient.StoreStatus{}, nil
}

func TestCredentialProvisionStoresOnlyDeclaredCredential(t *testing.T) {
	client := &credentialHandlerClient{refs: []credentialclient.CredentialRef{{LogicalID: webConsoleOpenRouterIdentity, Field: webConsoleOpenRouterField}}}
	server := &Server{credentialClient: client}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials/provision", strings.NewReader(`{"identity":"vrooli/openrouter","field":"api-key","value":"secret-value"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	server.credentialProvisionHandler(rec, req)
	if rec.Code != http.StatusCreated || strings.Contains(rec.Body.String(), "secret-value") {
		t.Fatalf("status/body = %d/%q; credential must not be returned", rec.Code, rec.Body.String())
	}
	if client.provision.Value != "secret-value" {
		t.Fatalf("authority received value %q", client.provision.Value)
	}
}

func TestCredentialProvisionRejectsUndeclaredCredential(t *testing.T) {
	client := &credentialHandlerClient{}
	server := &Server{credentialClient: client}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials/provision", strings.NewReader(`{"identity":"other","field":"token","value":"secret-value"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	server.credentialProvisionHandler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if client.provision.Value != "" {
		t.Fatal("undeclared credential reached authority")
	}
}

func TestCredentialProvisionRejectsCrossOriginBeforeAuthority(t *testing.T) {
	client := &credentialHandlerClient{refs: []credentialclient.CredentialRef{{LogicalID: webConsoleOpenRouterIdentity, Field: webConsoleOpenRouterField}}}
	server := &Server{credentialClient: client}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials/provision", strings.NewReader(`{"identity":"vrooli/openrouter","field":"api-key","value":"secret-value"}`))
	req.RemoteAddr = "192.0.2.20:1234"
	req.Host = "console.example"
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	server.credentialProvisionHandler(rec, req)
	if rec.Code != http.StatusForbidden || client.provision.Value != "" {
		t.Fatalf("cross-origin request status=%d provision=%q", rec.Code, client.provision.Value)
	}
}

func TestJourneyHandlerIsLoopbackOnlyAndMetadataOnly(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/internal/monetization/journey?operation=signin_shared_session", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	(&Server{}).journeyHandler(recorder, request)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "token") || !strings.Contains(recorder.Body.String(), "web-console") || !strings.Contains(recorder.Body.String(), "business_suite") {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/internal/monetization/journey?operation=signin_shared_session", nil)
	request.RemoteAddr = "192.0.2.20:1234"
	request.Host = "console.example"
	request.Header.Set("Origin", "https://evil.example")
	recorder = httptest.NewRecorder()
	(&Server{}).journeyHandler(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestResolveLPBSIdentityUsesAuthenticatedAuthorityResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"user":{"email":"verified@example.com"}}`))
	}))
	defer server.Close()
	identity, err := resolveLPBSIdentity(context.Background(), server.URL, "access-token")
	if err != nil || identity != "verified@example.com" {
		t.Fatalf("resolveLPBSIdentity() = %q, %v", identity, err)
	}
}

func TestActivationEventsAreEmittedOnceAcrossRepeatedCalls(t *testing.T) {
	db, err := sql.Open("sqlite", "file:activation-events-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	routed := database.NewFromPrimary(db)
	logger := events.NewLogger(20)
	server := &Server{db: routed, events: logger}
	if err := ensureActivationEvents(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	server.emitActivationOnce(context.Background(), activationFirstCommandRun)
	server.emitActivationOnce(context.Background(), activationFirstCommandRun)
	if got := logger.Count(); got != 1 {
		t.Fatalf("event count = %d, want 1", got)
	}
}
