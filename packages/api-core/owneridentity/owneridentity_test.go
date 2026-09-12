package owneridentity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type testResolver struct{ base string }

func (r testResolver) ResolveScenarioURLDefault(_ context.Context, _ string) (string, error) {
	return r.base, nil
}

func TestClientValidatesOwnerJWTAndCachesJWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	token := signTestToken(t, key, map[string]any{
		"user_id": "owner-1",
		"email":   "owner@example.test",
		"scope":   []string{"vrooli-bridge:scenario-screenshot", "agent-manager:write"},
		"iss":     ExpectedIssuer,
		"aud":     ExpectedAudience,
		"exp":     now.Add(time.Hour).Unix(),
	})

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"test-key","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()))
	}))
	defer server.Close()

	client := NewClient(Config{Resolver: testResolver{base: server.URL}, Now: func() time.Time { return now }})
	identity, err := client.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if identity.Subject != "owner-1" || identity.Email != "owner@example.test" {
		t.Fatalf("identity = %#v", identity)
	}
	if len(identity.Scopes) != 2 || identity.Scopes[0] != "vrooli-bridge:scenario-screenshot" {
		t.Fatalf("scopes = %#v", identity.Scopes)
	}
	if _, err := client.Validate(context.Background(), token); err != nil {
		t.Fatalf("cached Validate: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("JWKS requests = %d, want 1", requests.Load())
	}
}

func TestClientRejectsInvalidClaimsAndProviderOutage(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"test-key","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()))
	}))
	defer server.Close()
	client := NewClient(Config{Resolver: testResolver{base: server.URL}, Now: func() time.Time { return now }})

	badAudience := signTestToken(t, key, map[string]any{
		"user_id": "owner-1", "iss": ExpectedIssuer, "aud": "other", "exp": now.Add(time.Hour).Unix(),
	})
	if _, err := client.Validate(context.Background(), badAudience); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("bad audience error = %v", err)
	}
	badIssuer := signTestToken(t, key, map[string]any{
		"user_id": "owner-1", "iss": "other", "aud": ExpectedAudience, "exp": now.Add(time.Hour).Unix(),
	})
	if _, err := client.Validate(context.Background(), badIssuer); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("bad issuer error = %v", err)
	}

	server.Close()
	noCache := NewClient(Config{Resolver: testResolver{base: "http://127.0.0.1:1"}, Now: func() time.Time { return now }})
	if _, err := noCache.Validate(context.Background(), badAudience); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("provider outage error = %v", err)
	}
}

func signTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString(mustJSON(t, map[string]string{"alg": "RS256", "kid": "test-key", "typ": "JWT"}))
	payload := base64.RawURLEncoding.EncodeToString(mustJSON(t, claims))
	input := header + "." + payload
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
