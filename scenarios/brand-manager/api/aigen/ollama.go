// Ollama provider for local AI generation via the resource-ollama gateway CLI.
// [REQ:BM-REQ-AI-CHAIN]
package aigen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// OllamaProvider routes text generation through the resource-ollama gateway
// CLI. All daemon traffic is funnelled through the CLI so the host-wide
// semaphore can bound fleet-wide parallelism — never call Ollama HTTP directly.
// [REQ:BM-REQ-AI-CHAIN]
type OllamaProvider struct {
	Role string

	// Runner is an optional seam for tests. Production callers leave it nil.
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

func NewOllamaProvider(_baseURL, role string) *OllamaProvider {
	if role == "" {
		role = "chat.default"
	}
	return &OllamaProvider{Role: role}
}

// Name returns "ollama".
func (o *OllamaProvider) Name() string { return "ollama" }

// Available checks if the resource-ollama daemon is reachable.
func (o *OllamaProvider) Available(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "resource-ollama", "status")
	return cmd.Run() == nil
}

// GenerateText calls resource-ollama gateway generate. [REQ:BM-REQ-AI-TEXT]
func (o *OllamaProvider) GenerateText(ctx context.Context, req TextRequest) (*TextResponse, error) {
	role := req.Model
	if role == "" {
		role = o.Role
	}

	args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
	out, err := o.run(ctx, args, req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway generate failed: %w", err)
	}
	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode gateway generate response: %w", err)
	}
	return &TextResponse{
		Text:     decoded.Response,
		Provider: "ollama",
		Model:    role,
	}, nil
}

// GenerateImage is not supported by Ollama (text-only). [REQ:BM-REQ-AI-IMAGE]
func (o *OllamaProvider) GenerateImage(_ context.Context, _ ImageRequest) (*ImageResponse, error) {
	return nil, fmt.Errorf("ollama does not support image generation")
}

func (o *OllamaProvider) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
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
