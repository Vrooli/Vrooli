package identity_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"treasury/internal/identity"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

// [REQ:TRS-P0-005] Every missing, rejected, malformed, or unreachable identity fails closed.
func TestHTTPVerifierFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		token string
		doer  doerFunc
	}{
		{"absent token", "", func(*http.Request) (*http.Response, error) { t.Fatal("network must not be called"); return nil, nil }},
		{"unreachable authority", "token", func(*http.Request) (*http.Response, error) { return nil, errors.New("connection refused") }},
		{"expired or invalid token", "token", func(*http.Request) (*http.Response, error) {
			return response(http.StatusUnauthorized, `{"valid":false,"error":"expired"}`), nil
		}},
		{"malformed response", "token", func(*http.Request) (*http.Response, error) { return response(http.StatusOK, `{`), nil }},
		{"missing claims", "token", func(*http.Request) (*http.Response, error) { return response(http.StatusOK, `{"valid":true}`), nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier, err := identity.NewHTTPVerifier("http://agent-manager", test.doer)
			require.NoError(t, err)
			_, err = verifier.Verify(context.Background(), test.token)
			require.ErrorIs(t, err, identity.ErrUnverifiable)
		})
	}
}

func TestHTTPVerifierReturnsOnlyAuthorityClaims(t *testing.T) {
	var calls int
	verifier, err := identity.NewHTTPVerifier("http://agent-manager/", doerFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/api/v1/identity/verify", req.URL.Path)
		return response(http.StatusOK, `{"valid":true,"claims":{"run_id":"run-1","subject":"operator:1","scopes":["treasury:spend"],"meta":{"persona_id":"persona-1"}}}`), nil
	}))
	require.NoError(t, err)
	for range 2 {
		claims, err := verifier.Verify(context.Background(), "opaque")
		require.NoError(t, err)
		require.Equal(t, "operator:1", claims.Subject)
		require.Equal(t, []string{"treasury:spend"}, claims.Scopes)
		require.Equal(t, "persona-1", claims.Meta["persona_id"])
	}
	require.Equal(t, 2, calls, "claims are verified live on every request and never cached")
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}
