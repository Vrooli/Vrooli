package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenRouterProvider_ResolvesCredentialPerRequest(t *testing.T) {
	var gotKey string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotKey = r.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"echo ok"}}]}`)), Header: make(http.Header)}, nil
	})}
	p := &OpenRouterProvider{
		Role: "chat.default", Client: client, KeyResolver: StaticKeyResolver{Key: "authority-key"},
		Runner: func(context.Context, []string) ([]byte, error) { return []byte("vendor/model"), nil },
	}
	got, err := p.Generate(context.Background(), "system", "user")
	if err != nil || got != "echo ok" {
		t.Fatalf("Generate = %q, %v", got, err)
	}
	if gotKey != "Bearer authority-key" {
		t.Fatalf("authorization = %q, want authority key", gotKey)
	}
}

type meteredInferenceFixture struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	auth string
	last *inferencev1.RunRequest
}

func (f *meteredInferenceFixture) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.auth = request.Header().Get("Authorization")
	f.last = request.Msg
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `"echo routed"`, Provider: "fixture", Validated: true}), nil
}

func TestMeteredProvider_ForwardsConsumerTokenAndRemoteProfile(t *testing.T) {
	fixture := &meteredInferenceFixture{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	provider := NewMeteredProvider(server.URL)
	got, err := provider.Generate(WithConsumerToken(context.Background(), "signed-access"), "system", "user")
	if err != nil || got != "echo routed" {
		t.Fatalf("Generate = %q, %v", got, err)
	}
	if fixture.auth != "Bearer signed-access" {
		t.Fatalf("authorization = %q", fixture.auth)
	}
	if fixture.last == nil || fixture.last.GetProfile() != sharedv1.Profile_PROFILE_REMOTE_ONLY {
		t.Fatalf("request profile = %v, want remote_only", fixture.last.GetProfile())
	}
}

// TestOpenRouterProvider_ResolvesRole verifies the provider resolves a policy
// role (not a hard-coded slug) through the resource-openrouter seam.
func TestOpenRouterProvider_ResolvesRole(t *testing.T) {
	var gotArgs []string
	p := &OpenRouterProvider{
		Role: "chat.quality",
		Runner: func(_ context.Context, args []string) ([]byte, error) {
			gotArgs = args
			return []byte("vendor/some-model\n"), nil
		},
	}
	model, err := p.resolveModel(context.Background(), p.Role)
	if err != nil {
		t.Fatalf("resolveModel: %v", err)
	}
	if model != "vendor/some-model" {
		t.Errorf("expected trimmed resolved slug, got %q", model)
	}
	want := []string{"policy", "resolve", "--role", "chat.quality", "--field", "model"}
	if strings.Join(gotArgs, " ") != strings.Join(want, " ") {
		t.Errorf("unexpected resolve args: %v", gotArgs)
	}
}

// TestOpenRouterProvider_ResolveError surfaces resolver failures instead of
// falling back to a concrete model slug.
func TestOpenRouterProvider_ResolveError(t *testing.T) {
	p := &OpenRouterProvider{
		Role: "chat.default",
		Runner: func(_ context.Context, _ []string) ([]byte, error) {
			return nil, fmt.Errorf("boom")
		},
	}
	if _, err := p.resolveModel(context.Background(), p.Role); err == nil {
		t.Fatal("expected error when resolver fails")
	}
}

// TestNewOpenRouterProvider_DefaultRole verifies the default role is chat.default
// and no concrete model slug is baked in.
func TestNewOpenRouterProvider_DefaultRole(t *testing.T) {
	t.Setenv("WC_OPENROUTER_ROLE", "")
	p := NewOpenRouterProvider()
	if p.Role != "chat.default" {
		t.Errorf("expected default role chat.default, got %q", p.Role)
	}
}

func TestCheckProviderResponse_OK(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
	if err := checkProviderResponse(resp, "test"); err != nil {
		t.Errorf("200 should return nil, got %v", err)
	}
}

func TestCheckProviderResponse_NonOK(t *testing.T) {
	rec := httptest.NewRecorder()
	_, _ = rec.WriteString("rate limited")
	resp := rec.Result()
	resp.StatusCode = http.StatusTooManyRequests
	err := checkProviderResponse(resp, "openrouter")
	if err == nil {
		t.Fatal("non-200 should return error")
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Errorf("error should mention provider name, got %v", err)
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention status code, got %v", err)
	}
}
