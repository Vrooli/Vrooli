package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"

	"github.com/vrooli/cli-core/cliutil/hostsem"
)

type fakeClient struct {
	embed    func(ctx context.Context, model, input string) ([]float64, error)
	generate func(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error)
	chat     func(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error)
}

func (f *fakeClient) Embed(ctx context.Context, model, input string) ([]float64, error) {
	return f.embed(ctx, model, input)
}

func (f *fakeClient) Generate(ctx context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
	return f.generate(ctx, in)
}

func (f *fakeClient) Chat(ctx context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
	return f.chat(ctx, in)
}

func newHandlers(t *testing.T, client *fakeClient, env map[string]string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	sem, err := hostsem.New(t.TempDir(), 4)
	if err != nil {
		t.Fatalf("hostsem.New: %v", err)
	}
	h := &Handlers{
		NewClient: func() Client { return client },
		Sem:       sem,
		GetEnv:    func(k string) string { return env[k] },
		Stdin:     strings.NewReader(""),
		Stdout:    stdout,
		Stderr:    stderr,
	}
	return h, stdout, stderr
}

func TestEmbedJSONOutput(t *testing.T) {
	client := &fakeClient{
		embed: func(_ context.Context, model, input string) ([]float64, error) {
			if model != "nomic-embed-text" {
				t.Errorf("model = %q", model)
			}
			if input != "hello" {
				t.Errorf("input = %q", input)
			}
			return []float64{0.1, 0.2, 0.3}, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, nil)
	if err := h.Embed([]string{"--model", "nomic-embed-text", "--input", "hello", "--json"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
	var got struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if len(got.Embedding) != 3 {
		t.Fatalf("embedding length = %d, want 3 (raw=%q)", len(got.Embedding), stdout.String())
	}
}

func TestEmbedRequiresModel(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, nil)
	err := h.Embed([]string{"--input", "hi"})
	if err == nil || !strings.Contains(err.Error(), "--role or --model") {
		t.Fatalf("expected model selection error, got %v", err)
	}
}

func TestEmbedRoleResolvesModel(t *testing.T) {
	client := &fakeClient{
		embed: func(_ context.Context, model, input string) ([]float64, error) {
			if model != "nomic-embed-text:latest" {
				t.Errorf("model = %q", model)
			}
			if input != "hello" {
				t.Errorf("input = %q", input)
			}
			return []float64{0.1}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Embed([]string{"--role", "embedding.default", "--input", "hello", "--json"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
}

func TestEmbedRejectsGenerateRole(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Embed([]string{"--role", "chat.default", "--input", "hello"})
	if err == nil || !strings.Contains(err.Error(), "without embedding capability") {
		t.Fatalf("expected capability error, got %v", err)
	}
}

func TestEmbedRejectsRoleAndModelTogether(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Embed([]string{"--role", "embedding.default", "--model", "nomic-embed-text", "--input", "hello"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
}

func TestEmbedRejectsUnknownRole(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Embed([]string{"--role", "missing.role", "--input", "hello"})
	if err == nil || !strings.Contains(err.Error(), `unknown model role "missing.role"`) {
		t.Fatalf("expected unknown role error, got %v", err)
	}
}

func TestEmbedSurfacesUpstreamError(t *testing.T) {
	client := &fakeClient{
		embed: func(context.Context, string, string) ([]float64, error) {
			return nil, errors.New("HTTP 503: ollama unavailable")
		},
	}
	h, _, _ := newHandlers(t, client, nil)
	err := h.Embed([]string{"--model", "x", "--input", "y", "--json"})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected upstream error to surface, got %v", err)
	}
}

func TestGenerateJSONOutput(t *testing.T) {
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "llama3.2:1b" || in.Prompt != "hi" {
				t.Errorf("unexpected args: %+v", in)
			}
			if in.NumPredict == nil || *in.NumPredict != 123 {
				t.Errorf("num_predict = %v, want 123", in.NumPredict)
			}
			if in.Temperature == nil || *in.Temperature != 0.25 {
				t.Errorf("temperature = %v, want 0.25", in.Temperature)
			}
			if in.Think == nil || *in.Think {
				t.Errorf("think = %v, want false", in.Think)
			}
			return ensure.GenerateResponse{Response: "hello!", EvalCount: 7}, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, nil)
	if err := h.Generate([]string{"--model", "llama3.2:1b", "--prompt", "hi", "--max-tokens", "123", "--temperature", "0.25", "--json"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var got struct {
		Response  string `json:"response"`
		EvalCount int    `json:"eval_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (raw=%q)", err, stdout.String())
	}
	if got.Response != "hello!" {
		t.Fatalf("response = %q", got.Response)
	}
	if got.EvalCount != 7 {
		t.Fatalf("eval_count = %d, want 7", got.EvalCount)
	}
}

func TestGenerateRoleResolvesModel(t *testing.T) {
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "qwen3.5:9b" {
				t.Errorf("model = %q", in.Model)
			}
			if in.Prompt != "hi" {
				t.Errorf("prompt = %q", in.Prompt)
			}
			return ensure.GenerateResponse{Response: "hello!", EvalCount: 1}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--json"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestGenerateRejectsExplicitContextWindowOverflow(t *testing.T) {
	var called bool
	client := &fakeClient{
		generate: func(context.Context, ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			called = true
			return ensure.GenerateResponse{}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	err := h.Generate([]string{"--role", "chat.default", "--prompt", "hi", "--max-tokens", "40000"})
	if err == nil || !strings.Contains(err.Error(), "exceeds context window") {
		t.Fatalf("expected context window error, got %v", err)
	}
	if called {
		t.Fatal("Generate called upstream after context window rejection")
	}
}

func TestGenerateAllowsUnknownDirectModelWithoutPolicyWindow(t *testing.T) {
	client := &fakeClient{
		generate: func(_ context.Context, in ensure.GenerateRequest) (ensure.GenerateResponse, error) {
			if in.Model != "local-test-model:latest" {
				t.Errorf("model = %q", in.Model)
			}
			return ensure.GenerateResponse{Response: "ok"}, nil
		},
	}
	h, _, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Generate([]string{"--model", "local-test-model:latest", "--prompt", "hi", "--max-tokens", "40000"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
}

func TestChatRoleResolvesModelAndForwardsControls(t *testing.T) {
	client := &fakeClient{
		chat: func(_ context.Context, in ensure.ChatRequest) (ensure.ChatResponse, error) {
			if in.Model != "qwen3.5:9b" {
				t.Errorf("model = %q", in.Model)
			}
			if len(in.Messages) != 2 {
				t.Fatalf("messages len = %d, want 2", len(in.Messages))
			}
			if in.Messages[0].Role != "system" || in.Messages[0].Content != "be concise" {
				t.Errorf("system message = %+v", in.Messages[0])
			}
			if in.Messages[1].Role != "user" || in.Messages[1].Content != "hello" {
				t.Errorf("user message = %+v", in.Messages[1])
			}
			if in.NumPredict == nil || *in.NumPredict != 123 {
				t.Errorf("num_predict = %v, want 123", in.NumPredict)
			}
			if in.Temperature == nil || *in.Temperature != 0.25 {
				t.Errorf("temperature = %v, want 0.25", in.Temperature)
			}
			if in.Think == nil || *in.Think != false {
				t.Errorf("think = %v, want false", in.Think)
			}
			var out ensure.ChatResponse
			out.Message.Content = "summary"
			out.DoneReason = "stop"
			out.EvalCount = 7
			return out, nil
		},
	}
	h, stdout, _ := newHandlers(t, client, policyEnv(t))
	if err := h.Chat([]string{"--role", "summarize.default", "--system", "be concise", "--prompt", "hello", "--max-tokens", "123", "--temperature", "0.25", "--json"}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	var got struct {
		Response   string `json:"response"`
		DoneReason string `json:"done_reason"`
		EvalCount  int    `json:"eval_count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (raw=%q)", err, stdout.String())
	}
	if got.Response != "summary" || got.DoneReason != "stop" || got.EvalCount != 7 {
		t.Fatalf("unexpected JSON: %+v", got)
	}
}

func TestChatRejectsEmbeddingRole(t *testing.T) {
	h, _, _ := newHandlers(t, &fakeClient{}, policyEnv(t))
	err := h.Chat([]string{"--role", "embedding.default", "--prompt", "hello"})
	if err == nil || !strings.Contains(err.Error(), "without chat capability") {
		t.Fatalf("expected capability error, got %v", err)
	}
}

func TestEmbedStdinInput(t *testing.T) {
	client := &fakeClient{
		embed: func(_ context.Context, _, input string) ([]float64, error) {
			if input != "from-stdin" {
				t.Errorf("input = %q", input)
			}
			return []float64{1}, nil
		},
	}
	h, _, _ := newHandlers(t, client, nil)
	h.Stdin = strings.NewReader("from-stdin")
	if err := h.Embed([]string{"--model", "m", "--input-stdin", "--json"}); err != nil {
		t.Fatalf("Embed: %v", err)
	}
}

func TestSemaphoreSerializesConcurrentInvocations(t *testing.T) {
	const slots = 2
	const callers = 8

	sem, err := hostsem.New(t.TempDir(), slots)
	if err != nil {
		t.Fatalf("hostsem: %v", err)
	}

	var (
		current int32
		peak    int32
	)
	client := &fakeClient{
		embed: func(ctx context.Context, _, _ string) ([]float64, error) {
			n := atomic.AddInt32(&current, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
					break
				}
			}
			time.Sleep(15 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			return []float64{0}, nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h := &Handlers{
				NewClient: func() Client { return client },
				Sem:       sem,
				GetEnv:    func(string) string { return "" },
				Stdin:     strings.NewReader(""),
				Stdout:    io.Discard,
				Stderr:    io.Discard,
			}
			if err := h.Embed([]string{"--model", "m", "--input", fmt.Sprintf("p%d", i), "--json"}); err != nil {
				t.Errorf("Embed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt32(&peak); got > slots {
		t.Fatalf("peak in-flight = %d, want <= %d", got, slots)
	}
}

func TestAcquireRespectsContextDeadline(t *testing.T) {
	sem, err := hostsem.New(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("hostsem: %v", err)
	}
	hold, err := sem.Acquire(context.Background())
	if err != nil {
		t.Fatalf("priming: %v", err)
	}
	defer hold()

	h := &Handlers{
		NewClient: func() Client { return &fakeClient{} },
		Sem:       sem,
		GetEnv: func(k string) string {
			if k == envAcquireTO {
				return "30ms"
			}
			return ""
		},
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	err = h.Embed([]string{"--model", "m", "--input", "x", "--json"})
	if err == nil || !strings.Contains(err.Error(), "acquire host semaphore") {
		t.Fatalf("expected acquire timeout error, got %v", err)
	}
}

func policyEnv(t *testing.T) map[string]string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("policy path: %v", err)
	}
	return map[string]string{"OLLAMA_MODEL_POLICY_PATH": path}
}
