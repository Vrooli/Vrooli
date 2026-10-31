package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"web-console/internal/events"

	_ "modernc.org/sqlite"
)

type credentialHandlerClient struct {
	refs        []credentialclient.CredentialRef
	status      credentialclient.CredentialStatus
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
	return c.status, nil
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

func TestCredentialDeleteRejectsUndeclaredCredential(t *testing.T) {
	client := &credentialHandlerClient{}
	server := &Server{credentialClient: client}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/credentials/provision", strings.NewReader(`{"identity":"other","field":"token"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	server.credentialDeleteHandler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if client.deleteID != "" || client.deleteField != "" {
		t.Fatalf("undeclared credential was deleted: %s/%s", client.deleteID, client.deleteField)
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

func TestSubscriptionAuthorizationURLUsesConfiguredIssuer(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/subscription/authorization", nil)
	request.RemoteAddr = "192.0.2.20:1234"
	request.Host = "console.example"
	request.Header.Set("Origin", "http://console.example")
	response := httptest.NewRecorder()
	(&Server{subscriptionResolver: &credentialclient.ConsumerSessionResolver{LPBSBaseURL: "https://billing.example/"}}).subscriptionAuthorizationURL(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), `"authorization_url":"https://billing.example/auth/login"`) {
		t.Fatalf("status/body = %d/%q", response.Code, response.Body.String())
	}
}

func TestSubscriptionAuthorizationExchangeKeepsRefreshTokenServerSide(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/token" {
			t.Fatalf("LPBS path = %q", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["code"] != "one-use" || body["code_verifier"] != "verifier" || body["redirect_uri"] != "http://127.0.0.1:3210/" {
			t.Fatalf("LPBS exchange body = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "short-lived", "refresh_token": "durable-secret", "expires_at": expiresAt})
	}))
	defer lpbs.Close()

	credentials := &credentialHandlerClient{}
	resolver := &credentialclient.ConsumerSessionResolver{Credentials: credentials, LPBSBaseURL: lpbs.URL, HTTPClient: lpbs.Client()}
	server := &Server{subscriptionResolver: resolver}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/subscription/session/exchange", strings.NewReader(`{"code":"one-use","code_verifier":"verifier","redirect_uri":"http://127.0.0.1:3210/"}`))
	request.RemoteAddr = "192.0.2.20:1234"
	request.Host = "console.example"
	request.Header.Set("Origin", "http://console.example")
	response := httptest.NewRecorder()
	server.subscriptionAuthorizationExchange(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || strings.Contains(response.Body.String(), "durable-secret") {
		t.Fatalf("status/body = %d/%q", response.Code, response.Body.String())
	}
	if credentials.provision.Identity != credentialclient.SubscriptionIdentity || credentials.provision.Field != credentialclient.SubscriptionField || credentials.provision.Value != "durable-secret" {
		t.Fatalf("stored subscription credential = %+v", credentials.provision)
	}
	if !strings.Contains(response.Body.String(), `"access_token":"short-lived"`) {
		t.Fatalf("response did not return the short-lived access token: %q", response.Body.String())
	}
}

func TestSubscriptionAuthorizationExchangeRejectsCrossOriginBeforeLPBS(t *testing.T) {
	called := false
	lpbs := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer lpbs.Close()
	credentials := &credentialHandlerClient{}
	resolver := &credentialclient.ConsumerSessionResolver{Credentials: credentials, LPBSBaseURL: lpbs.URL, HTTPClient: lpbs.Client()}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/subscription/session/exchange", strings.NewReader(`{"code":"one-use","code_verifier":"verifier","redirect_uri":"http://127.0.0.1:3210/"}`))
	request.RemoteAddr = "192.0.2.20:1234"
	request.Host = "console.example"
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	(&Server{subscriptionResolver: resolver}).subscriptionAuthorizationExchange(response, request)
	if response.Code != http.StatusForbidden || called || credentials.provision.Value != "" {
		t.Fatalf("cross-origin exchange status=%d called=%v credential=%q", response.Code, called, credentials.provision.Value)
	}
}

func TestJourneyHandlerIsLoopbackOnlyAndMetadataOnly(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/internal/monetization/journey?operation=signin_shared_session", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	monetization.JourneyModule{Deps: monetization.JourneyDeps{AppKey: "web-console", BundleKey: "business_suite"}}.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "token") || !strings.Contains(recorder.Body.String(), "web-console") || !strings.Contains(recorder.Body.String(), "business_suite") || !strings.Contains(recorder.Body.String(), "unsupported:") {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/internal/monetization/journey?operation=signin_shared_session", nil)
	request.RemoteAddr = "192.0.2.20:1234"
	request.Host = "console.example"
	request.Header.Set("Origin", "https://evil.example")
	recorder = httptest.NewRecorder()
	monetization.JourneyModule{Deps: monetization.JourneyDeps{AppKey: "web-console", BundleKey: "business_suite"}}.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("cross-origin status = %d, want %d", recorder.Code, http.StatusNotFound)
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

func TestCommercialContextFailsClosedWhenAccountIsUnavailable(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/commercial-context?placement=integrations", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	(&Server{}).commercialContextHandler(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || strings.Contains(recorder.Body.String(), "entitlement") {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
}

func TestConnectionsHandlerReturnsMetadataWithoutSecret(t *testing.T) {
	client := &credentialHandlerClient{status: credentialclient.CredentialStatus{Configured: true, Provider: "credential-authority"}}
	server := &Server{credentialClient: client}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/connections", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	server.connectionsHandler(recorder, request)
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, "OpenRouter API credential") || !strings.Contains(body, "supported_actions") || !strings.Contains(body, "test") || !strings.Contains(body, "delete") || strings.Contains(body, "secret") || strings.Contains(body, "credential-authority") {
		t.Fatalf("status/body = %d/%q", recorder.Code, body)
	}
}

func TestConnectionsHandlerOmitsUnconfiguredCredential(t *testing.T) {
	server := &Server{credentialClient: &credentialHandlerClient{}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/connections", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	server.connectionsHandler(recorder, request)
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "OpenRouter") {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
}

func TestConnectionsHandlerDoesNotCollapseAuthorityOutageIntoEmptyState(t *testing.T) {
	server := &Server{credentialClient: &credentialHandlerClient{status: credentialclient.CredentialStatus{
		ProviderState:  "unavailable",
		ProviderDetail: "sensitive backend diagnostic",
	}}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/connections", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	server.connectionsHandler(recorder, request)
	if recorder.Code != http.StatusBadGateway || strings.Contains(recorder.Body.String(), "sensitive backend diagnostic") {
		t.Fatalf("status/body = %d/%q", recorder.Code, recorder.Body.String())
	}
}

func TestProjectHubStatusStaysWithinConsumerContract(t *testing.T) {
	for status, want := range map[commonv1.ConnectionStatus]string{
		commonv1.ConnectionStatus_CONNECTION_STATUS_UNSPECIFIED:          "unknown",
		commonv1.ConnectionStatus_CONNECTION_STATUS_UNKNOWN:              "unknown",
		commonv1.ConnectionStatus_CONNECTION_STATUS_CONNECTED:            "connected",
		commonv1.ConnectionStatus_CONNECTION_STATUS_PROVIDER_OUTAGE:      "needs_attention",
		commonv1.ConnectionStatus_CONNECTION_STATUS_PROVIDER_UNAVAILABLE: "needs_attention",
	} {
		if got := projectHubStatus(status); got != want {
			t.Errorf("projectHubStatus(%v) = %q, want %q", status, got, want)
		}
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
