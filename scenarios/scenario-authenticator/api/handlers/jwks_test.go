package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"scenario-authenticator/auth"
)

// TestJWKSHandlerPublishesVerifiableKey asserts the JWKS endpoint returns the
// loaded RSA public key in a form a consumer can reconstruct and use to verify
// signatures — the contract device-sync-hub's local verifier relies on.
func TestJWKSHandlerPublishesVerifiableKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	auth.SetTestKeys(priv, &priv.PublicKey)

	rr := httptest.NewRecorder()
	JWKSHandler(rr, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}

	var set struct {
		Keys []struct {
			Kty, Use, Alg, Kid, N, E string
		} `json:"keys"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &set); err != nil {
		t.Fatalf("decode jwks: %v; body=%s", err, rr.Body.String())
	}
	if len(set.Keys) != 1 {
		t.Fatalf("keys = %d, want 1", len(set.Keys))
	}
	k := set.Keys[0]
	if k.Kty != "RSA" || k.Use != "sig" || k.Alg != "RS256" {
		t.Fatalf("unexpected header fields: %+v", k)
	}
	if k.Kid == "" {
		t.Fatalf("kid is empty")
	}

	// Reconstruct the public key from n/e and assert it matches the signing key.
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		t.Fatalf("decode n: %v", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		t.Fatalf("decode e: %v", err)
	}
	got := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}
	if got.N.Cmp(priv.PublicKey.N) != 0 || got.E != priv.PublicKey.E {
		t.Fatalf("reconstructed key does not match signing key")
	}
}

// TestJWKSHandlerNoKey returns 503 when no signing key is loaded, rather than
// publishing a bogus key.
func TestJWKSHandlerNoKey(t *testing.T) {
	auth.SetTestKeys(nil, nil)

	rr := httptest.NewRecorder()
	JWKSHandler(rr, httptest.NewRequest(http.MethodGet, "/api/v1/auth/jwks", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
}
