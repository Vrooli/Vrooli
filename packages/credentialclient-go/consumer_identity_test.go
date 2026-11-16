package credentialclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveConsumerIdentityUsesLPBSAuthenticatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"user":{"email":" Verified@Example.com "}}`))
	}))
	defer server.Close()

	identity, err := ResolveConsumerIdentity(context.Background(), server.URL, "access-token", server.Client())
	if err != nil || identity != "verified@example.com" {
		t.Fatalf("ResolveConsumerIdentity() = %q, %v", identity, err)
	}
}

func TestResolveConsumerIdentityRejectsUnauthorizedAndRemoteHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	defer server.Close()
	if _, err := ResolveConsumerIdentity(context.Background(), server.URL, "access-token", server.Client()); !errors.Is(err, ErrSubscriptionUnauthorized) {
		t.Fatalf("unauthorized error = %v", err)
	}
	if _, err := ResolveConsumerIdentity(context.Background(), "http://identity.example", "access-token", nil); err == nil {
		t.Fatal("remote plain HTTP authority was accepted")
	}
}

func TestResolveConsumerIdentityDoesNotFollowRedirect(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("redirect forwarded authorization %q", got)
		}
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer source.Close()
	if _, err := ResolveConsumerIdentity(context.Background(), source.URL, "access-token", source.Client()); err == nil {
		t.Fatal("redirect response was accepted")
	}
	if called {
		t.Fatal("identity request followed redirect")
	}
}

func TestConsumerSessionResolverCachesIdentityPerIssuerAndToken(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"user":{"email":"user@example.com"}}`))
	}))
	defer server.Close()
	resolver := &ConsumerSessionResolver{HTTPClient: server.Client()}
	for i := 0; i < 2; i++ {
		identity, err := resolver.ResolveIdentityAt(context.Background(), server.URL, "same-token")
		if err != nil || identity != "user@example.com" {
			t.Fatalf("ResolveIdentityAt() = %q, %v", identity, err)
		}
	}
	if calls != 1 {
		t.Fatalf("identity authority calls = %d, want 1", calls)
	}
	if _, err := resolver.ResolveIdentityAt(context.Background(), server.URL, "rotated-token"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("identity authority calls after token rotation = %d, want 2", calls)
	}
}
