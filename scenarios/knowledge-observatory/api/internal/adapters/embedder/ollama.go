package embedder

// DOC: docs/concepts/ARCHITECTURE.md#integrations
// DOC: docs/reference/configuration.md#api-runtime-configuration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Ollama embeds text via the resource-ollama gateway CLI. All HTTP traffic to
// the Ollama daemon is funnelled through that CLI so the host-wide semaphore
// can bound fleet-wide parallelism — never call the daemon directly.
type Ollama struct {
	Role string

	// Runner is an optional seam for tests. Production callers leave it nil and
	// the default exec-based runner is used.
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

func (o *Ollama) Embed(ctx context.Context, text string) ([]float64, error) {
	role := strings.TrimSpace(o.Role)
	if role == "" {
		role = "embedding.default"
	}

	args := []string{"gateway", "embed", "--role", role, "--json", "--input-stdin"}
	out, err := o.run(ctx, args, text)
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway embed failed: %w", err)
	}

	var decoded struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode gateway embed response: %w", err)
	}
	return decoded.Embedding, nil
}

func (o *Ollama) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if o.Runner != nil {
		return o.Runner(ctx, args, stdin)
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
