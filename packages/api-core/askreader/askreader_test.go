package askreader

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/serviceprincipal"
)

func TestVerifierConfigOverridesOnlyTrustedJWKSAndKeepsAskReaderRole(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "kid": "fixture", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer server.Close()

	now := time.Now().UTC()
	encode := func(value any) string {
		raw, _ := json.Marshal(value)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	issue := func(subject, audience string, scopes []string) string {
		unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "fixture"}) + "." + encode(map[string]any{
			"sub": subject, "user_id": subject, "actor_kind": "service",
			"aud": audience, "iss": "scenario-authenticator", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "scope": scopes,
		})
		digest := sha256.Sum256([]byte(unsigned))
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatal(err)
		}
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	}
	token := issue("treasury-ask-reader", Audience, []string{Scope})
	verifier := NewVerifierWithConfig(authn.JWTConfig{
		JWKSURL: server.URL, Audience: "caller-cannot-widen", Issuer: "caller-cannot-replace",
		Kind: apiidentity.ActorHuman,
	})
	if source, err := VerifiedSource(context.Background(), verifier, http.Header{"Authorization": []string{"Bearer " + token}}); err != nil || source != "treasury" {
		t.Fatalf("custom JWKS verifier did not accept the exact ask-reader role: source=%q err=%v", source, err)
	}
	wrongAudience := issue("treasury-ask-reader", "scenario-authenticator:agent-manager", []string{Scope})
	if _, err := VerifiedSource(context.Background(), verifier, http.Header{"Authorization": []string{"Bearer " + wrongAudience}}); err == nil {
		t.Fatal("verifier accepted a token for another audience")
	}
}

func TestPrincipalNamesRoundTripToTheirSource(t *testing.T) {
	for _, source := range []string{"vrooli-autoheal", "research-desk", "a1"} {
		got, ok := SourceOf(Principal(source))
		if !ok || got != source {
			t.Fatalf("SourceOf(Principal(%q)) = %q, %v", source, got, ok)
		}
	}
	if Principal(" research-desk ") != "research-desk-ask-reader" {
		t.Fatalf("Principal must trim the source: %q", Principal(" research-desk "))
	}
	for _, subject := range []string{"", "research-desk", "-ask-reader", "Research-ask-reader", "1desk-ask-reader", "research desk-ask-reader", "research-desk-ask-reader-x", "research-desk-ask-reader\n"} {
		if got, ok := SourceOf(subject); ok {
			t.Fatalf("SourceOf(%q) = %q, want it refused", subject, got)
		}
	}
}

func TestCredentialIdentityAddressesTheSourcesOwnEntry(t *testing.T) {
	want := "scenario-authenticator/service-principal/vrooli-autoheal-ask-reader/notification-hub-ask-reader"
	if got := CredentialIdentity("vrooli-autoheal"); got != want {
		t.Fatalf("CredentialIdentity = %q, want %q", got, want)
	}
}

// fixture is a Tokens over a scripted credential and exchanger.
type fixture struct {
	now        time.Time
	credential string
	credErr    error
	exchanges  atomic.Int32
	seen       []string
	result     Exchanged
	exchangeEr error
	mu         sync.Mutex
}

func newFixture() *fixture {
	f := &fixture{now: time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC), credential: "cred-1"}
	f.result = Exchanged{AccessToken: "tok-1", ExpiresAt: f.now.Add(10 * time.Minute), Subject: "research-desk-ask-reader", Audience: Audience, Scopes: []string{Scope}}
	return f
}

func (f *fixture) credentialSource() CredentialSource {
	return func(context.Context) (string, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.credential, f.credErr
	}
}

func (f *fixture) options() []Option {
	return []Option{
		WithClock(func() time.Time { return f.now }),
		WithExchanger(func(_ context.Context, credential string) (Exchanged, error) {
			f.exchanges.Add(1)
			f.mu.Lock()
			f.seen = append(f.seen, credential)
			f.mu.Unlock()
			return f.result, f.exchangeEr
		}),
	}
}

func (f *fixture) tokens() *Tokens {
	return NewTokens("research-desk", f.credentialSource(), f.options()...)
}

// issue is the operator's `service-principals sync --apply` landing while the
// scenario is running.
func (f *fixture) issue(credential string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.credential, f.credErr = credential, nil
}

func TestAMissingCredentialIsReportedAsNotIssuedAndNamesTheCommand(t *testing.T) {
	for name, set := range map[string]func(*fixture){
		"read error": func(f *fixture) { f.credErr = errors.New("no such entry") },
		"empty":      func(f *fixture) { f.credential = "  " },
	} {
		f := newFixture()
		set(f)
		_, err := f.tokens().Token(context.Background())
		if !errors.Is(err, ErrNotIssued) || !strings.Contains(err.Error(), serviceprincipal.IssueCommand) {
			t.Fatalf("%s: err = %v, want ErrNotIssued naming %s", name, err, serviceprincipal.IssueCommand)
		}
		if f.exchanges.Load() != 0 {
			t.Fatalf("%s: nothing may be exchanged without a credential", name)
		}
	}
}

func TestATokenThatIsNotExactlyTheScenariosAskReaderIdentityIsNeverUsed(t *testing.T) {
	cases := map[string]func(*Exchanged){
		"another principal": func(e *Exchanged) { e.Subject = "vrooli-autoheal-ask-reader" },
		"not a reader":      func(e *Exchanged) { e.Subject = "research-desk" },
		"another audience":  func(e *Exchanged) { e.Audience = "scenario-authenticator:agent-manager" },
		"no scope":          func(e *Exchanged) { e.Scopes = nil },
		"extra scope":       func(e *Exchanged) { e.Scopes = []string{Scope, "agent-manager:write"} },
		"other scope":       func(e *Exchanged) { e.Scopes = []string{"notification-hub:answer"} },
	}
	for name, mutate := range cases {
		f := newFixture()
		mutate(&f.result)
		got, err := f.tokens().Token(context.Background())
		if !errors.Is(err, ErrIdentityMismatch) || got != "" {
			t.Fatalf("%s: Token = %q, %v, want ErrIdentityMismatch", name, got, err)
		}
	}
	for name, mutate := range map[string]func(*Exchanged){
		"empty token":      func(e *Exchanged) { e.AccessToken = " " },
		"already expiring": func(e *Exchanged) { e.ExpiresAt = e.ExpiresAt.Add(-10*time.Minute + time.Second) },
	} {
		f := newFixture()
		mutate(&f.result)
		if got, err := f.tokens().Token(context.Background()); !errors.Is(err, ErrNoUsableToken) || got != "" {
			t.Fatalf("%s: Token = %q, %v, want ErrNoUsableToken", name, got, err)
		}
	}
}

func TestAnUnconfiguredTokenClientRefuses(t *testing.T) {
	var none *Tokens
	for name, tokens := range map[string]*Tokens{
		"nil":           none,
		"no source":     NewTokens("", func(context.Context) (string, error) { return "c", nil }),
		"no credential": NewTokens("research-desk", nil),
	} {
		if _, err := tokens.Token(context.Background()); err == nil {
			t.Fatalf("%s: want an error", name)
		}
	}
}
