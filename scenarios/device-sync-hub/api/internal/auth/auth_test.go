package auth_test

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
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/discovery"

	"device-sync-hub/internal/auth"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions/sessions_v1connect"
)

// --- test helpers ------------------------------------------------------------

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

func jwksJSON(pub *rsa.PublicKey) []byte {
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	set := map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "k1",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(eBytes),
	}}}
	b, _ := json.Marshal(set)
	return b
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// signWithAlg builds a compact JWS with the given header alg, always signing
// with RSA (so an alg!=RS256 token still carries a real-looking signature — the
// point of the algorithm-confusion guard).
func signWithAlg(t *testing.T, priv *rsa.PrivateKey, alg string, claims map[string]any) string {
	t.Helper()
	hb, _ := json.Marshal(map[string]string{"alg": alg, "typ": "JWT"})
	pb, _ := json.Marshal(claims)
	signingInput := b64(hb) + "." + b64(pb)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	require.NoError(t, err)
	return signingInput + "." + b64(sig)
}

func signRS256(t *testing.T, priv *rsa.PrivateKey, claims map[string]any) string {
	return signWithAlg(t, priv, "RS256", claims)
}

func ownerClaims(exp time.Time) map[string]any {
	return map[string]any{
		"user_id": "owner-1",
		"email":   "o@x.io",
		"roles":   []string{"user", "admin"},
		"iss":     auth.AuthScenarioSlug,
		"aud":     auth.AuthExpectedAudience,
		"iat":     time.Now().Add(-time.Minute).Unix(),
		"exp":     exp.Unix(),
	}
}

// jwksServer serves a (rotatable) JWKS and records hits.
type jwksServer struct {
	mu       sync.Mutex
	pub      *rsa.PublicKey
	jwksHits int
}

func newJWKSServer(pub *rsa.PublicKey) (*jwksServer, *httptest.Server) {
	s := &jwksServer{pub: pub}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		s.jwksHits++
		pub := s.pub
		s.mu.Unlock()
		_, _ = w.Write(jwksJSON(pub))
	})
	return s, httptest.NewServer(mux)
}

func (s *jwksServer) hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jwksHits
}

func (s *jwksServer) rotate(pub *rsa.PublicKey) {
	s.mu.Lock()
	s.pub = pub
	s.mu.Unlock()
}

type failingResolver struct{}

func (failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("scenario not running")
}

// --- Validate ----------------------------------------------------------------

func TestClientValidate(t *testing.T) {
	t.Run("valid token verifies locally and yields identity", func(t *testing.T) {
		key := testKey(t)
		js, srv := newJWKSServer(&key.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		tok := signRS256(t, key, ownerClaims(time.Now().Add(time.Hour)))
		id, err := c.Validate(context.Background(), tok)
		require.NoError(t, err)
		assert.Equal(t, "owner-1", id.OwnerID)
		assert.Equal(t, "o@x.io", id.Email)
		assert.True(t, id.HasRole("admin"))
		assert.False(t, id.HasRole("nope"))

		// Second call must reuse the cached key — no second JWKS fetch.
		_, err = c.Validate(context.Background(), tok)
		require.NoError(t, err)
		assert.Equal(t, 1, js.hits(), "JWKS must be fetched once and cached")
	})

	t.Run("expired token is unauthenticated", func(t *testing.T) {
		key := testKey(t)
		_, srv := newJWKSServer(&key.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		tok := signRS256(t, key, ownerClaims(time.Now().Add(-time.Minute)))
		_, err := c.Validate(context.Background(), tok)
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("token signed by a different key is unauthenticated", func(t *testing.T) {
		serverKey := testKey(t)
		attackerKey := testKey(t)
		_, srv := newJWKSServer(&serverKey.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client(), MinRefetch: time.Nanosecond})

		tok := signRS256(t, attackerKey, ownerClaims(time.Now().Add(time.Hour)))
		_, err := c.Validate(context.Background(), tok)
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("non-RS256 alg is rejected without verification (alg confusion / none)", func(t *testing.T) {
		key := testKey(t)
		_, srv := newJWKSServer(&key.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		hs := signWithAlg(t, key, "HS256", ownerClaims(time.Now().Add(time.Hour)))
		_, err := c.Validate(context.Background(), hs)
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)

		none := signWithAlg(t, key, "none", ownerClaims(time.Now().Add(time.Hour)))
		_, err = c.Validate(context.Background(), none)
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("wrong issuer is unauthenticated", func(t *testing.T) {
		key := testKey(t)
		_, srv := newJWKSServer(&key.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		claims := ownerClaims(time.Now().Add(time.Hour))
		claims["iss"] = "some-other-service"
		_, err := c.Validate(context.Background(), signRS256(t, key, claims))
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("wrong audience (other realm) is unauthenticated", func(t *testing.T) {
		key := testKey(t)
		_, srv := newJWKSServer(&key.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		// A token minted for a different realm's aud must be rejected even though
		// signature + issuer + expiry are all valid (tenant isolation).
		claims := ownerClaims(time.Now().Add(time.Hour))
		claims["aud"] = "scenario-authenticator:other-realm"
		_, err := c.Validate(context.Background(), signRS256(t, key, claims))
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("missing audience is unauthenticated", func(t *testing.T) {
		key := testKey(t)
		_, srv := newJWKSServer(&key.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		claims := ownerClaims(time.Now().Add(time.Hour))
		delete(claims, "aud")
		_, err := c.Validate(context.Background(), signRS256(t, key, claims))
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("malformed token is unauthenticated", func(t *testing.T) {
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver("http://x")})
		_, err := c.Validate(context.Background(), "not.a.jwt.at.all")
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("empty token short-circuits", func(t *testing.T) {
		js, srv := newJWKSServer(&testKey(t).PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})
		_, err := c.Validate(context.Background(), "   ")
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
		assert.Equal(t, 0, js.hits())
	})

	t.Run("unobtainable signing key is unavailable not unauthenticated", func(t *testing.T) {
		key := testKey(t)
		c := auth.NewClient(auth.Config{Resolver: failingResolver{}})
		tok := signRS256(t, key, ownerClaims(time.Now().Add(time.Hour)))
		_, err := c.Validate(context.Background(), tok)
		assert.ErrorIs(t, err, auth.ErrAuthUnavailable)
		assert.NotErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("key rotation: refetch lets a token signed by the new key verify", func(t *testing.T) {
		oldKey := testKey(t)
		js, srv := newJWKSServer(&oldKey.PublicKey)
		defer srv.Close()
		c := auth.NewClient(auth.Config{
			Resolver:   discovery.NewStaticResolver(srv.URL),
			Doer:       srv.Client(),
			MinRefetch: time.Nanosecond, // allow immediate refetch on miss
		})

		// Warm the cache with the old key.
		_, err := c.Validate(context.Background(), signRS256(t, oldKey, ownerClaims(time.Now().Add(time.Hour))))
		require.NoError(t, err)
		require.Equal(t, 1, js.hits())

		// Rotate the server key and present a token signed by the new key. The
		// first verify misses the cached (old) key; the client refetches and the
		// retry succeeds.
		newKey := testKey(t)
		js.rotate(&newKey.PublicKey)
		id, err := c.Validate(context.Background(), signRS256(t, newKey, ownerClaims(time.Now().Add(time.Hour))))
		require.NoError(t, err)
		assert.Equal(t, "owner-1", id.OwnerID)
		assert.Equal(t, 2, js.hits(), "a signature miss must trigger exactly one refetch")
	})
}

// fakeSessions is a configurable Connect SessionsService stub for the revoke
// path (migrated off the old REST DELETE).
type fakeSessions struct {
	sessionsconnect.UnimplementedSessionsServiceHandler
	mu        sync.Mutex
	revokedID []string
	revokeErr error
}

func (f *fakeSessions) RevokeSession(_ context.Context, req *connect.Request[sessionsv1.RevokeSessionRequest]) (*connect.Response[sessionsv1.RevokeSessionResponse], error) {
	f.mu.Lock()
	f.revokedID = append(f.revokedID, req.Msg.GetSessionId())
	f.mu.Unlock()
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	return connect.NewResponse(&sessionsv1.RevokeSessionResponse{}), nil
}

func (f *fakeSessions) revoked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.revokedID...)
}

func newSessionsStub(t *testing.T, impl *fakeSessions) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := sessionsconnect.NewSessionsServiceHandler(impl)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestClientRevokeSession(t *testing.T) {
	t.Run("ok calls the Connect SessionsService.RevokeSession", func(t *testing.T) {
		impl := &fakeSessions{}
		srv := newSessionsStub(t, impl)
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})

		require.NoError(t, c.RevokeSession(context.Background(), "sess-1"))
		assert.Equal(t, []string{"sess-1"}, impl.revoked())
	})

	t.Run("not-found is idempotent success", func(t *testing.T) {
		impl := &fakeSessions{revokeErr: connect.NewError(connect.CodeNotFound, errors.New("gone"))}
		srv := newSessionsStub(t, impl)
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})
		require.NoError(t, c.RevokeSession(context.Background(), "gone"))
	})

	t.Run("blank session id is a no-op", func(t *testing.T) {
		impl := &fakeSessions{}
		srv := newSessionsStub(t, impl)
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})
		require.NoError(t, c.RevokeSession(context.Background(), ""))
		assert.Empty(t, impl.revoked())
	})

	t.Run("server error surfaces as unavailable", func(t *testing.T) {
		impl := &fakeSessions{revokeErr: connect.NewError(connect.CodeInternal, errors.New("boom"))}
		srv := newSessionsStub(t, impl)
		c := auth.NewClient(auth.Config{Resolver: discovery.NewStaticResolver(srv.URL), Doer: srv.Client()})
		assert.ErrorIs(t, c.RevokeSession(context.Background(), "s"), auth.ErrAuthUnavailable)
	})
}

// fakeValidator is a hand-written Validator for middleware tests.
type fakeValidator struct {
	id  auth.Identity
	err error
}

func (f fakeValidator) Validate(context.Context, string) (auth.Identity, error) {
	return f.id, f.err
}
func (f fakeValidator) RevokeSession(context.Context, string) error { return nil }

func TestMiddleware(t *testing.T) {
	t.Parallel()

	newReq := func(authz string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		if authz != "" {
			req.Header.Set("Authorization", authz)
		}
		return req
	}

	t.Run("injects identity on valid token", func(t *testing.T) {
		t.Parallel()
		var seen auth.Identity
		var ok bool
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen, ok = auth.OwnerFromContext(r.Context())
		})
		mw := auth.Middleware(fakeValidator{id: auth.Identity{OwnerID: "owner-1"}}, nil)
		mw(next).ServeHTTP(httptest.NewRecorder(), newReq("Bearer good"))

		require.True(t, ok)
		assert.Equal(t, "owner-1", seen.OwnerID)
	})

	t.Run("no identity injected on invalid token but request proceeds", func(t *testing.T) {
		t.Parallel()
		called := false
		var ok bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			_, ok = auth.OwnerFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		mw := auth.Middleware(fakeValidator{err: auth.ErrUnauthenticated}, nil)
		rec := httptest.NewRecorder()
		mw(next).ServeHTTP(rec, newReq("Bearer bad"))

		assert.True(t, called, "open RPCs must still run when the token is bad")
		assert.False(t, ok, "no identity must be injected for a bad token")
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("no header leaves context empty", func(t *testing.T) {
		t.Parallel()
		var ok bool
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			_, ok = auth.OwnerFromContext(r.Context())
		})
		mw := auth.Middleware(fakeValidator{id: auth.Identity{OwnerID: "should-not-appear"}}, nil)
		mw(next).ServeHTTP(httptest.NewRecorder(), newReq(""))
		assert.False(t, ok)
	})
}

func TestRequireOwner(t *testing.T) {
	t.Parallel()

	_, err := auth.RequireOwner(context.Background())
	assert.ErrorIs(t, err, auth.ErrUnauthenticated)

	ctx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "o", ExpiresAt: time.Now()})
	id, err := auth.RequireOwner(ctx)
	require.NoError(t, err)
	assert.Equal(t, "o", id.OwnerID)
}

func TestBearerToken(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Bearer abc":  "abc",
		"bearer abc":  "abc",
		"BEARER  abc": "abc",
		"Basic abc":   "",
		"abc":         "",
		"":            "",
	}
	for in, want := range cases {
		assert.Equalf(t, want, auth.BearerToken(in), "input %q", in)
	}
}
