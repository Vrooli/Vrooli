package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

type credentialHandlerStore struct{ value string }

func (s *credentialHandlerStore) Put(_, _ string, value string) error { s.value = value; return nil }
func (s *credentialHandlerStore) Get(_, _ string) (string, error)     { return s.value, nil }
func (s *credentialHandlerStore) Delete(_, _ string) error            { s.value = ""; return nil }

func testCredentialClient(t *testing.T) credentialclient.Client {
	t.Helper()
	authority, err := credentialauthority.NewAuthority(&credentialHandlerStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{
		Authority: authority,
		Descriptors: func() ([]credentialclient.CredentialRef, error) {
			return []credentialclient.CredentialRef{{LogicalID: "vrooli/openrouter", Field: "api-key"}}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestCredentialProvisionHandlerAcceptsDeclaredIdentityWithoutReturningSecret(t *testing.T) {
	secret := "must-not-be-returned"
	body, err := json.Marshal(credentialProvisionRequest{Identity: "vrooli/openrouter", Field: "api-key", Value: secret})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/credentials/provision", bytes.NewReader(body))
	req.Host = "127.0.0.1:18080"
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	credentialProvisionHandler(testCredentialClient(t))(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	response, _ := io.ReadAll(res.Body)
	if strings.Contains(string(response), secret) {
		t.Fatalf("response returned credential value: %s", response)
	}
	if !strings.Contains(string(response), `"configured":true`) {
		t.Fatalf("response = %s, want configured status", response)
	}
}

func TestCredentialProvisionHandlerRejectsUndeclaredIdentity(t *testing.T) {
	body, err := json.Marshal(credentialProvisionRequest{Identity: "vrooli/other", Field: "api-key", Value: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/credentials/provision", bytes.NewReader(body))
	req.Host = "127.0.0.1:18080"
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	credentialProvisionHandler(testCredentialClient(t))(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestSubscriptionSessionRejectsCrossOriginRemoteRequest(t *testing.T) {
	body := strings.NewReader(`{"refresh_token":"refresh-secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/subscription/session", body)
	req.Host = "bas.example.test"
	req.Header.Set("Origin", "https://attacker.example.test")
	res := httptest.NewRecorder()
	monetization.NewSessionModule(testCredentialClient(t), lpbsAccountIdentity, lpbsAccountField).Provision(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestSubscriptionSessionAcceptsLoopbackWithoutOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/subscription/session", strings.NewReader(`{"refresh_token":"refresh-secret"}`))
	req.Host = "127.0.0.1:18080"
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	monetization.NewSessionModule(testCredentialClient(t), lpbsAccountIdentity, lpbsAccountField).Provision(res, req)
	if res.Code != http.StatusCreated || strings.Contains(res.Body.String(), "refresh-secret") {
		t.Fatalf("status/body unsafe: status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestSubscriptionSessionDoesNotTrustSpoofedHostOrForwardedHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/subscription/session", strings.NewReader(`{"refresh_token":"refresh-secret"}`))
	req.Host = "127.0.0.1:18080"
	req.RemoteAddr = "198.51.100.10:54321"
	req.Header.Set("Origin", "https://attacker.example.test")
	req.Header.Set("X-Forwarded-Host", "attacker.example.test")
	res := httptest.NewRecorder()
	monetization.NewSessionModule(testCredentialClient(t), lpbsAccountIdentity, lpbsAccountField).Provision(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s; spoofed host headers must not bypass origin checks", res.Code, res.Body.String())
	}
}
