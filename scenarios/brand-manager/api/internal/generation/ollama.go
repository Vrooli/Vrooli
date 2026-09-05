package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// OllamaProvider routes text generation through the resource-ollama gateway
// CLI. All daemon traffic is funnelled through the CLI so the host-wide
// semaphore can bound fleet-wide parallelism — never call Ollama HTTP directly.
// Ported from the old aigen.OllamaProvider.
type OllamaProvider struct {
	Role string

	// Runner is an optional seam for tests. Production callers leave it nil and
	// the real resource-ollama binary is exec'd.
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

// NewOllamaProvider constructs an Ollama provider. An empty role defaults to
// "chat.default".
func NewOllamaProvider(role string) *OllamaProvider {
	if role == "" {
		role = "chat.default"
	}
	return &OllamaProvider{Role: role}
}

// Name returns "ollama".
func (o *OllamaProvider) Name() string { return "ollama" }

// Available checks if the resource-ollama daemon is reachable via the gateway
// CLI's status command.
func (o *OllamaProvider) Available(ctx context.Context) bool {
	if o.Runner != nil {
		_, err := o.Runner(ctx, []string{"status"}, "")
		return err == nil
	}
	return exec.CommandContext(ctx, "resource-ollama", "status").Run() == nil
}

// GenerateText calls `resource-ollama gateway generate`.
func (o *OllamaProvider) GenerateText(ctx context.Context, req TextRequest) (TextResponse, error) {
	role := req.Model
	if role == "" {
		role = o.Role
	}

	args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
	out, err := o.run(ctx, args, req.Prompt)
	if err != nil {
		return TextResponse{}, fmt.Errorf("resource-ollama gateway generate failed: %w", err)
	}
	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return TextResponse{}, fmt.Errorf("decode gateway generate response: %w", err)
	}
	return TextResponse{
		Text:     decoded.Response,
		Provider: "ollama",
		Model:    role,
	}, nil
}

func (o *OllamaProvider) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if o.Runner != nil {
		return o.Runner(ctx, args, stdin)
	}
	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

var _ Provider = (*OllamaProvider)(nil)
