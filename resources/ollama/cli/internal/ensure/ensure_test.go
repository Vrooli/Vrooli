package ensure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

type fakeOllama struct {
	installed   map[string]bool
	pullScripts map[string][]string
	failPull    map[string]string
	pullCalls   int32
	tagCalls    int32
}

func (f *fakeOllama) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.tagCalls, 1)
		if r.Method != http.MethodGet {
			t.Errorf("tags: unexpected method %s", r.Method)
		}
		type model struct {
			Name string `json:"name"`
		}
		out := struct {
			Models []model `json:"models"`
		}{}
		for name, ok := range f.installed {
			if ok {
				out.Models = append(out.Models, model{Name: name})
			}
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.pullCalls, 1)
		if r.Method != http.MethodPost {
			t.Errorf("pull: unexpected method %s", r.Method)
		}
		var body struct {
			Name   string `json:"name"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("pull: decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if msg, shouldFail := f.failPull[body.Name]; shouldFail {
			_, _ = w.Write([]byte(`{"status":"pulling","error":"` + msg + `"}` + "\n"))
			return
		}
		script := f.pullScripts[body.Name]
		if len(script) == 0 {
			script = []string{`{"status":"pulling"}`, `{"status":"success"}`}
		}
		flusher, _ := w.(http.Flusher)
		for _, line := range script {
			_, _ = w.Write([]byte(line + "\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
		if _, hasSuccess := f.pullScripts[body.Name]; !hasSuccess {
			// After running a custom script we still mark this pull as applied
			// so subsequent /api/tags lookups (if any) see it.
			if f.installed == nil {
				f.installed = map[string]bool{}
			}
			f.installed[body.Name] = true
		}
	})
	return mux
}

func newTestClient(t *testing.T, fake *fakeOllama) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(fake.handler(t))
	return &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}, srv
}

func TestRun_NoModelsIsNoop(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	var buf bytes.Buffer
	if err := Run(context.Background(), Config{}, client, &buf, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(buf.String(), "nothing to do") {
		t.Errorf("expected no-op log, got %q", buf.String())
	}
	if fake.tagCalls != 0 {
		t.Errorf("tags should not be queried when no models requested, got %d", fake.tagCalls)
	}
}

func TestRun_AllModelsAlreadyInstalled(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{"qwen3:4b": true, "nomic-embed-text:latest": true}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}, {Name: "nomic-embed-text"}}}
	var buf bytes.Buffer
	if err := Run(context.Background(), cfg, client, &buf, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.pullCalls != 0 {
		t.Errorf("should not have pulled anything, got %d pulls", fake.pullCalls)
	}
	if !strings.Contains(buf.String(), "already installed") {
		t.Errorf("expected already-installed log, got %q", buf.String())
	}
}

func TestRun_PullsOnlyMissing(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{"qwen3:4b": true},
		pullScripts: map[string][]string{
			"nomic-embed-text:latest": {`{"status":"pulling manifest"}`, `{"status":"success"}`},
		},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}, {Name: "nomic-embed-text:latest"}}}
	var buf bytes.Buffer
	if err := Run(context.Background(), cfg, client, &buf, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.pullCalls != 1 {
		t.Errorf("expected exactly one pull, got %d", fake.pullCalls)
	}
	out := buf.String()
	if !strings.Contains(out, "pull nomic-embed-text:latest OK") {
		t.Errorf("expected success log for missing model, got %q", out)
	}
	if strings.Contains(out, "qwen3:4b: ") && strings.Contains(out, "pulling") {
		t.Errorf("should not have streamed progress for already-installed qwen3:4b: %q", out)
	}
}

func TestRun_PullFailureIsReported(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{},
		failPull:  map[string]string{"broken:1.0": "manifest not found"},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "broken:1.0"}}}
	var buf bytes.Buffer
	err := Run(context.Background(), cfg, client, &buf, nil)
	if err == nil {
		t.Fatal("expected error from failed pull")
	}
	if !strings.Contains(err.Error(), "1 model pull(s) failed") {
		t.Errorf("error missing aggregate count: %v", err)
	}
	if !strings.Contains(buf.String(), "FAILED") {
		t.Errorf("expected FAILED log, got %q", buf.String())
	}
}

func TestRun_ContextCancelAbortsPull(t *testing.T) {
	// Pull handler writes slowly so ctx cancel triggers mid-stream.
	fake := &fakeOllama{installed: map[string]bool{}}
	slowMux := http.NewServeMux()
	slowMux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}{})
	})
	slowMux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 10; i++ {
			_, err := fmt.Fprintf(w, `{"status":"pulling"}`+"\n")
			if err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	})
	srv := httptest.NewServer(slowMux)
	defer srv.Close()
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err := Run(ctx, Config{Models: []policy.DirectModelRequest{{Name: "slow"}}}, client, &bytes.Buffer{}, nil)
	if err == nil {
		t.Fatal("expected error when ctx is cancelled")
	}
	_ = fake
}

func TestParseConfig_AcceptsStringsAndObjects(t *testing.T) {
	raw := []byte(`{"model_roles": ["embedding.default", {"role":"chat.default","reason":"answer synthesis","required":false}], "models": ["qwen3:4b", {"name":"nomic-embed-text","tag":"latest","reason":"fixture","owner":"test"}]}`)
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if len(cfg.ModelRoles) != 2 {
		t.Fatalf("expected 2 model roles, got %d", len(cfg.ModelRoles))
	}
	if cfg.ModelRoles[0].Role != "embedding.default" {
		t.Errorf("model_roles[0] = %#v", cfg.ModelRoles[0])
	}
	if cfg.ModelRoles[1].IsRequired() {
		t.Errorf("model_roles[1] required = true, want false")
	}
	if len(cfg.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(cfg.Models))
	}
	if got := cfg.Models[0].Ref(); got != "qwen3:4b" {
		t.Errorf("model[0] ref = %q, want qwen3:4b", got)
	}
	if got := cfg.Models[1].Ref(); got != "nomic-embed-text:latest" {
		t.Errorf("model[1] ref = %q, want nomic-embed-text:latest", got)
	}
}

func TestRun_ResolvesRolesBeforePulling(t *testing.T) {
	fake := &fakeOllama{
		installed: map[string]bool{},
		pullScripts: map[string][]string{
			"nomic-embed-text:latest": {`{"status":"pulling manifest"}`, `{"status":"success"}`},
		},
	}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{ModelRoles: []policy.RoleRequest{{Role: "embedding.default"}}}
	var buf bytes.Buffer
	if err := Run(context.Background(), cfg, client, &buf, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.pullCalls != 1 {
		t.Fatalf("expected exactly one pull, got %d", fake.pullCalls)
	}
	out := buf.String()
	if !strings.Contains(out, "resolved role embedding.default -> nomic-embed-text:latest") {
		t.Fatalf("missing role resolution log: %q", out)
	}
	if !strings.Contains(out, "pull nomic-embed-text:latest OK") {
		t.Fatalf("missing pull success log: %q", out)
	}
}

func TestResolveBaseURL_Defaults(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("OLLAMA_PORT", "")
	if got := resolveBaseURL(); got != "http://127.0.0.1:11434" {
		t.Errorf("default base URL = %q", got)
	}
	t.Setenv("OLLAMA_HOST", "box")
	t.Setenv("OLLAMA_PORT", "9000")
	if got := resolveBaseURL(); got != "http://box:9000" {
		t.Errorf("host+port base URL = %q", got)
	}
	t.Setenv("OLLAMA_HOST", "box:8080")
	if got := resolveBaseURL(); got != "http://box:8080" {
		t.Errorf("host-with-port base URL = %q", got)
	}
	t.Setenv("OLLAMA_BASE_URL", "http://override:11/")
	if got := resolveBaseURL(); got != "http://override:11" {
		t.Errorf("override base URL = %q", got)
	}
}

func TestGeneratePassesOptionsAndReturnsEvalCount(t *testing.T) {
	var got struct {
		Model   string         `json:"model"`
		Prompt  string         `json:"prompt"`
		Images  []string       `json:"images"`
		Stream  bool           `json:"stream"`
		Think   *bool          `json:"think"`
		Options map[string]any `json:"options"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("path = %q, want /api/generate", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(GenerateResponse{Response: "ok", Done: true, EvalCount: 42})
	}))
	defer srv.Close()

	maxTokens := 123
	temperature := 0.25
	think := false
	numGPU := 0
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}
	resp, err := client.Generate(context.Background(), GenerateRequest{
		Model:       "llama3.2:1b",
		Prompt:      "hello",
		Think:       &think,
		Images:      []string{"aGVsbG8="},
		NumGPU:      &numGPU,
		NumPredict:  &maxTokens,
		Temperature: &temperature,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Response != "ok" || resp.EvalCount != 42 {
		t.Fatalf("response = %+v, want response ok with eval_count 42", resp)
	}
	if got.Model != "llama3.2:1b" || got.Prompt != "hello" || got.Stream {
		t.Fatalf("request body = %+v", got)
	}
	if len(got.Images) != 1 || got.Images[0] != "aGVsbG8=" {
		t.Fatalf("images = %#v, want one base64 image", got.Images)
	}
	if got.Think == nil || *got.Think != false {
		t.Fatalf("think = %v, want false", got.Think)
	}
	if got.Options["num_predict"] != float64(123) {
		t.Fatalf("num_predict = %#v, want 123", got.Options["num_predict"])
	}
	if got.Options["temperature"] != 0.25 {
		t.Fatalf("temperature = %#v, want 0.25", got.Options["temperature"])
	}
	if got.Options["num_gpu"] != float64(0) {
		t.Fatalf("num_gpu = %#v, want 0", got.Options["num_gpu"])
	}
}

func TestChatSendsMessagesOptionsAndThink(t *testing.T) {
	var got struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream  bool           `json:"stream"`
		Think   *bool          `json:"think"`
		Options map[string]any `json:"options"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q, want /api/chat", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":     map[string]string{"content": "summary"},
			"done_reason": "stop",
			"eval_count":  17,
		})
	}))
	defer srv.Close()

	maxTokens := 123
	temperature := 0.25
	think := false
	client := &Client{BaseURL: srv.URL, HTTP: http.DefaultClient}
	resp, err := client.Chat(context.Background(), ChatRequest{
		Model: "chat-model",
		Messages: []ChatMessage{
			{Role: "system", Content: "be concise"},
			{Role: "user", Content: "hello"},
		},
		NumPredict:  &maxTokens,
		Temperature: &temperature,
		Think:       &think,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Message.Content != "summary" || resp.DoneReason != "stop" || resp.EvalCount != 17 {
		t.Fatalf("response = %+v", resp)
	}
	if got.Model != "chat-model" || got.Stream {
		t.Fatalf("request model/stream = %q/%v", got.Model, got.Stream)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Content != "hello" {
		t.Fatalf("messages = %+v", got.Messages)
	}
	if got.Think == nil || *got.Think != false {
		t.Fatalf("think = %v, want false", got.Think)
	}
	if got.Options["num_predict"] != float64(123) {
		t.Fatalf("num_predict = %#v, want 123", got.Options["num_predict"])
	}
	if got.Options["temperature"] != 0.25 {
		t.Fatalf("temperature = %#v, want 0.25", got.Options["temperature"])
	}
}

// TestRun_AdmissionGateBlocksOnValidatorFailure proves the fail-closed gate:
// when the injected validator rejects a resolved tool role, ensure returns an
// error (the model is not silently seated).
func TestRun_AdmissionGateBlocksOnValidatorFailure(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{"qwen3:4b": true}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}}}
	called := false
	validator := func(ctx context.Context, roles []string) error {
		called = true
		return fmt.Errorf("model cannot tool-call")
	}
	err := Run(context.Background(), cfg, client, &bytes.Buffer{}, validator)
	if !called {
		t.Fatal("validator should run after models are confirmed installed")
	}
	if err == nil || !strings.Contains(err.Error(), "admission gate") {
		t.Fatalf("expected admission-gate error, got %v", err)
	}
}

// TestRun_AdmissionGatePassesWhenValidatorOK confirms a passing validator does
// not block ensure.
func TestRun_AdmissionGatePassesWhenValidatorOK(t *testing.T) {
	fake := &fakeOllama{installed: map[string]bool{"qwen3:4b": true}}
	client, srv := newTestClient(t, fake)
	defer srv.Close()

	cfg := Config{Models: []policy.DirectModelRequest{{Name: "qwen3:4b"}}}
	err := Run(context.Background(), cfg, client, &bytes.Buffer{}, func(context.Context, []string) error { return nil })
	if err != nil {
		t.Fatalf("passing validator must not block ensure: %v", err)
	}
}
