package credentialclient

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPKCEAuthorizationURLUsesS256AndLoopback(t *testing.T) {
	challenge, err := NewPKCEChallenge()
	if err != nil {
		t.Fatal(err)
	}
	redirect := "http://127.0.0.1:43127/callback"
	raw, err := challenge.AuthorizationURL("https://billing.example", "browser-automation-studio", redirect)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("code_challenge_method") != "S256" || query.Get("state") != challenge.State || query.Get("redirect_uri") != redirect {
		t.Fatalf("authorization query = %v", query)
	}
	digest := sha256.Sum256([]byte(challenge.Verifier))
	if query.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(digest[:]) {
		t.Fatal("code challenge does not match verifier")
	}
}

func TestPKCERejectsCustomAndRemoteRedirects(t *testing.T) {
	for _, redirect := range []string{"vrooli://callback", "http://example.com:43127/callback", "http://127.0.0.1/callback"} {
		if err := ValidateLoopbackRedirect(redirect); !errors.Is(err, ErrInvalidLoopbackRedirect) {
			t.Fatalf("redirect %q error = %v", redirect, err)
		}
	}
}

func TestExchangeAuthorizationCodeSendsVerifierAndReturnsTokens(t *testing.T) {
	challenge, err := NewPKCEChallenge()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["code_verifier"] != challenge.Verifier || body["grant_type"] != "authorization_code" {
			t.Fatalf("exchange body = %v", body)
		}
		if !strings.HasSuffix(r.URL.Path, "/api/v1/auth/token") {
			t.Fatalf("exchange path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_at":"2099-01-01T00:00:00Z"}`))
	}))
	defer server.Close()
	access, err := ExchangeAuthorizationCode(context.Background(), server.Client(), server.URL, "client", "http://127.0.0.1:43127/callback", "code", challenge)
	if err != nil || access.AccessToken != "access" {
		t.Fatalf("access = %#v err = %v", access, err)
	}
}
