package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"resource-ollama/cli/internal/ensure"

	"github.com/vrooli/cli-core/cliutil/hostsem"
)

type fakeClient struct {
	embed    func(ctx context.Context, model, input string) ([]float64, error)
	generate func(ctx context.Context, in ensure.GenerateRequest) (string, error)
}

func (f *fakeClient) Embed(ctx context.Context, model, input string) ([]float64, error) {
	return f.embed(ctx, model, input)
}

func (f *fakeClient) Generate(ctx context.Context, in ensure.GenerateRequest) (string, error) {
	return f.generate(ctx, in)
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
	if err == nil || !strings.Contains(err.Error(), "--model") {
		t.Fatalf("expected --model error, got %v", err)
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
		generate: func(_ context.Context, in ensure.GenerateRequest) (string, error) {
			if in.Model != "llama3.2:1b" || in.Prompt != "hi" {
				t.Errorf("unexpected args: %+v", in)
			}
			return "hello!", nil
		},
	}
	h, stdout, _ := newHandlers(t, client, nil)
	if err := h.Generate([]string{"--model", "llama3.2:1b", "--prompt", "hi", "--json"}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var got struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (raw=%q)", err, stdout.String())
	}
	if got.Response != "hello!" {
		t.Fatalf("response = %q", got.Response)
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
				Stdout:    io.Discard.(io.Writer),
				Stderr:    io.Discard.(io.Writer),
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
		GetEnv:    func(k string) string {
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
