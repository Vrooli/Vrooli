package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	intai "web-console/internal/ai"

	"connectrpc.com/connect"

	aiH "web-console/handlers/ai"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	intworkspace "web-console/internal/workspace"
)

// fakeAIProvider is a test double for intai.Provider.
type fakeAIProvider struct {
	name   string
	result string
	err    error
	called bool
}

func (f *fakeAIProvider) Name() string { return f.name }
func (f *fakeAIProvider) Generate(_ context.Context, _, _ string) (string, error) {
	f.called = true
	return f.result, f.err
}

// [REQ:P0-005a] AI Command Generation API - aiH.ExtractCommand tests
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
			got := aiH.ExtractCommand(tt.input)
			if got != tt.want {
				t.Errorf("aiH.ExtractCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// [REQ:P0-005a] AI Command Generation API - provider chain failover
func TestAIProviderChainFailover(t *testing.T) {
	primary := &fakeAIProvider{name: "ollama", err: fmt.Errorf("connection refused")}
	fallback := &fakeAIProvider{name: "openrouter", result: "ls -la"}

	chain := intai.NewChain(primary, fallback)
	cmd, provider, err := chain.Generate(context.Background(), "system", "list files")
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

	chain := intai.NewChain(primary, fallback)
	cmd, provider, err := chain.Generate(context.Background(), "system", "list containers")
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

	chain := intai.NewChain(primary, fallback)
	_, _, err := chain.Generate(context.Background(), "system", "list files")

	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

// [REQ:P0-005a] - no providers configured
func TestAIProviderChainEmpty(t *testing.T) {
	chain := intai.NewChain()
	_, _, err := chain.Generate(context.Background(), "system", "list files")

	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

// newTestServerWithAI creates a test server with a fake AI provider chain.
func newTestServerWithAI(providers ...intai.Provider) *Server {
	srv := &Server{
		router:    nil,
		sessions:  newSessionManagerWithFactory(ptyfake.NewFactory()),
		events:    events.NewLogger(100),
		metrics:   metrics.New(),
		aiChain:   intai.NewChain(providers...),
		shortcuts: NewShortcutProfileStore(),
		aiConfig:  intai.NewMemConfigStore(),
		workspace: intworkspace.NewMemStore(),
	}
	srv.ai = intai.NewService(srv.aiChain, srv.aiConfig, nil, srv.events, &srv.metrics.AIGenerations, &srv.metrics.AISuggestions)
	return srv
}

// aiConnectIface is the public method surface NewConnectHandler returns.
// The concrete *connectHandler type is unexported in the ai package, but
// its exported methods are reachable through this interface.
type aiConnectIface interface {
	Generate(context.Context, *connect.Request[aiv1.GenerateRequest]) (*connect.Response[aiv1.GenerateResponse], error)
	Suggest(context.Context, *connect.Request[aiv1.SuggestRequest]) (*connect.Response[aiv1.SuggestResponse], error)
	GetConfig(context.Context, *connect.Request[aiv1.GetConfigRequest]) (*connect.Response[aiv1.GetConfigResponse], error)
	UpdateConfig(context.Context, *connect.Request[aiv1.UpdateConfigRequest]) (*connect.Response[aiv1.UpdateConfigResponse], error)
	GetHealth(context.Context, *connect.Request[aiv1.GetHealthRequest]) (*connect.Response[aiv1.GetHealthResponse], error)
}

func newAIConnectHandlerForServer(srv *Server) aiConnectIface {
	return aiH.NewConnectHandler(aiH.Deps{Service: &aiH.Adapter{Backend: srv.ai}})
}

// [REQ:P0-005a] - handler returns generated command
func TestConnect_AIGenerate_Success(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", result: "find . -name '*.go'"}
	srv := newTestServerWithAI(provider)
	h := newAIConnectHandlerForServer(srv)

	resp, err := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{Prompt: "find all Go files"}))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Msg.GetCommand() != "find . -name '*.go'" {
		t.Errorf("got command %q, want %q", resp.Msg.GetCommand(), "find . -name '*.go'")
	}
	if resp.Msg.GetProvider() != "ollama" {
		t.Errorf("got provider %q, want %q", resp.Msg.GetProvider(), "ollama")
	}
}

// [REQ:P0-005a] - handler returns InvalidArgument for empty prompt
func TestConnect_AIGenerate_EmptyPrompt(t *testing.T) {
	srv := newTestServerWithAI(&fakeAIProvider{name: "ollama", result: "ls"})
	h := newAIConnectHandlerForServer(srv)

	_, err := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{Prompt: ""}))
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", got)
	}
}

// [REQ:P0-005a] - handler returns Unavailable when all providers fail
func TestConnect_AIGenerate_AllProvidersFail(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", err: fmt.Errorf("connection refused")}
	srv := newTestServerWithAI(provider)
	h := newAIConnectHandlerForServer(srv)

	_, err := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{Prompt: "list files"}))
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", got)
	}
}

// [REQ:P0-005a] - handler increments metrics
func TestConnect_AIGenerate_Metrics(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", result: "pwd"}
	srv := newTestServerWithAI(provider)
	h := newAIConnectHandlerForServer(srv)

	before := srv.metrics.AIGenerations.Load()

	if _, err := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{Prompt: "current directory"})); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	after := srv.metrics.AIGenerations.Load()
	if after != before+1 {
		t.Errorf("AIGenerations: got %d, want %d", after, before+1)
	}
}

// [REQ:P0-005a] - handler includes terminal context in user prompt
func TestConnect_AIGenerate_WithContext(t *testing.T) {
	var capturedUserPrompt string
	provider := &contextCapturingProvider{
		name:   "ollama",
		result: "ls /tmp",
		capture: func(_, userPrompt string) {
			capturedUserPrompt = userPrompt
		},
	}
	srv := newTestServerWithAI(provider)
	h := newAIConnectHandlerForServer(srv)

	if _, err := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{
		Prompt:  "list temp files",
		Context: "cwd: /home/user",
	})); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(capturedUserPrompt, "cwd: /home/user") {
		t.Errorf("expected user prompt to contain terminal context, got %q", capturedUserPrompt)
	}
}

// contextCapturingProvider is a test provider that captures both prompts.
type contextCapturingProvider struct {
	name    string
	result  string
	capture func(systemPrompt, userPrompt string)
}

func (c *contextCapturingProvider) Name() string { return c.name }
func (c *contextCapturingProvider) Generate(_ context.Context, systemPrompt, userPrompt string) (string, error) {
	c.capture(systemPrompt, userPrompt)
	return c.result, nil
}

// TestConnect_AIGenerate_ExtractsCommand verifies the adapter applies aiH.ExtractCommand
// to raw AI output (strips code fences).
func TestConnect_AIGenerate_ExtractsCommand(t *testing.T) {
	provider := &fakeAIProvider{name: "ollama", result: "```bash\nls -la\n```"}
	srv := newTestServerWithAI(provider)
	h := newAIConnectHandlerForServer(srv)

	resp, err := h.Generate(context.Background(), connect.NewRequest(&aiv1.GenerateRequest{Prompt: "list files"}))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Msg.GetCommand() != "ls -la" {
		t.Errorf("got command %q, want %q", resp.Msg.GetCommand(), "ls -la")
	}
}

// TestExecuteAI_PassesSystemPrompt verifies system prompt reaches provider unchanged.
func TestExecuteAI_PassesSystemPrompt(t *testing.T) {
	var capturedSystem string
	provider := &contextCapturingProvider{
		name:   "ollama",
		result: "ok",
		capture: func(sp, _ string) {
			capturedSystem = sp
		},
	}
	srv := newTestServerWithAI(provider)

	_, _, err := srv.ai.Execute(context.Background(), "custom system prompt", "user input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSystem != "custom system prompt" {
		t.Errorf("system prompt was %q, want %q", capturedSystem, "custom system prompt")
	}
}

// TestExecuteAI_ReturnsRawOutput verifies no aiH.ExtractCommand is applied.
func TestExecuteAI_ReturnsRawOutput(t *testing.T) {
	raw := "```bash\nls -la\n```"
	provider := &fakeAIProvider{name: "ollama", result: raw}
	srv := newTestServerWithAI(provider)

	result, _, err := srv.ai.Execute(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != raw {
		t.Errorf("expected raw output %q, got %q", raw, result)
	}
}
