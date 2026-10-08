package identity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/cloudflareaccess"
	"github.com/vrooli/api-core/owneridentity"
)

const (
	testTeamDomain = "https://team.example.test"
	testAudience   = "hub-aud"
)

type accessFixture struct {
	key *rsa.PrivateKey
	now time.Time
	cfg cloudflareaccess.Config
}

func newAccessFixture(t *testing.T) accessFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "alg": "RS256", "kid": "k1",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
		}}})
	}))
	t.Cleanup(jwks.Close)
	now := time.Now().UTC()
	return accessFixture{key: key, now: now, cfg: cloudflareaccess.Config{TeamDomain: testTeamDomain, Audience: testAudience, JWKSURL: jwks.URL, Now: func() time.Time { return now }}}
}

func (f accessFixture) assertion(t *testing.T, key *rsa.PrivateKey, overrides map[string]any) string {
	t.Helper()
	claims := map[string]any{
		"iss": testTeamDomain, "aud": []string{testAudience}, "type": "app", "sub": "cf-user-1", "email": "operator@example.test",
		"iat": f.now.Add(-time.Minute).Unix(), "exp": f.now.Add(time.Hour).Unix(),
	}
	for name, value := range overrides {
		claims[name] = value
	}
	encode := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	input := encode(map[string]any{"alg": "RS256", "kid": "k1"}) + "." + encode(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// cookieVerifier stands in for scenario-authenticator: only "cookie-token"
// is a valid session.
var cookieVerifier = verifierFunc(func(_ context.Context, token string) (owneridentity.Identity, error) {
	if token == "cookie-token" {
		return owneridentity.Identity{Subject: "authenticator-user"}, nil
	}
	return owneridentity.Identity{}, owneridentity.ErrUnauthenticated
})

// ownerThrough runs a request through the owner middleware the way the API
// mounts it and returns what the Connect handlers would resolve as owner.
func ownerThrough(t *testing.T, provider authn.Provider, mutate func(*http.Request)) (owneridentity.Identity, error) {
	t.Helper()
	auth := OwnerAuthenticator{Providers: []authn.Provider{provider}, OwnerSubject: func(context.Context) string { return "operator" }}
	var owner owneridentity.Identity
	var ownerErr error
	handler := auth.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		owner, ownerErr = Owner(r.Context(), r.Header, cookieVerifier)
	}))
	req := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.conversations.ConversationsService/ListAsks", nil)
	if mutate != nil {
		mutate(req)
	}
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return owner, ownerErr
}

func TestAccessAssertionIsTheOwner(t *testing.T) {
	fixture := newAccessFixture(t)
	provider := NewAccessProvider(func(context.Context) (cloudflareaccess.Config, error) { return fixture.cfg, nil })
	owner, err := ownerThrough(t, provider, func(r *http.Request) {
		r.Header.Set(accessAssertionHeader, fixture.assertion(t, fixture.key, nil))
	})
	if err != nil {
		t.Fatalf("valid assertion refused: %v", err)
	}
	if owner.Subject != "operator" || owner.Email != "operator@example.test" {
		t.Fatalf("owner = %#v, want the recipient subject with the Access email", owner)
	}
}

func TestAccessAssertionWinsOverCookieSession(t *testing.T) {
	fixture := newAccessFixture(t)
	provider := NewAccessProvider(func(context.Context) (cloudflareaccess.Config, error) { return fixture.cfg, nil })
	owner, err := ownerThrough(t, provider, func(r *http.Request) {
		r.Header.Set(accessAssertionHeader, fixture.assertion(t, fixture.key, nil))
		r.AddCookie(&http.Cookie{Name: OwnerCookieName, Value: "cookie-token"})
	})
	if err != nil || owner.Subject != "operator" {
		t.Fatalf("owner = %#v, err = %v; want the Access owner", owner, err)
	}
}

func TestInvalidAccessAssertionIsRefusedEvenWithACookie(t *testing.T) {
	fixture := newAccessFixture(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"wrong audience": fixture.assertion(t, fixture.key, map[string]any{"aud": []string{"another-app"}}),
		"expired":        fixture.assertion(t, fixture.key, map[string]any{"exp": fixture.now.Add(-time.Hour).Unix()}),
		"bad signature":  fixture.assertion(t, otherKey, nil),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			provider := NewAccessProvider(func(context.Context) (cloudflareaccess.Config, error) { return fixture.cfg, nil })
			_, err := ownerThrough(t, provider, func(r *http.Request) {
				r.Header.Set(accessAssertionHeader, token)
				r.AddCookie(&http.Cookie{Name: OwnerCookieName, Value: "cookie-token"})
			})
			if !errors.Is(err, owneridentity.ErrUnauthenticated) {
				t.Fatalf("err = %v, want unauthenticated", err)
			}
		})
	}
}

func TestNoAccessAssertionFallsBackToTheCookieSession(t *testing.T) {
	resolved := false
	provider := NewAccessProvider(func(context.Context) (cloudflareaccess.Config, error) {
		resolved = true
		return cloudflareaccess.Config{}, errors.New("must not be called")
	})
	owner, err := ownerThrough(t, provider, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: OwnerCookieName, Value: "cookie-token"})
	})
	if err != nil || owner.Subject != "authenticator-user" {
		t.Fatalf("owner = %#v, err = %v; want the cookie session", owner, err)
	}
	if _, err := ownerThrough(t, provider, nil); !errors.Is(err, owneridentity.ErrUnauthenticated) {
		t.Fatalf("no credential err = %v, want unauthenticated", err)
	}
	if resolved {
		t.Fatal("a request without an assertion resolved the Access binding")
	}
}

// An unresolvable binding (tunnel-manager or Cloudflare down) grants
// nothing: the request is handled as if it carried no assertion.
func TestUnavailableAccessBindingGrantsNothing(t *testing.T) {
	fixture := newAccessFixture(t)
	provider := NewAccessProvider(func(context.Context) (cloudflareaccess.Config, error) {
		return cloudflareaccess.Config{}, errors.New("tunnel-manager unreachable")
	})
	token := fixture.assertion(t, fixture.key, nil)
	if _, err := ownerThrough(t, provider, func(r *http.Request) { r.Header.Set(accessAssertionHeader, token) }); !errors.Is(err, owneridentity.ErrUnauthenticated) {
		t.Fatalf("err = %v, want unauthenticated", err)
	}
	owner, err := ownerThrough(t, provider, func(r *http.Request) {
		r.Header.Set(accessAssertionHeader, token)
		r.AddCookie(&http.Cookie{Name: OwnerCookieName, Value: "cookie-token"})
	})
	if err != nil || owner.Subject != "authenticator-user" {
		t.Fatalf("owner = %#v, err = %v; want the cookie session", owner, err)
	}
}

func TestAccessBindingPrefersTheRuntimeEnvironment(t *testing.T) {
	env := map[string]string{"VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN": testTeamDomain, "VROOLI_CLOUDFLARE_ACCESS_AUDIENCE": testAudience}
	cfg, err := AccessBindingFromEnvironment(func(key string) string { return env[key] }, nil, "notification-hub")(context.Background())
	if err != nil || cfg.TeamDomain != testTeamDomain || cfg.Audience != testAudience {
		t.Fatalf("cfg = %#v, err = %v", cfg, err)
	}
}

func TestLoopbackOwnerIsOnlyTheExplicitLocalMode(t *testing.T) {
	providers := OwnerProvidersFromEnvironment(func(key string) string {
		if key == "VROOLI_AUTH_MODE" {
			return "personal_local"
		}
		return ""
	}, nil, "notification-hub")
	if len(providers) != 1 {
		t.Fatalf("providers = %d", len(providers))
	}
	owner, err := ownerThrough(t, providers[0], func(r *http.Request) { r.RemoteAddr = "127.0.0.1:5000" })
	if err != nil || owner.Subject != "operator" {
		t.Fatalf("loopback owner = %#v, err = %v", owner, err)
	}
	if _, err := ownerThrough(t, providers[0], func(r *http.Request) {
		r.RemoteAddr = "127.0.0.1:5000"
		r.Header.Set("Cf-Connecting-Ip", "203.0.113.9")
	}); err == nil {
		t.Fatal("a tunnel-forwarded request was treated as local")
	}
	if _, err := ownerThrough(t, providers[0], func(r *http.Request) { r.RemoteAddr = "192.0.2.4:5000" }); err == nil {
		t.Fatal("a remote request was treated as local")
	}
	defaults := OwnerProvidersFromEnvironment(func(string) string { return "" }, nil, "notification-hub")
	if _, ok := defaults[0].(*AccessProvider); !ok || len(defaults) != 1 {
		t.Fatalf("default providers = %#v, want Cloudflare Access only", defaults)
	}
}
