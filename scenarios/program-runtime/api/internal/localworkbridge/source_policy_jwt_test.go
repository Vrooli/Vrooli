package localworkbridge

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
)

type sourceJWKSFixture struct {
	key   *rsa.PrivateKey
	calls int
}

func (d *sourceJWKSFixture) Do(r *http.Request) (*http.Response, error) {
	d.calls++
	if r.URL.String() != "https://fixture.invalid/jwks" || r.Header.Get("Authorization") != "" {
		return nil, ErrSourceAuthority
	}
	body := fmt.Sprintf(`{"keys":[{"kid":"fixture","kty":"RSA","alg":"RS256","n":%q,"e":"AQAB"}]}`, base64.RawURLEncoding.EncodeToString(d.key.N.Bytes()))
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
}
func TestSourceGateCanonicalSignedJWT(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	doer := &sourceJWKSFixture{key: key}
	g, _, b := sourceFixture(t, SourceEvents)
	now := g.clock()
	verifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:events", ExclusiveAudience: true, JWKSURL: "https://fixture.invalid/jwks", Doer: doer, Now: func() time.Time { return now }})
	g, err = NewSourceGate(SourceGateConfig{Enabled: true, Policy: g.policy, Bindings: b, Verifier: verifier})
	if err != nil {
		t.Fatal(err)
	}
	g.clock = func() time.Time { return now }
	for _, tt := range []struct {
		name    string
		aud     any
		allowed bool
	}{
		{"exact", "vrooli-local-work:events", true}, {"single array", []string{"vrooli-local-work:events"}, true},
		{"alternate", "scenario-authenticator:default", false}, {"multi audience", []string{"vrooli-local-work:events", "scenario-authenticator:program-runtime"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(map[string]any{"sub": "bridge-service", "iss": "scenario-authenticator", "actor_kind": "service", "scope": []string{"local-work:events"}, "aud": tt.aud, "exp": now.Add(time.Hour).Unix()})
			input := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"fixture"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload)
			digest := sha256.Sum256([]byte(input))
			signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
			if err != nil {
				t.Fatal(err)
			}
			r := sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", "")
			r.Header.Set("Authorization", "Bearer "+input+"."+base64.RawURLEncoding.EncodeToString(signature))
			_, err = g.Admit(context.Background(), r)
			if (err == nil) != tt.allowed {
				t.Fatalf("allowed=%v err=%v", tt.allowed, err)
			}
		})
	}
	if doer.calls != 1 {
		t.Fatal("fixture transport was not cached or used safely")
	}
}
func TestSourceGateCopiesEventIDsAndPrincipalScopes(t *testing.T) {
	g, v, b := sourceFixture(t, SourceEvents)
	policy := g.policy
	policy.EventIDs = append([]string(nil), g.policy.EventIDs...)
	copied, err := NewSourceGate(SourceGateConfig{Enabled: true, Policy: policy, Bindings: b, Verifier: sourceCanonicalVerifier(SourceEvents, g.clock())})
	if err != nil {
		t.Fatal(err)
	}
	policy.EventIDs[0] = "other"
	if copied.policy.EventIDs[0] != "event-1" {
		t.Fatal("event selection alias")
	}
	p, err := g.Admit(context.Background(), sourceRequest("GET", "/api/v1/events?event_id=event-1&limit=2", ""))
	if err != nil {
		t.Fatal(err)
	}
	v.p.Scopes[0] = "admin"
	if err := p.Recheck(context.Background()); err != nil {
		t.Fatal("verifier result mutated admitted principal", err)
	}
}
