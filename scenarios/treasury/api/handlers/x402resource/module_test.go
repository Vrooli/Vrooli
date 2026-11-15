package x402resource

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	x402rail "treasury/internal/rail/x402"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

func invoke(handler http.HandlerFunc, path string, principal *identity.Principal) *httptest.ResponseRecorder {
	body := `{"job_id":"job-1","resource_id":"resource-1","order_id":"order-1","payment_signature":"signed"}`
	if path == pathSettle || path == pathCancel {
		body = `{"job_id":"job-1","resource_id":"resource-1","order_id":"order-1"}`
	}
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if principal != nil {
		request = request.WithContext(identity.WithPrincipal(request.Context(), *principal))
	}
	response := httptest.NewRecorder()
	handler(response, request)
	return response
}

func TestModuleUsesConfiguredScenarioAuthenticatorJWKS(t *testing.T) {
	jwksURL, token := x402ResourceServiceJWT(t, capAuthorize)
	t.Setenv("VROOLI_AUTH_SCENARIO_ISSUER", "scenario-authenticator")
	t.Setenv("VROOLI_AUTH_SCENARIO_JWKS_URL", jwksURL)
	router := mux.NewRouter()
	Module(nil).Mount(router)
	request := httptest.NewRequest(http.MethodPost, pathVerify, strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("configured run-local service JWT should pass auth and reach the unavailable test gate: status=%d body=%s", response.Code, response.Body.String())
	}
}

func x402ResourceServiceJWT(t *testing.T, scopes ...string) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := func(value any) string {
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "configured-jwks"}) + "." + encode(map[string]any{
		"user_id": "test-service", "sub": "test-service", "actor_kind": "service", "scope": scopes,
		"aud": audience, "iss": "scenario-authenticator", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	token := unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	public, err := json.Marshal(map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "configured-jwks", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(public)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/jwks", token
}

func TestAuthorizedResponseUsesClientContractAndDoesNotExposeRecoveryReference(t *testing.T) {
	value := response(x402rail.Admission{ID: "admission-1", Status: "verified", Mode: "local-fork", JobID: "job-1", ResourceID: "resource-1", OrderID: "order-1", RecoveryReference: "secret-reference"})
	if value.AdmissionID != "admission-1" || value.Status != "authorized" || value.Mode != "local-fork" || value.JobID != "job-1" {
		t.Fatalf("unexpected authorized response contract: %+v", value)
	}
	if value.Detail == "secret-reference" {
		t.Fatal("recovery reference leaked into response")
	}
}

func TestResourceEndpointsRequireVerifiedScopedService(t *testing.T) {
	h := &handler{}
	if got := invoke(h.verify, pathVerify, nil).Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous verification status=%d, want 401", got)
	}
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "human:test", Verified: true, Scopes: []string{capAuthorize}}
	if got := invoke(h.verify, pathVerify, &human).Code; got != http.StatusUnauthorized {
		t.Fatalf("human verification status=%d, want 401", got)
	}
	wrongScope := identity.Principal{Kind: identity.ActorService, Subject: "service:test", Verified: true, Scopes: []string{"treasury:other"}}
	if got := invoke(h.verify, pathVerify, &wrongScope).Code; got != http.StatusForbidden {
		t.Fatalf("wrong-scope verification status=%d, want 403", got)
	}
	service := identity.Principal{Kind: identity.ActorService, Subject: "service:test", Verified: true, Scopes: []string{capAuthorize, capSettle, capCancel}}
	if got := invoke(h.verify, pathVerify, &service).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("authorized verification without configured gate status=%d, want 503", got)
	}
	if got := invoke(h.settle, pathSettle, &wrongScope).Code; got != http.StatusForbidden {
		t.Fatalf("wrong-scope settlement status=%d, want 403", got)
	}
	if got := invoke(h.settle, pathSettle, &service).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("authorized settlement without configured gate status=%d, want 503", got)
	}
	if got := invoke(h.cancel, pathCancel, &wrongScope).Code; got != http.StatusForbidden {
		t.Fatalf("wrong-scope cancellation status=%d, want 403", got)
	}
	if got := invoke(h.cancel, pathCancel, &service).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("authorized cancellation without configured gate status=%d, want 503", got)
	}
}
