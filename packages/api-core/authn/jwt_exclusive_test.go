package authn

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type exclusiveJWTDoer struct {
	handler http.Handler
	calls   int
}

func (d *exclusiveJWTDoer) Do(r *http.Request) (*http.Response, error) {
	d.calls++
	w := httptest.NewRecorder()
	d.handler.ServeHTTP(w, r)
	return w.Result(), nil
}
func TestJWTVerifierExclusiveAudience(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0).UTC()
	doer := &exclusiveJWTDoer{handler: jwksHandler(key)}
	v := NewJWTVerifier(JWTConfig{Issuer: "scenario-authenticator", Audience: "vrooli-local-work:events", ExclusiveAudience: true, JWKSURL: "https://fixture.invalid/jwks", Doer: doer, Now: func() time.Time { return now }})
	if !v.UsesExclusiveAudience("vrooli-local-work:events") || v.UsesExclusiveAudience("other") {
		t.Fatal("exclusive construction contract")
	}
	for _, tt := range []struct {
		name    string
		aud     any
		allowed bool
	}{
		{"scalar", "vrooli-local-work:events", true}, {"singleton", []string{"vrooli-local-work:events"}, true},
		{"alternate", "scenario-authenticator:default", false}, {"multiple", []string{"vrooli-local-work:events", "scenario-authenticator:default"}, false},
		{"duplicate audience", []string{"vrooli-local-work:events", "vrooli-local-work:events"}, false}, {"missing", nil, false}, {"empty", []string{}, false}, {"mixed", []any{"vrooli-local-work:events", 1}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			token := signJWT(t, key, map[string]any{"sub": "service", "iss": "scenario-authenticator", "aud": tt.aud, "exp": now.Add(time.Hour).Unix()})
			_, err := v.Verify(context.Background(), token)
			if (err == nil) != tt.allowed {
				t.Fatalf("allowed=%v err=%v", tt.allowed, err)
			}
		})
	}
	if doer.calls != 1 {
		t.Fatal("unexpected fixture JWKS reads")
	}
}
func TestJWTVerifierExclusiveRejectsMigrationConfiguration(t *testing.T) {
	doer := &exclusiveJWTDoer{handler: http.NotFoundHandler()}
	v := NewJWTVerifier(JWTConfig{Issuer: "scenario-authenticator", Audience: "vrooli-local-work:events", ExclusiveAudience: true, AcceptedAudiences: []string{"scenario-authenticator:default"}, Doer: doer, JWKSURL: "https://fixture.invalid/jwks"})
	if v.UsesExclusiveAudience("vrooli-local-work:events") {
		t.Fatal("mixed construction admitted")
	}
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	token := signJWT(t, key, map[string]any{"sub": "service", "iss": "scenario-authenticator", "aud": "vrooli-local-work:events", "exp": time.Now().Add(time.Hour).Unix()})
	if _, err := v.Verify(context.Background(), token); err == nil || doer.calls != 0 {
		t.Fatal("invalid config contacted verifier or admitted")
	}
}
func TestJWTVerifierAudienceConfigurationIsImmutable(t *testing.T) {
	aud := []string{"old"}
	v := NewJWTVerifier(JWTConfig{Audience: "primary", AcceptedAudiences: aud})
	aud[0] = "broad"
	if v.cfg.AcceptedAudiences[0] != "old" {
		t.Fatal("configuration alias")
	}
}
func TestJWTVerifierExclusiveRejectsAmbiguousSignedClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0).UTC()
	doer := &exclusiveJWTDoer{handler: jwksHandler(key)}
	v := NewJWTVerifier(JWTConfig{Issuer: "scenario-authenticator", Audience: "vrooli-local-work:events", ExclusiveAudience: true, JWKSURL: "https://fixture.invalid/jwks", Doer: doer, Now: func() time.Time { return now }})
	for _, payload := range []string{
		`{"sub":"service","iss":"scenario-authenticator","aud":"other","aud":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"service","iss":"scenario-authenticator","AUD":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"service","iss":"scenario-authenticator","aud":"other","AUD":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"other","sub":"service","iss":"scenario-authenticator","aud":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"service","scope":["admin"],"ſcope":["local-work:events"],"iss":"scenario-authenticator","aud":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"service","scope":["admin"],"\u017fcope":["local-work:events"],"iss":"scenario-authenticator","aud":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"service","iss":"other","iſſ":"scenario-authenticator","aud":"vrooli-local-work:events","exp":1700003600}`,
		`{"sub":"service","iss":"other","i\u017f\u017f":"scenario-authenticator","aud":"vrooli-local-work:events","exp":1700003600}`,
	} {
		input := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test-key"}`)) + "." + base64.RawURLEncoding.EncodeToString([]byte(payload))
		digest := sha256.Sum256([]byte(input))
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := v.Verify(context.Background(), input+"."+base64.RawURLEncoding.EncodeToString(signature)); err == nil {
			t.Fatal("ambiguous signed claims accepted")
		}
	}
}
