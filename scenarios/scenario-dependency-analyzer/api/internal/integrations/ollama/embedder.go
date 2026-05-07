package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Embedder produces text embeddings via the resource-ollama gateway CLI. All
// daemon traffic is funnelled through that CLI so the host-wide semaphore can
// bound fleet-wide parallelism — never call Ollama HTTP directly.
type Embedder struct {
	model string

	// Runner is an optional seam for tests. Production callers leave it nil and
	// the default exec-based runner is used.
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

// NewEmbedder constructs an Embedder with an explicit model.
func NewEmbedder(model string) *Embedder {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "nomic-embed-text"
	}
	return &Embedder{model: model}
}

// NewEmbedderFromEnv constructs an Embedder picking the model from env vars in
// priority order.
func NewEmbedderFromEnv() *Embedder {
	model := firstEnv(
		"OLLAMA_EMBEDDING_MODEL",
		"QDRANT_EMBEDDING_MODEL_OVERRIDE",
		"QDRANT_EMBEDDING_MODEL",
	)
	return NewEmbedder(model)
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if e == nil {
		return nil, fmt.Errorf("ollama embedder not initialized")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("text required for embedding")
	}

	args := []string{"gateway", "embed", "--model", e.model, "--json", "--input-stdin"}
	out, err := e.run(ctx, args, text)
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway embed failed: %w", err)
	}

	var decoded struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode gateway embed response: %w", err)
	}
	if len(decoded.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned empty embedding")
	}
	return decoded.Embedding, nil
}

func (e *Embedder) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if e.Runner != nil {
		return e.Runner(ctx, args, stdin)
	}
	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
