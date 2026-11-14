package apiauth_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

// Exercise real issuer/audience/signature verification and the normal generated
// mutation client against private SQLite storage, without touching host lights.
func TestScopedIssuerTokenMutatesEnrollmentThroughPublicHTTP(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "private-issuer", "alg": "RS256", "use": "sig", "n": encode(key.N.Bytes()), "e": encode(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer issuer.Close()
	provider := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "scenario-authenticator:lighting-control", JWKSURL: issuer.URL})
	f := newFixture(t, identity.Principal{}, false, false, provider)
	sign := func(audience string, scopes []string) string {
		header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "private-issuer"})
		claims, _ := json.Marshal(map[string]any{"iss": "scenario-authenticator", "aud": audience, "sub": "private-operator", "exp": time.Now().Add(time.Minute).Unix(), "scope": scopes})
		message := encode(header) + "." + encode(claims)
		digest := sha256.Sum256([]byte(message))
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatal(err)
		}
		return message + "." + encode(signature)
	}
	before := databaseSnapshot(t, f.db)
	for _, token := range []string{sign("scenario-authenticator:other", []string{"lighting-control:write"}), sign("scenario-authenticator:lighting-control", []string{"lighting-control:read"})} {
		req := connect.NewRequest(&controlv1.EnrollControllerRequest{ControllerId: f.controllerID, Enrolled: false})
		req.Header().Set("Authorization", "Bearer "+token)
		if _, err := f.control.EnrollController(context.Background(), req); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("invalid authority admitted: %v", err)
		}
		if databaseSnapshot(t, f.db) != before || f.backend.calls.Load() != 0 {
			t.Fatal("denial changed storage/backend")
		}
	}
	req := connect.NewRequest(&controlv1.EnrollControllerRequest{ControllerId: f.controllerID, Enrolled: false})
	req.Header().Set("Authorization", "Bearer "+sign("scenario-authenticator:lighting-control", []string{"lighting-control:write"}))
	if _, err := f.control.EnrollController(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if databaseSnapshot(t, f.db) == before {
		t.Fatal("authorized enrollment was not persisted")
	}
	if f.backend.calls.Load() != 0 {
		t.Fatal("metadata-only mutation touched lighting backend")
	}
}
