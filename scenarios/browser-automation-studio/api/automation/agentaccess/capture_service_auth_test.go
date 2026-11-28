package agentaccess

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
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type captureVerifierFixture struct {
	principal   identity.Principal
	err         error
	acceptToken string
}

func (v captureVerifierFixture) Verify(_ context.Context, token string) (identity.Principal, error) {
	if v.acceptToken != "" && token != v.acceptToken {
		return identity.Principal{}, errors.New("unknown token")
	}
	return v.principal, v.err
}

func validCapturePrincipal() identity.Principal {
	return identity.Principal{
		Kind: identity.ActorService, Subject: CaptureServicePrincipal,
		Verified: true, Issuer: "scenario-authenticator", Audience: CaptureServiceAudience,
		Scopes: []string{CaptureServiceScope},
	}
}

func TestCaptureServiceAdmissionRequiresBearerAndExactRole(t *testing.T) {
	for _, tc := range []struct {
		name      string
		header    string
		principal identity.Principal
		err       error
	}{
		{name: "missing bearer", principal: validCapturePrincipal()},
		{name: "bad signature", header: "Bearer bad", err: errors.New("signature rejected")},
		{name: "wrong principal", header: "Bearer token", principal: identity.Principal{Kind: identity.ActorService, Subject: "another-bas-capture", Verified: true, Issuer: "scenario-authenticator", Audience: CaptureServiceAudience, Scopes: []string{CaptureServiceScope}}},
		{name: "wrong audience", header: "Bearer token", principal: func() identity.Principal {
			p := validCapturePrincipal()
			p.Audience = "scenario-authenticator:other"
			return p
		}()},
		{name: "wrong issuer", header: "Bearer token", principal: func() identity.Principal { p := validCapturePrincipal(); p.Issuer = "other-issuer"; return p }()},
		{name: "extra scope", header: "Bearer token", principal: func() identity.Principal {
			p := validCapturePrincipal()
			p.Scopes = append(p.Scopes, "browser-automation-studio:execute")
			return p
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			admission, err := NewCaptureServiceAdmission(captureVerifierFixture{principal: tc.principal, err: tc.err}, 1)
			if err != nil {
				t.Fatal(err)
			}
			headers := http.Header{}
			if tc.header != "" {
				headers.Set("Authorization", tc.header)
			}
			if _, _, err := admission.Admit(context.Background(), headers); err == nil {
				t.Fatal("invalid service identity was admitted")
			}
		})
	}
}

func TestCaptureServiceAdmissionRejectsWhenBoundedLaneIsFull(t *testing.T) {
	admission, err := NewCaptureServiceAdmission(captureVerifierFixture{principal: validCapturePrincipal()}, 1)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{"Authorization": []string{"Bearer token"}}
	_, release, err := admission.Admit(context.Background(), headers)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, _, err := admission.Admit(context.Background(), headers); !errors.Is(err, ErrCaptureLaneFull) {
		t.Fatal("second request entered a full system capture lane")
	}
	if !admission.IsLimitError(ErrCaptureLaneFull) || !admission.IsLimitError(ErrCaptureRateLimited) || admission.IsLimitError(errors.New("invalid token")) {
		t.Fatal("finite lane errors were not classified separately from authorization failures")
	}
}

func TestCaptureServiceAdmissionAppliesFiniteRequestRate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	admission, err := NewCaptureServiceAdmissionWithLimits(captureVerifierFixture{principal: validCapturePrincipal()}, 1, 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	admission.now = func() time.Time { return now }
	headers := http.Header{"Authorization": []string{"Bearer token"}}
	_, release, err := admission.Admit(context.Background(), headers)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if _, _, err := admission.Admit(context.Background(), headers); !errors.Is(err, ErrCaptureRateLimited) {
		t.Fatalf("second request error = %v, want finite rate limit", err)
	}
	now = now.Add(time.Minute)
	_, release, err = admission.Admit(context.Background(), headers)
	if err != nil {
		t.Fatalf("request after rate window: %v", err)
	}
	release()
}

func TestCaptureOnlyPrincipalCannotReadGenericArtifactRoute(t *testing.T) {
	admission, err := NewCaptureServiceAdmission(captureVerifierFixture{principal: validCapturePrincipal(), acceptToken: "service-token"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	handler := admission.RejectServiceToken(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/execution/result.json", nil)
	request.Header.Set("Authorization", "Bearer service-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("artifact route status=%d called=%t; capture-only service token must be refused", response.Code, called)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/execution/result.json", nil)
	request.Header.Set("Authorization", "Bearer ordinary-user-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !called {
		t.Fatalf("ordinary artifact request status=%d called=%t; existing handler must remain reachable", response.Code, called)
	}
}

func TestCaptureServiceVerifierChecksSignatureIssuerAudienceAndScope(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "kid": "capture-fixture", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	now := time.Now().UTC()
	issue := func(issuer, audience string, scopes []string) string {
		encode := func(value any) string {
			raw, _ := json.Marshal(value)
			return base64.RawURLEncoding.EncodeToString(raw)
		}
		unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "capture-fixture"}) + "." + encode(map[string]any{
			"sub": CaptureServicePrincipal, "user_id": CaptureServicePrincipal, "actor_kind": "service",
			"aud": audience, "iss": issuer, "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "scope": scopes,
		})
		digest := sha256.Sum256([]byte(unsigned))
		signature, signErr := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if signErr != nil {
			t.Fatal(signErr)
		}
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	}
	verifier := CaptureServiceVerifierWithConfig(authn.JWTConfig{JWKSURL: server.URL, Now: func() time.Time { return now }})
	admission, err := NewCaptureServiceAdmission(verifier, 1)
	if err != nil {
		t.Fatal(err)
	}
	header := func(token string) http.Header { return http.Header{"Authorization": []string{"Bearer " + token}} }
	valid := issue("scenario-authenticator", CaptureServiceAudience, []string{CaptureServiceScope})
	_, release, err := admission.Admit(context.Background(), header(valid))
	if err != nil {
		t.Fatalf("valid signed capture role rejected: %v", err)
	}
	release()
	for name, token := range map[string]string{
		"wrong issuer":   issue("untrusted-issuer", CaptureServiceAudience, []string{CaptureServiceScope}),
		"wrong audience": issue("scenario-authenticator", "scenario-authenticator:other", []string{CaptureServiceScope}),
		"extra scope":    issue("scenario-authenticator", CaptureServiceAudience, []string{CaptureServiceScope, "browser-automation-studio:execute"}),
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := admission.Admit(context.Background(), header(token)); err == nil {
				t.Fatalf("invalid signed role accepted: %v", err)
			}
		})
	}
	parts := strings.Split(valid, ".")
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	signature[0] ^= 0xff
	tampered := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(signature)
	if _, _, err := admission.Admit(context.Background(), header(tampered)); err == nil {
		t.Fatal("invalid signature was accepted")
	}
}
