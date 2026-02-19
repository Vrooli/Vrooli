package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeAIProvider is a test double for AIProvider.
type fakeAIProvider struct {
	name    string
	result  string
	err     error
	called  bool
}

func (f *fakeAIProvider) Name() string { return f.name }
func (f *fakeAIProvider) Generate(_ context.Context, _ string) (string, error) {
	f.called = true
	return f.result, f.err
}

// [REQ:P0-005a] AI Command Generation API - extractCommand tests
func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain command", "ls -la", "ls -la"},
		{"with whitespace", "  ls -la  \n", "ls -la"},
		{"markdown fenced bash", "```bash\nls -la\n```", "ls -la"},
		{"markdown fenced sh", "```sh\nfind . -name '*.go'\n```", "find . -name '*.go'"},
		{"markdown fenced plain", "```\ncat file.txt\n```", "cat file.txt"},
		{"multiline takes first", "ls -la\ncd /tmp\npwd", "ls -la"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommand(tt.input)
			if got != tt.want {
				t.Errorf("extractCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// [REQ:P0-005a] AI Command Generation API - provider chain failover
func TestAIProviderChainFailover(t *testing.T) {
	primary := &fakeAIProvider{name: "ollama", err: fmt.Errorf("connection refused")}
	fallback := &fakeAIProvider{name: "openrouter", result: "ls -la"}

	chain := NewAIProviderChain(primary, fallback)
	cmd, provider, err := chain.Generate(context.Background(), "list files")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "ls -la" {
		t.Errorf("got command %q, want %q", cmd, "ls -la")
	}
	if provider != "openrouter" {
		t.Errorf("got provider %q, want %q", provider, "openrouter")
	}
	if !primary.called {
		t.Error("primary provider was not called")
	}
	if !fallback.called {
		t.Error("fallback provider was not called")
	}
}

// [REQ:P0-005a] - primary provider success (no fallback)
func TestAIProviderChainPrimarySuccess(t *testing.T) {
	primary := &fakeAIProvider{name: "ollama", result: "docker ps"}
	fallback := &fakeAIProvider{name: "openrouter", result: "should not be called"}

	chain := NewAIProviderChain(primary, fallback)
	cmd, provider, err := chain.Generate(context.Background(), "list containers")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "docker ps" {
		t.Errorf("got command %q, want %q", cmd, "docker ps")
	}
	if provider != "ollama" {
		t.Errorf("got provider %q, want %q", provider, "ollama")
	}
	if fallback.called {
		t.Error("fallback provider should not have been called")
	}
}

// [REQ:P0-005a] - all providers fail
func TestAIProviderChainAllFail(t *testing.T) {
	primary := &fakeAIProvider{name: "ollama", err: fmt.Errorf("timeout")}
	fallback := &fakeAIProvider{name: "openrouter", err: fmt.Errorf("api key missing")}

	chain := NewAIProviderChain(primary, fallback)
	_, _, err := chain.Generate(context.Background(), "list files")

	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

// [REQ:P0-005a] - no providers configured
func TestAIProviderChainEmpty(t *testing.T) {
	chain := NewAIProviderChain()
	_, _, err := chain.Generate(context.Background(), "list files")

	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

// newTestServerWithAI creates a test server with a fake AI provider chain.
func newTestServerWithAI(providers ...AIProvider) *Server {
	return &Server{
		router:    nil,
		sessions:  NewSessionManagerWithFactory(newFakePTYFactory()),
		events:    NewEventLogger(100),
		metrics:   NewMetrics(),
		aiChain:   NewAIProviderChain(providers...),
		shortcuts: NewShortcutProfileStore(),
		aiConfig:  NewAIProviderConfigStore(),
	}
}

// [REQ:P0-005a] - handler returns generated command
func TestHandleAIGenerateSuccess(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", result: "find . -name '*.go'"}
	srv := newTestServerWithAI(provider)

	body := strings.NewReader(`{"prompt":"find all Go files"}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/generate", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAIGenerate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp AIGenerateResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Command != "find . -name '*.go'" {
		t.Errorf("got command %q, want %q", resp.Command, "find . -name '*.go'")
	}
	if resp.Provider != "ollama" {
		t.Errorf("got provider %q, want %q", resp.Provider, "ollama")
	}
}

// [REQ:P0-005a] - handler returns 400 for empty prompt
func TestHandleAIGenerateEmptyPrompt(t *testing.T) {
	srv := newTestServerWithAI(&fakeAIProvider{name: "ollama", result: "ls"})

	body := strings.NewReader(`{"prompt":""}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/generate", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAIGenerate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// [REQ:P0-005a] - handler returns 503 when all providers fail
func TestHandleAIGenerateAllProvidersFail(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", err: fmt.Errorf("connection refused")}
	srv := newTestServerWithAI(provider)

	body := strings.NewReader(`{"prompt":"list files"}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/generate", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAIGenerate(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Code != "ai_provider_unavailable" {
		t.Errorf("got error code %q, want %q", errResp.Code, "ai_provider_unavailable")
	}
	if !errResp.Retry {
		t.Error("expected retry=true for provider unavailable")
	}
}

// [REQ:P0-005a] - handler increments metrics
func TestHandleAIGenerateMetrics(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", result: "pwd"}
	srv := newTestServerWithAI(provider)

	before := srv.metrics.AIGenerations.Load()

	body := strings.NewReader(`{"prompt":"current directory"}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/generate", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAIGenerate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	after := srv.metrics.AIGenerations.Load()
	if after != before+1 {
		t.Errorf("AIGenerations: got %d, want %d", after, before+1)
	}
}

// [REQ:P0-005a] - handler includes terminal context in prompt
func TestHandleAIGenerateWithContext(t *testing.T) {
	var capturedPrompt string
	provider := &contextCapturingProvider{
		name:   "ollama",
		result: "ls /tmp",
		capture: func(prompt string) {
			capturedPrompt = prompt
		},
	}
	srv := newTestServerWithAI(provider)

	body := strings.NewReader(`{"prompt":"list temp files","context":"cwd: /home/user"}`)
	req := httptest.NewRequest("POST", "/api/v1/ai/generate", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleAIGenerate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(capturedPrompt, "cwd: /home/user") {
		t.Errorf("expected prompt to contain terminal context, got %q", capturedPrompt)
	}
}

// contextCapturingProvider is a test provider that captures the prompt.
type contextCapturingProvider struct {
	name    string
	result  string
	capture func(string)
}

func (c *contextCapturingProvider) Name() string { return c.name }
func (c *contextCapturingProvider) Generate(_ context.Context, prompt string) (string, error) {
	c.capture(prompt)
	return c.result, nil
}
