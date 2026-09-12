package policygate

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestJWTVerifierValidatesAuthenticatorContract(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "alg": "RS256", "kid": "test-key", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})}}})
	}))
	defer server.Close()
	now := time.Date(2026, 9, 6, 20, 0, 0, 0, time.UTC)
	verifier := NewJWTVerifier(AuthenticatorConfig{Issuer: "scenario-authenticator", Realm: "default", Audience: "scenario-authenticator:default", JWKSURL: server.URL, Now: func() time.Time { return now }})
	token := signTestJWT(t, key, map[string]any{"alg": "RS256", "kid": "test-key"}, map[string]any{"user_id": "user-1", "sub": "user-1", "email": "operator@example.test", "roles": []string{"user"}, "scope": []string{"repo:write"}, "iss": "scenario-authenticator", "aud": "scenario-authenticator:default", "exp": now.Add(time.Minute).Unix()})
	principal, err := verifier.Verify(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if !principal.Verified || principal.Kind != cliutil.CallerKindHuman || principal.Subject != "user-1" || principal.Realm != "default" {
		t.Fatalf("principal=%#v", principal)
	}
	revokedVerifier := NewJWTVerifier(AuthenticatorConfig{
		Issuer: "scenario-authenticator", Realm: "default", Audience: "scenario-authenticator:default", JWKSURL: server.URL,
		Now: func() time.Time { return now },
		Validate: func(_ context.Context, _ string) (Principal, error) {
			return Principal{}, nil
		},
	})
	if _, err := revokedVerifier.Verify(t.Context(), token); err == nil || !strings.Contains(err.Error(), "revoked authenticator credential") {
		t.Fatalf("revoked credential err=%v", err)
	}
}

func TestJWTVerifierRejectsIssuerAudienceExpiryAndAlgorithmFailures(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "alg": "RS256", "kid": "test-key", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})}}})
	}))
	defer server.Close()
	now := time.Date(2026, 9, 6, 20, 0, 0, 0, time.UTC)
	verifier := NewJWTVerifier(AuthenticatorConfig{Issuer: "scenario-authenticator", Audience: "scenario-authenticator:default", JWKSURL: server.URL, Now: func() time.Time { return now }})
	for _, tc := range []struct {
		name     string
		alg      string
		issuer   string
		audience string
		exp      int64
	}{
		{"wrong issuer", "RS256", "other", "scenario-authenticator:default", now.Add(time.Minute).Unix()},
		{"wrong audience", "RS256", "scenario-authenticator", "other", now.Add(time.Minute).Unix()},
		{"expired", "RS256", "scenario-authenticator", "scenario-authenticator:default", now.Add(-time.Minute).Unix()},
		{"wrong algorithm", "HS256", "scenario-authenticator", "scenario-authenticator:default", now.Add(time.Minute).Unix()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := signTestJWT(t, key, map[string]any{"alg": tc.alg, "kid": "test-key"}, map[string]any{"user_id": "user-1", "iss": tc.issuer, "aud": tc.audience, "exp": tc.exp})
			if _, err := verifier.Verify(t.Context(), token); err == nil || !strings.Contains(err.Error(), "invalid authenticator credential") {
				t.Fatalf("err=%v, want invalid credential", err)
			}
		})
	}
}

func TestJWTVerifierRefreshesJWKSOnExpiryAndKeyRotation(t *testing.T) {
	firstKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	secondKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	currentKey := firstKey
	currentKid := "first"
	fetches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches++
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "alg": "RS256", "kid": currentKid,
			"n": base64.RawURLEncoding.EncodeToString(currentKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}}})
	}))
	defer server.Close()

	now := time.Date(2026, 9, 6, 20, 0, 0, 0, time.UTC)
	verifier := NewJWTVerifier(AuthenticatorConfig{
		Issuer: "scenario-authenticator", Realm: "default", Audience: "scenario-authenticator:default",
		JWKSURL: server.URL, JWKSCacheTTL: time.Minute, Now: func() time.Time { return now },
	})
	claims := func(exp time.Time) map[string]any {
		return map[string]any{"user_id": "user-1", "iss": "scenario-authenticator", "aud": "scenario-authenticator:default", "exp": exp.Unix()}
	}
	if _, err := verifier.Verify(t.Context(), signTestJWT(t, firstKey, map[string]any{"alg": "RS256", "kid": "first"}, claims(now.Add(time.Hour)))); err != nil {
		t.Fatal(err)
	}
	if fetches != 1 {
		t.Fatalf("initial JWKS fetches=%d, want 1", fetches)
	}

	now = now.Add(30 * time.Second)
	if _, err := verifier.Verify(t.Context(), signTestJWT(t, firstKey, map[string]any{"alg": "RS256", "kid": "first"}, claims(now.Add(time.Hour)))); err != nil {
		t.Fatal(err)
	}
	if fetches != 1 {
		t.Fatalf("cached JWKS fetches=%d, want 1", fetches)
	}

	currentKey, currentKid = secondKey, "second"
	now = now.Add(10 * time.Second)
	if _, err := verifier.Verify(t.Context(), signTestJWT(t, secondKey, map[string]any{"alg": "RS256", "kid": "second"}, claims(now.Add(time.Hour)))); err != nil {
		t.Fatal(err)
	}
	if fetches != 2 {
		t.Fatalf("rotation refreshes=%d, want 2", fetches)
	}

	now = now.Add(2 * time.Minute)
	if _, err := verifier.Verify(t.Context(), signTestJWT(t, secondKey, map[string]any{"alg": "RS256", "kid": "second"}, claims(now.Add(time.Hour)))); err != nil {
		t.Fatal(err)
	}
	if fetches != 3 {
		t.Fatalf("expired cache fetches=%d, want 3", fetches)
	}
}

func signTestJWT(t *testing.T, key *rsa.PrivateKey, header, claims map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	input := encode(header) + "." + encode(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
