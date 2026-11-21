package embedder

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestOllamaEmbedUsesDefaultRole(t *testing.T) {
	var capturedArgs []string
	var capturedStdin string
	client := &Ollama{
		Runner: func(_ context.Context, args []string, stdin string) ([]byte, error) {
			capturedArgs = args
			capturedStdin = stdin
			return json.Marshal(struct {
				Embedding []float64 `json:"embedding"`
			}{Embedding: []float64{0.2, 0.4}})
		},
	}

	out, err := client.Embed(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected embedding length 2, got %d", len(out))
	}
	if capturedStdin != "test" {
		t.Fatalf("expected stdin %q, got %q", "test", capturedStdin)
	}
	joined := strings.Join(capturedArgs, " ")
	if !strings.Contains(joined, "gateway embed") {
		t.Fatalf("expected gateway embed subcommand, got %q", joined)
	}
	if !strings.Contains(joined, "--role embedding.default") {
		t.Fatalf("expected default role in args, got %q", joined)
	}
}

func TestOllamaEmbedPassesRole(t *testing.T) {
	var capturedArgs []string
	client := &Ollama{
		Role: "custom.embedding",
		Runner: func(_ context.Context, args []string, _ string) ([]byte, error) {
			capturedArgs = args
			return []byte(`{"embedding":[0.1]}`), nil
		},
	}
	if _, err := client.Embed(context.Background(), "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.Join(capturedArgs, " "), "--role custom.embedding") {
		t.Fatalf("expected --role custom.embedding in args, got %v", capturedArgs)
	}
}

func TestOllamaEmbedSurfacesRunnerError(t *testing.T) {
	client := &Ollama{
		Runner: func(context.Context, []string, string) ([]byte, error) {
			return nil, errors.New("boom")
		},
	}
	_, err := client.Embed(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped runner error, got %v", err)
	}
}
