package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type businessAccountCredentialFixture struct {
	credentialclient.Client
	provisioned string
}

func (f *businessAccountCredentialFixture) Status(context.Context, string, string) (credentialclient.CredentialStatus, error) {
	return credentialclient.CredentialStatus{Configured: true, ProviderState: "available"}, nil
}

func (f *businessAccountCredentialFixture) Resolve(context.Context, string, string) (string, error) {
	return "refresh-fixture", nil
}

func (f *businessAccountCredentialFixture) Provision(_ context.Context, request credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	f.provisioned = request.Value
	return credentialclient.ProvisionResponse{Status: "provisioned"}, nil
}

func TestBusinessAccountProxyUsesLinkedSessionAndSelectedAudience(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			if r.Method != http.MethodPost {
				t.Errorf("refresh method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"access_token":"access-fixture","refresh_token":"refresh-rotated","expires_at":"2030-01-01T00:00:00Z"}`))
		case "/api/v1/business-accounts":
			if got := r.Header.Get("Authorization"); got != "Bearer access-fixture" {
				t.Errorf("account list authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"accounts":[{"id":"acct:one","display_name":"Fixture account","billing_email":"private@example.com","role":"owner","created_at":"2026-10-30T00:00:00Z"}]}`))
		case "/api/v1/usage/business-account-summary":
			if got := r.URL.Query().Get("business_account_id"); got != "acct:one" {
				t.Errorf("selected account query = %q", got)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer access-fixture" {
				t.Errorf("usage authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"business_account_id":"acct:one","billing_period":"2026-10","tier":"pro","limits":{"ai_credits":100},"usage":{"ai_credits":20},"reserved":{"ai_credits":10},"remaining":{"ai_credits":70},"display_credits":{"ai_credits":0.02}}`))
		default:
			t.Errorf("unexpected LPBS path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	credentials := &businessAccountCredentialFixture{}
	resolver := &credentialclient.ConsumerSessionResolver{
		Credentials: credentials, LPBSBaseURL: server.URL, HTTPClient: server.Client(),
		Now: func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	svc := &Server{subscriptionResolver: resolver}
	accounts := httptest.NewRecorder()
	accountsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/business-accounts", nil)
	accountsRequest.RemoteAddr = "127.0.0.1:1234"
	svc.businessAccountsHandler(accounts, accountsRequest)
	if accounts.Code != http.StatusOK {
		t.Fatalf("account list status = %d body=%q", accounts.Code, accounts.Body.String())
	}
	if strings.Contains(accounts.Body.String(), "private@example.com") || !strings.Contains(accounts.Body.String(), "Fixture account") {
		t.Fatalf("account list projection should include display facts and omit billing PII: %s", accounts.Body.String())
	}
	if credentials.provisioned != "refresh-rotated" {
		t.Fatalf("rotated refresh token persisted = %q", credentials.provisioned)
	}

	usage := httptest.NewRecorder()
	usageRequest := httptest.NewRequest(http.MethodGet, "/api/v1/business-account-usage?business_account_id=acct%3Aone", nil)
	usageRequest.RemoteAddr = "127.0.0.1:1234"
	svc.businessAccountUsageHandler(usage, usageRequest)
	if usage.Code != http.StatusOK {
		t.Fatalf("account usage status = %d body=%q", usage.Code, usage.Body.String())
	}
	if !strings.Contains(usage.Body.String(), `"business_account_id":"acct:one"`) || !strings.Contains(usage.Body.String(), `"reserved":{"ai_credits":10}`) {
		t.Fatalf("account usage did not preserve selected audience and pending holds: %s", usage.Body.String())
	}
}

func TestBusinessAccountUsageProxyRequiresExplicitAccount(t *testing.T) {
	svc := &Server{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/business-account-usage", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()
	svc.businessAccountUsageHandler(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing account status = %d, want 400", response.Code)
	}
}

func TestBusinessAccountProxyRejectsCrossOriginRequest(t *testing.T) {
	svc := &Server{}
	request := httptest.NewRequest(http.MethodGet, "http://console.local/api/v1/business-accounts", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	request.Host = "console.local"
	request.Header.Set("Origin", "https://attacker.local")
	response := httptest.NewRecorder()
	svc.businessAccountsHandler(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin account request status = %d, want 403", response.Code)
	}
}
