package authn

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func TestJWTVerifierValidatesCanonicalPrincipalAndCachesJWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	token := signJWT(t, key, map[string]any{
		"user_id": "owner-1", "email": "owner@example.test", "roles": []string{"user"},
		"scope": []string{"demo:read", "demo:write"}, "iss": "scenario-authenticator",
		"aud": []string{"scenario:demo"}, "exp": now.Add(time.Hour).Unix(),
	})
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"test-key","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	}))
	defer server.Close()

	verifier := NewJWTVerifier(JWTConfig{Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Audience: "scenario:demo", JWKSURL: server.URL + "/jwks", Now: func() time.Time { return now }})
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !principal.IsHuman() || principal.Subject != "owner-1" || principal.Email != "owner@example.test" || len(principal.Scopes) != 2 {
		t.Fatalf("principal = %#v", principal)
	}
	if _, err := verifier.Verify(context.Background(), token); err != nil {
		t.Fatalf("cached Verify() error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("JWKS requests = %d, want 1", requests)
	}
}

func TestJWTVerifierFailsClosedForIssuerAudienceAlgorithmAndExpiry(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"test-key","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	}))
	defer server.Close()
	verifier := NewJWTVerifier(JWTConfig{Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Audience: "scenario:demo", JWKSURL: server.URL, Now: func() time.Time { return now }})
	cases := []struct {
		name   string
		claims map[string]any
		alg    string
		class  identity.FailureClass
	}{
		{name: "wrong issuer", claims: map[string]any{"iss": "other", "aud": "scenario:demo", "exp": now.Add(time.Hour).Unix()}, class: identity.FailureInvalid},
		{name: "wrong audience", claims: map[string]any{"iss": "scenario-authenticator", "aud": "other", "exp": now.Add(time.Hour).Unix()}, class: identity.FailureInvalid},
		{name: "expired", claims: map[string]any{"iss": "scenario-authenticator", "aud": "scenario:demo", "exp": now.Add(-time.Second).Unix()}, class: identity.FailureExpired},
		{name: "algorithm confusion", claims: map[string]any{"iss": "scenario-authenticator", "aud": "scenario:demo", "exp": now.Add(time.Hour).Unix()}, alg: "HS256", class: identity.FailureInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := signJWTWithAlgorithm(t, key, tc.alg, tc.claims)
			_, err := verifier.Verify(context.Background(), token)
			failure, ok := identity.FailureFromError(err)
			if !ok || failure.Class != tc.class {
				t.Fatalf("error = %v, failure = %#v, ok = %t", err, failure, ok)
			}
		})
	}
}

func TestJWTVerifierAcceptsBoundedMigrationAudience(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	server := httptest.NewServer(jwksHandler(key))
	defer server.Close()
	verifier := NewJWTVerifier(JWTConfig{
		Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator",
		Audience: "scenario-authenticator:bridge", AcceptedAudiences: []string{"scenario-authenticator:default"},
		JWKSURL: server.URL, Now: func() time.Time { return now },
	})
	token := signJWT(t, key, map[string]any{"sub": "user-1", "iss": "scenario-authenticator", "aud": "scenario-authenticator:default", "exp": now.Add(time.Hour).Unix()})
	if principal, err := verifier.Verify(context.Background(), token); err != nil || principal.Subject != "user-1" {
		t.Fatalf("migration audience verification = %#v, %v", principal, err)
	}
}

func TestJWTVerifierReadsConfiguredSameOriginCookie(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	server := httptest.NewServer(jwksHandler(key))
	defer server.Close()
	verifier := NewJWTVerifier(JWTConfig{Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Audience: "scenario:demo", JWKSURL: server.URL, CookieName: "session", Now: func() time.Time { return now }})
	token := signJWT(t, key, map[string]any{"sub": "user-1", "iss": "scenario-authenticator", "aud": "scenario:demo", "exp": now.Add(time.Hour).Unix()})
	principal, err := verifier.VerifyRequest(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err == nil || principal.IsVerified() {
		t.Fatal("missing cookie unexpectedly authenticated")
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	principal, err = verifier.VerifyRequest(context.Background(), req)
	if err != nil || !principal.IsHuman() {
		t.Fatalf("cookie VerifyRequest() = %#v, %v", principal, err)
	}
}

func jwksHandler(key *rsa.PrivateKey) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"test-key","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()), base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	})
}

func signJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	return signJWTWithAlgorithm(t, key, "RS256", claims)
}

func signJWTWithAlgorithm(t *testing.T, key *rsa.PrivateKey, alg string, claims map[string]any) string {
	t.Helper()
	if alg == "" {
		alg = "RS256"
	}
	header, err := json.Marshal(map[string]any{"alg": alg, "kid": "test-key", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	input := encodedHeader + "." + encodedPayload
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
