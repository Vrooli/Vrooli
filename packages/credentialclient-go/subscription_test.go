package credentialclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type subscriptionCredentialClient struct {
	Client
	configured  bool
	refresh     string
	provisioned string
	statusErr   error
	resolveErr  error
}

func (c *subscriptionCredentialClient) Status(context.Context, string, string) (CredentialStatus, error) {
	if c.statusErr != nil {
		return CredentialStatus{}, c.statusErr
	}
	return CredentialStatus{Configured: c.configured, ProviderState: "available"}, nil
}

func (c *subscriptionCredentialClient) Resolve(context.Context, string, string) (string, error) {
	if c.resolveErr != nil {
		return "", c.resolveErr
	}
	return c.refresh, nil
}

func (c *subscriptionCredentialClient) Provision(_ context.Context, request ProvisionRequest) (ProvisionResponse, error) {
	c.provisioned = request.Value
	return ProvisionResponse{Status: "provisioned"}, nil
}

func TestConsumerSessionResolverRotatesSharedRefreshTokenAndCachesAccessInMemory(t *testing.T) {
	client := &subscriptionCredentialClient{configured: true, refresh: "refresh-old"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth/refresh" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Fatalf("content type = %q", got)
		}
		_, _ = w.Write([]byte(`{"access_token":"access-1","refresh_token":"refresh-new","expires_at":"2030-01-01T00:00:00Z"}`))
	}))
	defer server.Close()
	resolver := &ConsumerSessionResolver{Credentials: client, LPBSBaseURL: server.URL, HTTPClient: server.Client(), Now: func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) }}
	access, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if access.AccessToken != "access-1" || client.provisioned != "refresh-new" {
		t.Fatalf("access = %#v provisioned = %q", access, client.provisioned)
	}
	client.resolveErr = errors.New("should not read while access is warm")
	cached, err := resolver.Resolve(context.Background())
	if err != nil || cached.AccessToken != access.AccessToken {
		t.Fatalf("cached access = %#v, err = %v", cached, err)
	}
	resolver.Clear()
	if _, err := resolver.Resolve(context.Background()); !errors.Is(err, ErrCredentialAuthorityUnavailable) {
		t.Fatalf("cleared resolver error = %v", err)
	}
}

func TestConsumerSessionResolverDistinguishesAbsentAndUnavailableCredentials(t *testing.T) {
	absent := &ConsumerSessionResolver{Credentials: &subscriptionCredentialClient{configured: false}, LPBSBaseURL: "http://127.0.0.1"}
	if _, err := absent.Resolve(context.Background()); !errors.Is(err, ErrCredentialAbsent) {
		t.Fatalf("absent error = %v", err)
	}
	unavailable := &ConsumerSessionResolver{Credentials: &subscriptionCredentialClient{statusErr: errors.New("native store down")}, LPBSBaseURL: "http://127.0.0.1"}
	if _, err := unavailable.Resolve(context.Background()); !errors.Is(err, ErrCredentialAuthorityUnavailable) {
		t.Fatalf("unavailable error = %v", err)
	}
}

func TestConsumerSessionResolverDoesNotReuseAccessAcrossAuthorities(t *testing.T) {
	client := &subscriptionCredentialClient{configured: true, refresh: "refresh-shared"}
	var firstCalls, secondCalls int
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		firstCalls++
		_, _ = w.Write([]byte(`{"access_token":"access-first","refresh_token":"refresh-first","expires_at":"2030-01-01T00:00:00Z"}`))
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		secondCalls++
		_, _ = w.Write([]byte(`{"access_token":"access-second","refresh_token":"refresh-second","expires_at":"2030-01-01T00:00:00Z"}`))
	}))
	defer second.Close()

	resolver := &ConsumerSessionResolver{
		Credentials: client,
		HTTPClient:  first.Client(),
		Now:         func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	firstAccess, err := resolver.ResolveAt(context.Background(), first.URL)
	if err != nil {
		t.Fatal(err)
	}
	secondAccess, err := resolver.ResolveAt(context.Background(), second.URL)
	if err != nil {
		t.Fatal(err)
	}
	if firstAccess.AccessToken != "access-first" || secondAccess.AccessToken != "access-second" {
		t.Fatalf("access tokens = %#v, %#v", firstAccess, secondAccess)
	}
	if firstCalls != 1 || secondCalls != 1 {
		t.Fatalf("authority calls = first:%d second:%d", firstCalls, secondCalls)
	}
}

func TestConsumerSessionResolverSerializesSingleUseRefreshes(t *testing.T) {
	client := &subscriptionCredentialClient{configured: true, refresh: "refresh-shared"}
	var callsMu sync.Mutex
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callsMu.Lock()
		calls++
		callsMu.Unlock()
		_, _ = w.Write([]byte(`{"access_token":"access-one","refresh_token":"refresh-one","expires_at":"2030-01-01T00:00:00Z"}`))
	}))
	defer server.Close()
	resolver := &ConsumerSessionResolver{
		Credentials: client,
		HTTPClient:  server.Client(),
		LPBSBaseURL: server.URL,
		Now:         func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := resolver.Resolve(context.Background())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	if calls != 1 {
		t.Fatalf("refresh calls = %d, want one serialized refresh", calls)
	}
}
