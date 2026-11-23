package lpbsidentity

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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/principalrole"
)

func TestGatewayCreditInferenceRequiresExactServiceRole(t *testing.T) {
	role := GatewayCreditInference
	valid := apiidentity.Principal{Kind: apiidentity.ActorService, Subject: "web-console-lpbs-ai-inference", Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true}
	if source, err := role.Source(valid); err != nil || source != "web-console" {
		t.Fatalf("Source(valid) = %q, %v", source, err)
	}
	for name, principal := range map[string]apiidentity.Principal{
		"unverified":        {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{role.Scope}},
		"human":             {Kind: apiidentity.ActorHuman, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true},
		"wrong audience":    {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: "scenario:ai-gateway", Scopes: []string{role.Scope}, Verified: true},
		"wrong scope":       {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{"lpbs:release.manage"}, Verified: true},
		"extra scope":       {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{role.Scope, "lpbs:release.manage"}, Verified: true},
		"wrong name":        {Kind: apiidentity.ActorService, Subject: "web-console", Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true},
		"wrong source name": {Kind: apiidentity.ActorService, Subject: "web-console-lpbs-credit-grant", Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := role.Source(principal); !errors.Is(err, principalrole.ErrRefused) {
				t.Fatalf("Source(%#v) error = %v, want ErrRefused", principal, err)
			}
		})
	}
}

func TestGatewayCreditInferenceCredentialIdentityIsScenarioSpecific(t *testing.T) {
	want := "scenario-authenticator/service-principal/browser-automation-studio-lpbs-ai-inference/landing-page-business-suite"
	if got := GatewayCreditInference.CredentialIdentity("browser-automation-studio"); got != want {
		t.Fatalf("CredentialIdentity() = %q, want %q", got, want)
	}
	if got := GatewayCreditInference.Principal("web-console"); got != "web-console-lpbs-ai-inference" {
		t.Fatalf("Principal() = %q", got)
	}
}

func TestGatewayCreditInferenceAcceptsOnlyFreshSignedPrincipal(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"synthetic","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()), base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	}))
	defer server.Close()
	verifier := authn.NewJWTVerifier(authn.JWTConfig{
		Source: apiidentity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Audience: GatewayCreditInference.Audience,
		ExclusiveAudience: true, Kind: apiidentity.ActorService, JWKSURL: server.URL, Now: func() time.Time { return now },
	})
	token := signedScenarioPrincipal(t, key, map[string]any{
		"user_id": GatewayCreditInference.Principal("web-console"), "actor_kind": "service", "scope": []string{GatewayCreditInference.Scope},
		"iss": "scenario-authenticator", "aud": []string{GatewayCreditInference.Audience}, "exp": now.Add(time.Minute).Unix(),
	})
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("synthetic issuer token verification failed: %v", err)
	}
	if source, err := GatewayCreditInference.Source(principal); err != nil || source != "web-console" {
		t.Fatalf("verified source = %q, %v", source, err)
	}
	for name, claims := range map[string]map[string]any{
		"wrong audience": {"user_id": GatewayCreditInference.Principal("web-console"), "actor_kind": "service", "scope": []string{GatewayCreditInference.Scope}, "iss": "scenario-authenticator", "aud": []string{"scenario:ai-gateway"}, "exp": now.Add(time.Minute).Unix()},
		"expired":        {"user_id": GatewayCreditInference.Principal("web-console"), "actor_kind": "service", "scope": []string{GatewayCreditInference.Scope}, "iss": "scenario-authenticator", "aud": []string{GatewayCreditInference.Audience}, "exp": now.Add(-time.Second).Unix()},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Verify(context.Background(), signedScenarioPrincipal(t, key, claims)); err == nil {
				t.Fatal("invalid synthetic principal token was accepted")
			}
		})
	}
	forged := strings.TrimRight(token, "=")
	if strings.HasSuffix(forged, "a") {
		forged = forged[:len(forged)-1] + "b"
	} else {
		forged += "a"
	}
	if _, err := verifier.Verify(context.Background(), forged); err == nil {
		t.Fatal("forged signature was accepted")
	}
}

func signedScenarioPrincipal(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "synthetic", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}
