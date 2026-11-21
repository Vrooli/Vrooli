package administration

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeAuthorizationCodeRejectsOversizedBody(t *testing.T) {
	deps := testUserAuthDependencies()
	recorder := httptest.NewRecorder()
	body := bytes.Repeat([]byte("x"), 9<<10)
	ExchangeAuthorizationCode(deps, nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(body)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", recorder.Code)
	}
}

func TestPKCEMatchesS256Challenge(t *testing.T) {
	verifier := "native-verifier-with-enough-entropy"
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	if !pkceMatches(verifier, challenge) || pkceMatches("wrong", challenge) {
		t.Fatal("PKCE comparison is incorrect")
	}
}

func TestValidLoopbackRedirectSupportsIPv6Literal(t *testing.T) {
	if !validLoopbackRedirect("http://[::1]:43111/callback") {
		t.Fatal("IPv6 loopback redirect was rejected")
	}
	for _, redirect := range []string{"vrooli://auth/callback", "http://192.168.1.4:1234/callback", "https://127.0.0.1:1234/callback"} {
		if validLoopbackRedirect(redirect) {
			t.Fatalf("redirect %q unexpectedly accepted", redirect)
		}
	}
}
