package byok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"audio-tools/internal/ai/summarizechain"
	"github.com/vrooli/api-core/scheduletest"
)

func TestOpenRouterContract(t *testing.T) {
	a := NewOpenRouterSummarize()
	if a.ID() != "openrouter" {
		t.Fatalf("id: %s", a.ID())
	}
	if a.Model() != "summarize.default" {
		t.Fatalf("model role: %q", a.Model())
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatal("empty key unavailable")
	}
}

func TestSummarizationSystemPrompt(t *testing.T) {
	if !strings.Contains(summarizationSystemPrompt("light"), "one sentence") {
		t.Fatal("light prompt missing one-sentence directive")
	}
	if !strings.Contains(summarizationSystemPrompt("heavy"), "compression") {
		t.Fatal("heavy prompt missing compression directive")
	}
	if !strings.Contains(summarizationSystemPrompt("moderate"), "2–4 sentences") {
		t.Fatal("default prompt missing 2-4 directive")
	}
}

func TestOpenRouterSummarizeUsesFilteredCLIAndParsesResponse(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "inherited-secret")
	t.Setenv("UNRELATED_SECRET", "must-not-be-forwarded")
	t.Setenv("OPENROUTER_API_BASE", "https://policy.example/v1")
	a := NewOpenRouterSummarize()
	clock := scheduletest.New(time.Unix(1000, 0))
	a.Clock = clock
	a.runCommand = func(ctx context.Context, command string, args, env []string, payload []byte, outputLimit int) ([]byte, error) {
		if command != "resource-openrouter" {
			t.Errorf("command=%q", command)
		}
		if fmt.Sprint(args) != "[generate --byok-json-stdin --json]" {
			t.Errorf("args=%q", args)
		}
		var req map[string]any
		if err := json.Unmarshal(payload, &req); err != nil {
			t.Fatal(err)
		}
		if req["api_key"] != "request-secret" || !strings.Contains(req["prompt"].(string), "one sentence") {
			t.Errorf("bad one-shot payload: %#v", req)
		}
		if !strings.HasSuffix(req["prompt"].(string), "\n\nsource text") {
			t.Errorf("prompt=%q", req["prompt"])
		}
		if outputLimit != openRouterOutputLimit {
			t.Errorf("output limit=%d", outputLimit)
		}
		childEnv := envMap(env)
		if _, ok := childEnv["OPENROUTER_API_KEY"]; ok {
			t.Errorf("credential leaked into environment: %v", childEnv)
		}
		if childEnv["OPENROUTER_API_BASE"] != "https://policy.example/v1" {
			t.Errorf("resource configuration not retained: %v", childEnv)
		}
		if _, exists := childEnv["UNRELATED_SECRET"]; exists {
			t.Error("unrelated secret was forwarded")
		}

		clock.Advance(125 * time.Millisecond)
		return []byte(`{"choices":[{"message":{"content":"summary"}}],"usage":{"prompt_tokens":10,"completion_tokens":3},"model":"resolved/model"}`), nil
	}

	res, err := a.Summarize(context.Background(), "request-secret", summarizechain.Request{Text: "source text", Level: "light"})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if res.Text != "summary" || res.ModelID != "resolved/model" {
		t.Fatalf("response: %+v", res)
	}
	if res.PromptTokens != 10 || res.OutputTokens != 3 || res.Latency != 125*time.Millisecond {
		t.Fatalf("metadata: %+v", res)
	}
}

func TestOpenRouterSummarizeUsesExplicitModelSelection(t *testing.T) {
	a := NewOpenRouterSummarize()
	a.Role = "summarize.fast"
	a.runCommand = func(_ context.Context, _ string, args, _ []string, _ []byte, _ int) ([]byte, error) {
		if fmt.Sprint(args) != "[generate --byok-json-stdin --json]" {
			t.Errorf("args=%q", args)
		}
		return []byte(`{"choices":[{"message":{"content":"ok"}}],"model":"vendor/model-x"}`), nil
	}
	if _, err := a.Summarize(context.Background(), "key", summarizechain.Request{Text: "source", Model: "vendor/model-x"}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRouterSummarizeCancellationAndFailuresAreSafe(t *testing.T) {
	a := NewOpenRouterSummarize()
	called := false
	a.runCommand = func(ctx context.Context, _ string, _ []string, _ []string, _ []byte, _ int) ([]byte, error) {
		called = true
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := a.Summarize(ctx, "key", summarizechain.Request{Text: "source"})
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("pre-cancel result err=%v runner-called=%v", err, called)
	}
	started := make(chan struct{})
	a.runCommand = func(ctx context.Context, _ string, _ []string, _ []string, _ []byte, _ int) ([]byte, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	activeCtx, cancelActive := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, runErr := a.Summarize(activeCtx, "key", summarizechain.Request{Text: "source"})
		result <- runErr
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("generation runner did not start")
	}
	cancelActive()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("active cancellation error=%v", err)
	}

	secret := "request-secret"
	a.runCommand = func(context.Context, string, []string, []string, []byte, int) ([]byte, error) {
		return nil, fmt.Errorf("child failed and echoed %s", secret)
	}
	_, err = a.Summarize(context.Background(), secret, summarizechain.Request{Text: "source"})
	if err == nil || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "child failed") {
		t.Fatalf("failure leaked child output: %v", err)
	}
}

func TestOpenRouterSummarizeBoundsAndValidatesCLIResponse(t *testing.T) {
	a := NewOpenRouterSummarize()
	a.runCommand = func(context.Context, string, []string, []string, []byte, int) ([]byte, error) {
		return []byte(strings.Repeat("x", openRouterOutputLimit+1)), nil
	}
	if _, err := a.Summarize(context.Background(), "key", summarizechain.Request{Text: "source"}); err == nil || !strings.Contains(err.Error(), "oversized") {
		t.Fatalf("oversized output err=%v", err)
	}
	a.runCommand = func(context.Context, string, []string, []string, []byte, int) ([]byte, error) {
		return []byte(`not-json`), nil
	}
	if _, err := a.Summarize(context.Background(), "key", summarizechain.Request{Text: "source"}); err == nil || !strings.Contains(err.Error(), "invalid response") {
		t.Fatalf("invalid response err=%v", err)
	}
}

func TestOpenRouterChildEnvironmentFiltersInheritedCredentials(t *testing.T) {
	env := envMap(openRouterChildEnv([]string{
		"PATH=/bin", "HOME=/tmp/home", "OPENROUTER_API_KEY=inherited",
		"OPENROUTER_API_BASE=https://openrouter.invalid/v1", "UNRELATED_TOKEN=secret",
	}))
	if env["PATH"] != "/bin" || env["HOME"] != "/tmp/home" || env["OPENROUTER_API_BASE"] != "https://openrouter.invalid/v1" {
		t.Fatalf("safe runtime configuration missing: %v", env)
	}
	if _, ok := env["OPENROUTER_API_KEY"]; ok {
		t.Fatalf("inherited key forwarded: %v", env)
	}
	if _, ok := env["UNRELATED_TOKEN"]; ok {
		t.Fatalf("unrelated token forwarded: %v", env)
	}
}

func envMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		name, content, ok := strings.Cut(value, "=")
		if ok {
			result[name] = content
		}
	}
	return result
}
