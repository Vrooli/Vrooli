package sessions

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
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// fixtureIssuer stands in for scenario-authenticator: an RS256 key published
// over JWKS. Tokens exist only inside this test.
type fixtureIssuer struct {
	key  *rsa.PrivateKey
	jwks *httptest.Server
}

func newFixtureIssuer(t *testing.T) fixtureIssuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding
	body, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "kid": "fixture", "alg": "RS256", "use": "sig",
		"n": enc.EncodeToString(key.N.Bytes()), "e": enc.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return fixtureIssuer{key: key, jwks: srv}
}

func (f fixtureIssuer) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	enc := base64.RawURLEncoding
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "fixture", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	input := enc.EncodeToString(header) + "." + enc.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + enc.EncodeToString(sig)
}

func serviceClaims(aud string) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": "scenario-authenticator", "aud": []string{aud}, "sub": "agent-manager", "user_id": "agent-manager",
		"actor_kind": "service", "scope": []string{"web-console:write"},
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
}

// TestStableCreateOwnerRequiresVerifiedServicePrincipal drives the real
// authentication middleware and Connect handler: Agent Manager's service
// principal token for the web-console audience yields a pinned creator owner
// and an owner-bound reserved create, while anonymous, wrong-audience and
// forged callers receive no owner.
func TestStableCreateOwnerRequiresVerifiedServicePrincipal(t *testing.T) {
	issuer := newFixtureIssuer(t)
	store := sessionstore.NewInMemory()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	mgr.SetStore(store)
	adapter := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	path, handler := sessionsconnect.NewSessionsServiceHandler(NewConnectHandler(Deps{Service: adapter}))
	cfg := authn.Config{Providers: []authn.Provider{authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{
		Audience: "scenario-authenticator:web-console", JWKSURL: issuer.jwks.URL,
	})}}
	mux := http.NewServeMux()
	mux.Handle(path, authn.Middleware(cfg)(handler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	clientWith := func(bearer string) sessionsconnect.SessionsServiceClient {
		return sessionsconnect.NewSessionsServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(connect.UnaryInterceptorFunc(
			func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx2 context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					if bearer != "" {
						req.Header().Set("Authorization", "Bearer "+bearer)
					}
					return next(ctx2, req)
				}
			})))
	}
	listOwner := func(c sessionsconnect.SessionsServiceClient) string {
		t.Helper()
		resp, err := c.List(t.Context(), connect.NewRequest(&sessionsv1.ListRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Msg.SupportsStableCreateIdentity {
			t.Fatal("stable-create support not advertised")
		}
		return resp.Msg.StableCreateOwner
	}

	service := clientWith(issuer.sign(t, serviceClaims("scenario-authenticator:web-console")))
	if owner := listOwner(service); owner != "agent-manager" {
		t.Fatalf("authenticated service owner = %q, want agent-manager", owner)
	}
	if owner := listOwner(clientWith("")); owner != "" {
		t.Fatalf("anonymous caller received owner %q", owner)
	}
	if owner := listOwner(clientWith(issuer.sign(t, serviceClaims("scenario-authenticator:default")))); owner != "" {
		t.Fatalf("wrong-audience caller received owner %q", owner)
	}
	forger := newFixtureIssuer(t)
	if owner := listOwner(clientWith(forger.sign(t, serviceClaims("scenario-authenticator:web-console")))); owner != "" {
		t.Fatalf("unknown-key caller received owner %q", owner)
	}

	id := uuid.NewString()
	created, err := service.Create(t.Context(), connect.NewRequest(&sessionsv1.CreateRequest{
		RequestedSessionId: id, Shell: "/bin/sh", Backend: "standard", Owner: "forged",
		Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Delete(t.Context(), id) })
	if created.Msg.Session.Id != id || created.Msg.Session.Owner != "agent-manager" || created.Msg.Session.CreateFingerprint == "" {
		t.Fatalf("reserved create not pinned to the service principal: %+v", created.Msg.Session)
	}

	// Phase 5: when the verified service caller carries the human-owner header (a
	// tunnel-UI spawn), the stable-create owner AND the created session owner are
	// the HUMAN, not the service principal. The header is a creator-identity
	// label: the verified service principal still gated the call (see below),
	// web-console derives no authority from the header.
	clientWithHeader := func(bearer, human string) sessionsconnect.SessionsServiceClient {
		return sessionsconnect.NewSessionsServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(connect.UnaryInterceptorFunc(
			func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx2 context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					if bearer != "" {
						req.Header().Set("Authorization", "Bearer "+bearer)
					}
					if human != "" {
						req.Header().Set("X-Human-Owner-Subject", human)
					}
					return next(ctx2, req)
				}
			})))
	}
	svcToken := issuer.sign(t, serviceClaims("scenario-authenticator:web-console"))
	humanService := clientWithHeader(svcToken, "operator@example.test")
	if owner := listOwner(humanService); owner != "operator@example.test" {
		t.Fatalf("human-owner header did not set the stable-create owner: got %q", owner)
	}
	hid := uuid.NewString()
	hcreated, err := humanService.Create(t.Context(), connect.NewRequest(&sessionsv1.CreateRequest{
		RequestedSessionId: hid, Shell: "/bin/sh", Backend: "standard", Owner: "forged",
		Origin:                sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
		AgentRunIdentityToken: "test-only-child-bearer",
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Delete(t.Context(), hid) })
	if hcreated.Msg.Session.Owner != "operator@example.test" {
		t.Fatalf("human-session spawn not attributed to the operator: owner=%q", hcreated.Msg.Session.Owner)
	}

	// Guardrail 1 — the header is NEVER authority at web-console. An ANONYMOUS
	// caller that sets the human header gets NO owner: the header only relabels a
	// request the caller's own verified identity already authorized; with no
	// verified caller there is nothing to relabel.
	if owner := listOwner(clientWithHeader("", "operator@example.test")); owner != "" {
		t.Fatalf("anonymous caller with human header received owner %q — header must not confer identity", owner)
	}
}
