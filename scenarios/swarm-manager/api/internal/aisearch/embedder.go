package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Embedder turns text into a dense embedding vector. The production
// implementation shells out to `resource-ollama gateway embed`, which fronts
// the shared Ollama daemon with a host-wide cross-process semaphore. Tests
// substitute a fake runner via newEmbedderWithRunner.
//
// Decision boundary: "what does this text mean?" — the single named seam the
// reconciler and the write-through hooks consume.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	Available(ctx context.Context) bool
}

// embedderRunner abstracts CLI invocation so tests can avoid spawning
// processes. Production binds it to exec.CommandContext.
type embedderRunner func(ctx context.Context, args []string, stdin []byte) ([]byte, error)

// cliEmbedder is the production Embedder, backed by the resource-ollama gateway CLI.
type cliEmbedder struct {
	bin  string
	role string
	run  embedderRunner
}

const (
	defaultEmbedderBin  = "resource-ollama"
	defaultEmbedderRole = "embedding.default"
)

// NewEmbedder returns the production Embedder. The CLI binary `resource-ollama`
// must be on $PATH; if it is not, every Embed/Available call fails fast.
func NewEmbedder(role string) Embedder {
	if strings.TrimSpace(role) == "" {
		role = defaultEmbedderRole
	}
	return &cliEmbedder{
		bin:  defaultEmbedderBin,
		role: role,
		run:  defaultRunner,
	}
}

func newEmbedderWithRunner(role string, run embedderRunner) Embedder {
	if strings.TrimSpace(role) == "" {
		role = defaultEmbedderRole
	}
	return &cliEmbedder{
		bin:  defaultEmbedderBin,
		role: role,
		run:  run,
	}
}

func defaultRunner(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	// #nosec G702 -- no shell; args[0] is the fixed "resource-ollama" binary and
	// the remaining argv are internal flags. The text to embed is passed via
	// stdin, never as a command argument, so injection is not possible.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return stdout.Bytes(), nil
}

type cliEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed generates an embedding for the given text by invoking the gateway CLI.
func (e *cliEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if e.run == nil {
		return nil, errors.New("embedder runner is not configured")
	}
	args := []string{e.bin, "gateway", "embed", "--role", e.role, "--json", "--input-stdin"}
	out, err := e.run(ctx, args, []byte(text))
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway embed: %w", err)
	}
	var decoded cliEmbedResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(decoded.Embedding) == 0 {
		return nil, errors.New("embed response contained no vector")
	}
	return decoded.Embedding, nil
}

// Available reports whether the embedder can reach the gateway. Implemented as
// a tiny probe embed so it exercises the same code path as Embed.
func (e *cliEmbedder) Available(ctx context.Context) bool {
	if e.run == nil {
		return false
	}
	_, err := e.Embed(ctx, "ping")
	return err == nil
}
