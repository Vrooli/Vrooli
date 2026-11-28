package providers

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func testLookup(_ context.Context, host string) ([]net.IP, error) {
	if host == "example.test" {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	return nil, errors.New("unknown host")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestURLFetcherRejectsPrivateAndCredentialedDestinations(t *testing.T) {
	fetcher := NewURLFetcher(nil, testLookup, 1024, 2, time.Second)
	for _, rawURL := range []string{"http://127.0.0.1/", "http://169.254.169.254/latest", "http://localhost/", "file:///tmp/secret", "https://user:pass@example.test/"} {
		if _, err := fetcher.Fetch(context.Background(), rawURL); !errors.Is(err, ErrBlockedDestination) && !errors.Is(err, ErrUnsupportedScheme) {
			t.Fatalf("url %q err=%v", rawURL, err)
		}
	}
}

func TestURLFetcherValidatesRedirectsAndResponseSize(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"http://127.0.0.1/private"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})}
	fetcher := NewURLFetcher(client, testLookup, 4, 2, time.Second)
	if _, err := fetcher.Fetch(context.Background(), "https://example.test/start"); !errors.Is(err, ErrBlockedDestination) {
		t.Fatalf("redirect to private address was allowed: %v", err)
	}

	largeClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: io.NopCloser(strings.NewReader("12345")), Request: req}, nil
	})}
	if _, err := NewURLFetcher(largeClient, testLookup, 4, 2, time.Second).Fetch(context.Background(), "https://example.test/large"); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("oversize response err=%v", err)
	}
}
