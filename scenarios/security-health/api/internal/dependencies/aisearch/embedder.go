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

// Embedder generates text embeddings. The production implementation shells out
// to `resource-ollama gateway embed`; tests inject fakes implementing this
// interface directly.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	Available(ctx context.Context) bool
}

type embedderRunner func(ctx context.Context, args []string, stdin []byte) ([]byte, error)

type cliEmbedder struct {
	bin   string
	model string
	run   embedderRunner
}

const defaultEmbedderBin = "resource-ollama"

// NewEmbedder returns the production CLI-backed Embedder.
func NewEmbedder(model string) Embedder {
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	return &cliEmbedder{
		bin:   defaultEmbedderBin,
		model: model,
		run:   defaultRunner,
	}
}

func defaultRunner(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
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

func (e *cliEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if e.run == nil {
		return nil, errors.New("embedder runner is not configured")
	}
	args := []string{e.bin, "gateway", "embed", "--model", e.model, "--json", "--input-stdin"}
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

func (e *cliEmbedder) Available(ctx context.Context) bool {
	if e.run == nil {
		return false
	}
	_, err := e.Embed(ctx, "ping")
	return err == nil
}
