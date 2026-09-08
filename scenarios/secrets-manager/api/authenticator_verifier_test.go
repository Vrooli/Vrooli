package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticatorVerifierRequiresIssuerAudienceAndRS256(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const kid = "test-kid"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": kid, "kty": "RSA", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}}})
	}))
	defer server.Close()
	verifier := &authenticatorVerifier{
		jwksURL: server.URL, issuer: "scenario-authenticator", aud: "scenario-authenticator:default",
		client: server.Client(), keys: make(map[string]*rsa.PublicKey),
	}

	mint := func(issuer, audience string, method jwt.SigningMethod) string {
		claims := jwt.MapClaims{"iss": issuer, "aud": audience, "sub": "account-1", "exp": time.Now().Add(time.Minute).Unix()}
		token := jwt.NewWithClaims(method, claims)
		token.Header["kid"] = kid
		if method == jwt.SigningMethodRS256 {
			raw, _ := token.SignedString(privateKey)
			return raw
		}
		raw, _ := token.SignedString([]byte("wrong-secret"))
		return raw
	}

	identity, err := verifier.Verify(t.Context(), mint("scenario-authenticator", "scenario-authenticator:default", jwt.SigningMethodRS256))
	if err != nil || identity.Subject != "account-1" {
		t.Fatalf("valid token = %+v, err=%v", identity, err)
	}
	if _, err := verifier.Verify(t.Context(), mint("scenario-authenticator", "other-audience", jwt.SigningMethodRS256)); err == nil {
		t.Fatal("cross-audience token accepted")
	}
	if _, err := verifier.Verify(t.Context(), mint("scenario-authenticator", "scenario-authenticator:default", jwt.SigningMethodHS256)); err == nil {
		t.Fatal("HS256 token accepted")
	}
}
