package providers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type meteredEgressStub struct {
	provider string
	role     string
	profile  sharedv1.Profile
	status   int
}

func (s *meteredEgressStub) Record(_ context.Context, provider, role string, profile sharedv1.Profile, status int) error {
	s.provider, s.role, s.profile, s.status = provider, role, profile, status
	return nil
}

func TestMeteredClientForwardsBearerAndUsesProfileEndpoint(t *testing.T) {
	var gotAuthorization string
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		var body []byte
		body, _ = io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"run-1","content":"{\"ok\":true}","prompt_tokens":4,"completion_tokens":2,"credits_charged":8}`))
	}))
	defer server.Close()
	egress := &meteredEgressStub{}
	client := NewMeteredClient(MeteredClientOptions{
		BaseURL:     "http://invalid.example",
		ProfileURLs: map[sharedv1.Profile]string{sharedv1.Profile_PROFILE_REMOTE_ONLY: server.URL},
		Egress:      egress,
	})

	result, err := client.Run(WithAccessToken(context.Background(), "Bearer consumer-token"), MeteredRequest{
		Role: "classify.fast", Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY,
		Messages: []MeteredMessage{{Role: "user", Content: "classify this"}}, ConstraintsJSON: `{"type":"boolean"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "run-1" || result.CreditsCharged != 8 {
		t.Fatalf("result = %+v", result)
	}
	if gotAuthorization != "Bearer consumer-token" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
	if !strings.Contains(gotBody, `"role":"classify.fast"`) || strings.Contains(gotBody, `"model"`) {
		t.Fatalf("request body exposed an invalid contract: %s", gotBody)
	}
	if egress.provider != ProviderMetered || egress.role != "classify.fast" || egress.status != http.StatusOK {
		t.Fatalf("egress = %+v", egress)
	}
}

func TestMeteredClientRequiresToken(t *testing.T) {
	client := NewMeteredClient(MeteredClientOptions{BaseURL: "http://invalid.example"})
	_, err := client.Run(context.Background(), MeteredRequest{Role: "classify.fast", Messages: []MeteredMessage{{Role: "user", Content: "x"}}})
	if err == nil || !strings.Contains(err.Error(), "access token") {
		t.Fatalf("error = %v", err)
	}
}

func TestMeteredClientResolvesSharedAccessTokenOnlyWhenCallerDidNotForwardOne(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resolved","content":"{\"ok\":true}"}`))
	}))
	defer server.Close()
	resolverCalls := 0
	client := NewMeteredClient(MeteredClientOptions{
		BaseURL: server.URL,
		ResolveAccessToken: func(_ context.Context, baseURL string) (string, error) {
			resolverCalls++
			if baseURL != server.URL {
				t.Fatalf("resolver base URL = %q", baseURL)
			}
			return "Bearer shared-access", nil
		},
	})
	request := MeteredRequest{Role: "classify.fast", Messages: []MeteredMessage{{Role: "user", Content: "x"}}}
	if _, err := client.Run(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer shared-access" || resolverCalls != 1 {
		t.Fatalf("authorization = %q, resolver calls = %d", gotAuthorization, resolverCalls)
	}
	if _, err := client.Run(WithAccessToken(context.Background(), "Bearer caller-access"), request); err != nil {
		t.Fatal(err)
	}
	if gotAuthorization != "Bearer caller-access" || resolverCalls != 1 {
		t.Fatalf("forwarded authorization = %q, resolver calls = %d", gotAuthorization, resolverCalls)
	}
}

func TestMeteredClientOpensCircuitAfterBoundedFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer server.Close()
	client := NewMeteredClient(MeteredClientOptions{BaseURL: server.URL, FailureLimit: 2, Cooldown: time.Hour})
	request := MeteredRequest{Role: "classify.fast", Messages: []MeteredMessage{{Role: "user", Content: "x"}}}
	ctx := WithAccessToken(context.Background(), "Bearer token")
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := client.Run(ctx, request); err == nil {
			t.Fatal("expected upstream failure")
		}
	}
	_, err := client.Run(ctx, request)
	if err == nil || !strings.Contains(err.Error(), "circuit breaker") {
		t.Fatalf("third request error = %v", err)
	}
}
