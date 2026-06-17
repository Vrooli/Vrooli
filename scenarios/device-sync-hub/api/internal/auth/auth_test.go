package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid token yields identity", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddResponse(http.StatusOK, []byte(`{"valid":true,"user_id":"owner-1","email":"o@x.io","roles":["user","admin"]}`))
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local/", Doer: doer})

		id, err := c.Validate(context.Background(), "tok")
		require.NoError(t, err)
		assert.Equal(t, "owner-1", id.OwnerID)
		assert.Equal(t, "o@x.io", id.Email)
		assert.True(t, id.HasRole("admin"))
		assert.False(t, id.HasRole("nope"))

		// Bearer header + canonical path.
		require.Len(t, doer.Requests, 1)
		req := doer.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "http://auth.local/api/v1/auth/validate", req.URL.String())
		assert.Equal(t, "Bearer tok", req.Header.Get("Authorization"))
	})

	t.Run("valid=false is unauthenticated", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddResponse(http.StatusOK, []byte(`{"valid":false}`))
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})

		_, err := c.Validate(context.Background(), "tok")
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("empty token short-circuits without a call", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})

		_, err := c.Validate(context.Background(), "   ")
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
		assert.Equal(t, int64(0), doer.Calls.Load())
	})

	t.Run("non-200 is unavailable not unauthenticated", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddResponse(http.StatusBadGateway, []byte(`upstream down`))
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})

		_, err := c.Validate(context.Background(), "tok")
		assert.ErrorIs(t, err, auth.ErrAuthUnavailable)
		assert.NotErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("transport error is unavailable", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddError(errors.New("dial tcp: refused"))
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})

		_, err := c.Validate(context.Background(), "tok")
		assert.ErrorIs(t, err, auth.ErrAuthUnavailable)
	})
}

func TestClientRevokeSession(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddResponse(http.StatusNoContent, nil)
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})

		require.NoError(t, c.RevokeSession(context.Background(), "sess-1"))
		require.Len(t, doer.Requests, 1)
		assert.Equal(t, http.MethodDelete, doer.Requests[0].Method)
		assert.Equal(t, "http://auth.local/api/v1/sessions/sess-1", doer.Requests[0].URL.String())
	})

	t.Run("404 is idempotent success", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddResponse(http.StatusNotFound, nil)
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})
		require.NoError(t, c.RevokeSession(context.Background(), "gone"))
	})

	t.Run("blank session id is a no-op", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})
		require.NoError(t, c.RevokeSession(context.Background(), ""))
		assert.Equal(t, int64(0), doer.Calls.Load())
	})

	t.Run("server error surfaces", func(t *testing.T) {
		t.Parallel()
		doer := &mocks.FakeDoer{}
		doer.AddResponse(http.StatusInternalServerError, nil)
		c := auth.NewClient(auth.Config{BaseURL: "http://auth.local", Doer: doer})
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
