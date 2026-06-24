package codecs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// ollamaLister — cache + degrade
// =============================================================================

func TestOllamaLister_CachesWithinTTL(t *testing.T) {
	calls := 0
	now := time.Unix(0, 0)
	fetch := func(ctx context.Context) ([]string, error) {
		calls++
		return []string{"ollama/gemma4:12b"}, nil
	}
	l := newOllamaListerForTest(fetch, 60*time.Second, func() time.Time { return now })

	if got := l.list(); len(got) != 1 || got[0] != "ollama/gemma4:12b" {
		t.Fatalf("first list = %v", got)
	}
	// Second call within TTL must not re-fetch.
	now = now.Add(30 * time.Second)
	_ = l.list()
	if calls != 1 {
		t.Fatalf("expected 1 fetch within TTL, got %d", calls)
	}
	// After TTL elapses, it re-fetches.
	now = now.Add(31 * time.Second)
	_ = l.list()
	if calls != 2 {
		t.Fatalf("expected 2 fetches after TTL, got %d", calls)
	}
}

func TestOllamaLister_DegradesOnError(t *testing.T) {
	now := time.Unix(0, 0)
	fail := true
	fetch := func(ctx context.Context) ([]string, error) {
		if fail {
			return nil, errors.New("connection refused")
		}
		return []string{"ollama/llama3:8b"}, nil
	}
	l := newOllamaListerForTest(fetch, time.Second, func() time.Time { return now })

	// Cold miss + error → empty, never panics.
	if got := l.list(); len(got) != 0 {
		t.Fatalf("cold error list = %v, want empty", got)
	}
	// Recovers once the daemon answers (after TTL).
	fail = false
	now = now.Add(2 * time.Second)
	if got := l.list(); len(got) != 1 || got[0] != "ollama/llama3:8b" {
		t.Fatalf("recovered list = %v", got)
	}
	// A later error degrades to the last-known list, not empty.
	fail = true
	now = now.Add(2 * time.Second)
	if got := l.list(); len(got) != 1 || got[0] != "ollama/llama3:8b" {
		t.Fatalf("degraded list = %v, want last-known", got)
	}
}

func TestOllamaLister_NilSafe(t *testing.T) {
	var l *ollamaLister
	if got := l.list(); got != nil {
		t.Fatalf("nil lister list = %v, want nil", got)
	}
}

// TestParseOllamaListJSON covers the SSOT payload decode: prefixing, sorting,
// blank-skip, and unknown-field tolerance.
func TestParseOllamaListJSON(t *testing.T) {
	out, err := parseOllamaListJSON([]byte(`{"models":["qwen3.5:9b","gemma4:12b","  "],"extra":"ignored"}`))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	want := []string{"ollama/gemma4:12b", "ollama/qwen3.5:9b"} // sorted, blank dropped
	if len(out) != len(want) {
		t.Fatalf("got %v want %v", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("got %v want %v", out, want)
		}
	}
}

// TestDefaultOllamaFetch_ExecBridge exercises the real exec path against a
// fake `resource-ollama` placed on PATH, proving the codec shells the SSOT
// (not raw HTTP) and maps its JSON to prefixed ids.
func TestDefaultOllamaFetch_ExecBridge(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, ollamaSSOTCommand)
	body := "#!/bin/sh\nif [ \"$1\" = \"models\" ] && [ \"$2\" = \"list\" ]; then\n  printf '%s' '{\"models\":[\"gemma4:12b\",\"llama3.1:8b\"]}'\n  exit 0\nfi\nexit 2\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := defaultOllamaFetch(context.Background())
	if err != nil {
		t.Fatalf("fetch err: %v", err)
	}
	want := []string{"ollama/gemma4:12b", "ollama/llama3.1:8b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v want %v", got, want)
	}
}

// TestDefaultOllamaFetch_DegradesWhenSSOTAbsent: a missing SSOT binary returns
// an error (the lister then degrades to the last-known list).
func TestDefaultOllamaFetch_DegradesWhenSSOTAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no resource-ollama here
	if _, err := defaultOllamaFetch(context.Background()); err == nil {
		t.Fatal("expected error when the SSOT binary is absent")
	}
}

func TestSplitOllamaModel(t *testing.T) {
	if bare, ok := splitOllamaModel("ollama/gemma4:12b"); !ok || bare != "gemma4:12b" {
		t.Fatalf("ollama-prefixed split = (%q,%v)", bare, ok)
	}
	if bare, ok := splitOllamaModel("gpt-5.5"); ok || bare != "gpt-5.5" {
		t.Fatalf("cloud model split = (%q,%v)", bare, ok)
	}
}

// =============================================================================
// codex — Ollama OSS routing + image attachments
// =============================================================================

func TestCodex_BuildArgs_OllamaRoutesToOSS(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID: uuid.New(),
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeCodex,
			Model:      "ollama/gemma4:12b",
		},
	})
	assertContainsSeq(t, args, "--oss", "--local-provider", "ollama")
	assertContainsSeq(t, args, "-m", "gemma4:12b")
}

func TestCodex_BuildArgs_CloudModelNoOSS(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-5.5"},
	})
	for _, a := range args {
		if a == "--oss" {
			t.Fatalf("cloud model must not emit --oss: %v", args)
		}
	}
	assertContainsSeq(t, args, "-m", "gpt-5.5")
}

func TestCodex_BuildArgs_ImageAttachments(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-5.5"},
		Attachments: []runner.Attachment{
			{FilePath: "/tmp/a.png"},
			{FilePath: "/tmp/b.jpg"},
			{FilePath: ""}, // skipped
		},
	})
	assertContainsSeq(t, args, "-i", "/tmp/a.png")
	assertContainsSeq(t, args, "-i", "/tmp/b.jpg")
	// stdin sentinel still last.
	if args[len(args)-1] != "-" {
		t.Fatalf("stdin sentinel not last: %v", args)
	}
}

func TestCodex_BuildContinueArgs_OllamaAndImages(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID:          uuid.New(),
		SessionID:      "thread-123",
		Prompt:         "go on",
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "ollama/llama3:8b"},
		Attachments:    []runner.Attachment{{FilePath: "/tmp/c.png"}},
	})
	assertContainsSeq(t, args, "--oss", "--local-provider", "ollama")
	assertContainsSeq(t, args, "-i", "/tmp/c.png")
	// session id positional precedes the prompt.
	sidIdx, promptIdx := indexOf(args, "thread-123"), indexOf(args, "go on")
	if sidIdx == -1 || promptIdx == -1 || sidIdx > promptIdx {
		t.Fatalf("session id must precede prompt: %v", args)
	}
}

func TestCodex_Capabilities_AppendsOllamaModels(t *testing.T) {
	c := NewCodexForTest()
	c.ollama = newOllamaListerForTest(
		func(ctx context.Context) ([]string, error) { return []string{"ollama/gemma4:12b"}, nil },
		time.Minute, func() time.Time { return time.Unix(0, 0) },
	)
	models := c.Capabilities().SupportedModels
	if indexOf(models, "gpt-5.5") == -1 {
		t.Fatalf("curated cloud model missing: %v", models)
	}
	if indexOf(models, "ollama/gemma4:12b") == -1 {
		t.Fatalf("local ollama model not appended: %v", models)
	}
}

func TestCodex_Capabilities_NilListerCloudOnly(t *testing.T) {
	c := NewCodexForTest() // ForTest leaves ollama nil
	models := c.Capabilities().SupportedModels
	for _, m := range models {
		if len(m) >= len(ollamaModelPrefix) && m[:len(ollamaModelPrefix)] == ollamaModelPrefix {
			t.Fatalf("nil lister must not surface ollama models: %v", models)
		}
	}
}

// =============================================================================
// opencode — image attachments
// =============================================================================

func TestOpenCode_BuildArgs_ImageAttachments(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:          uuid.New(),
		Prompt:         "describe",
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode, Model: "anthropic/claude-sonnet-4-6"},
		Attachments:    []runner.Attachment{{FilePath: "/tmp/x.png"}, {FilePath: ""}},
	})
	assertContainsSeq(t, args, "-f", "/tmp/x.png")
}

func TestOpenCode_BuildContinueArgs_ImageAttachments(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID:       uuid.New(),
		SessionID:   "sess-1",
		Prompt:      "more",
		Attachments: []runner.Attachment{{FilePath: "/tmp/y.png"}},
	})
	assertContainsSeq(t, args, "-f", "/tmp/y.png")
	assertContainsSeq(t, args, "--session", "sess-1")
}

// =============================================================================
// helpers
// =============================================================================

func indexOf(args []string, want string) int {
	for i, a := range args {
		if a == want {
			return i
		}
	}
	return -1
}

// assertContainsSeq asserts that the given ordered subsequence appears
// contiguously in args.
func assertContainsSeq(t *testing.T, args []string, seq ...string) {
	t.Helper()
	for i := 0; i+len(seq) <= len(args); i++ {
		match := true
		for j, s := range seq {
			if args[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Fatalf("expected contiguous %v in %v", seq, args)
}
