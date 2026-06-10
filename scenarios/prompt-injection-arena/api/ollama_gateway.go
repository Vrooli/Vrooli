package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// runOllamaGateway shells out to `resource-ollama gateway <args>`. All Ollama
// daemon traffic goes through this CLI so the host-wide semaphore can bound
// fleet-wide parallelism — never hit Ollama HTTP directly.
func runOllamaGateway(ctx context.Context, args []string, stdin string) ([]byte, error) {
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

func ollamaGatewayEmbed(ctx context.Context, role, text string) ([]float64, error) {
	out, err := runOllamaGateway(ctx, []string{"gateway", "embed", "--role", role, "--json", "--input-stdin"}, text)
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

func ollamaGatewayGenerate(ctx context.Context, role, prompt string) (string, error) {
	out, err := runOllamaGateway(ctx, []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}, prompt)
	if err != nil {
		return "", fmt.Errorf("resource-ollama gateway generate failed: %w", err)
	}
	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return "", fmt.Errorf("decode gateway generate response: %w", err)
	}
	return decoded.Response, nil
}
