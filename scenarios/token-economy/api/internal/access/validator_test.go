package access

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWKSValidatorAcceptsAuthenticatorClaims(t *testing.T) {
	key := newTestKey(t)
	server := newJWKSServer(t, &key.PublicKey)
	defer server.Close()
	now := time.Unix(2_000_000_000, 0)
	validator := NewJWKSValidator(staticResolver(server.URL), server.Client())
	validator.now = func() time.Time { return now }
	token := signTestToken(t, key, map[string]any{
		"user_id": "parent-1",
		"roles":   []string{"minter"},
		"scope":   []string{ScopeMinter},
		"iss":     authenticatorScenario,
		"aud":     []string{authenticatorAudience},
		"exp":     now.Add(time.Hour).Unix(),
		"nbf":     now.Add(-time.Minute).Unix(),
	})

	identity, err := validator.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if identity.Subject != "parent-1" || !identity.HasScope(ScopeMinter) || len(identity.Roles) != 1 || identity.Roles[0] != "minter" {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestJWKSValidatorRejectsUntrustedOrExpiredClaims(t *testing.T) {
	key := newTestKey(t)
	server := newJWKSServer(t, &key.PublicKey)
	defer server.Close()
	now := time.Unix(2_000_000_000, 0)
	validator := NewJWKSValidator(staticResolver(server.URL), server.Client())
	validator.now = func() time.Time { return now }

	tests := map[string]map[string]any{
		"wrong issuer":   {"user_id": "parent-1", "iss": "caller", "aud": []string{authenticatorAudience}, "exp": now.Add(time.Hour).Unix()},
		"wrong audience": {"user_id": "parent-1", "iss": authenticatorScenario, "aud": []string{"other"}, "exp": now.Add(time.Hour).Unix()},
		"expired":        {"user_id": "parent-1", "iss": authenticatorScenario, "aud": []string{authenticatorAudience}, "exp": now.Add(-time.Second).Unix()},
		"missing expiry": {"user_id": "parent-1", "iss": authenticatorScenario, "aud": []string{authenticatorAudience}},
		"future nbf":     {"user_id": "parent-1", "iss": authenticatorScenario, "aud": []string{authenticatorAudience}, "exp": now.Add(time.Hour).Unix(), "nbf": now.Add(time.Minute).Unix()},
	}
	for name, claims := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := validator.Validate(context.Background(), signTestToken(t, key, claims))
			if !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("error = %v, want ErrUnauthenticated", err)
			}
		})
	}
}

func TestJWKSValidatorFailsClosedWhenAuthenticatorUnavailable(t *testing.T) {
	validator := NewJWKSValidator(failingResolver{}, http.DefaultClient)
	_, err := validator.Validate(context.Background(), "header.payload.signature")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("malformed token error = %v, want ErrUnauthenticated", err)
	}

	key := newTestKey(t)
	now := time.Now()
	token := signTestToken(t, key, map[string]any{
		"user_id": "parent-1", "iss": authenticatorScenario,
		"aud": []string{authenticatorAudience}, "exp": now.Add(time.Hour).Unix(),
	})
	_, err = validator.Validate(context.Background(), token)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("resolver error = %v, want ErrUnavailable", err)
	}
}

type staticResolver string

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

type failingResolver struct{}

func (failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("offline")
}

func newJWKSServer(t *testing.T, key *rsa.PublicKey) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "alg": "RS256", "n": n, "e": e}}})
	}))
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func signTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
