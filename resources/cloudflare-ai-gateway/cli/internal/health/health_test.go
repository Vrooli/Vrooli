package health

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"resource-cloudflare-ai-gateway/cli/internal/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestProbeAuthenticated(t *testing.T) {
	t.Parallel()

	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.cloudflare.com/client/v4/accounts/acct-123/ai-gateway" {
			t.Fatalf("Probe() url = %q", req.URL.String())
		}
		if got := req.Header.Get("Authorization"); got != "Bearer token-456" {
			t.Fatalf("Probe() auth header = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	result, err := Probe(context.Background(), client, "https://api.cloudflare.com/client/v4/accounts", auth.Credentials{
		AccountID: "acct-123",
		APIToken:  "token-456",
	})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if result.Status != "reachable" {
		t.Fatalf("Probe().Status = %q, want reachable", result.Status)
	}
}

func TestProbeWithoutCredentials(t *testing.T) {
	t.Parallel()

	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.cloudflare.com/client/v4/accounts" {
			t.Fatalf("Probe() url = %q", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	result, err := Probe(context.Background(), client, "https://api.cloudflare.com/client/v4/accounts", auth.Credentials{})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if result.Status != "reachable" {
		t.Fatalf("Probe().Status = %q, want reachable", result.Status)
	}
	if result.Authenticated {
		t.Fatal("Probe().Authenticated = true, want false")
	}
}

func TestProbeUnreachable(t *testing.T) {
	t.Parallel()

	client := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp timeout")
	})

	result, err := Probe(context.Background(), client, "https://api.cloudflare.com/client/v4/accounts", auth.Credentials{})
	if err == nil {
		t.Fatal("Probe() error = nil, want non-nil")
	}
	if result.Status != "unreachable" {
		t.Fatalf("Probe().Status = %q, want unreachable", result.Status)
	}
}
