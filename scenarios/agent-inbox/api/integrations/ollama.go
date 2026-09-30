package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"agent-inbox/config"
)

// OllamaClient generates short text via the resource-ollama gateway CLI. All
// daemon traffic is funnelled through that CLI so the host-wide semaphore can
// bound fleet-wide parallelism — never call Ollama HTTP directly.
type OllamaClient struct {
	cfg config.NamingConfig

	// Runner is an optional seam for tests. Production callers leave it nil and
	// the default exec-based runner is used.
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

// NewOllamaClient creates a new client using default naming configuration.
func NewOllamaClient() *OllamaClient {
	return NewOllamaClientWithConfig(config.Default().Integration.Naming)
}

// NewOllamaClientWithConfig creates a client with explicit naming configuration.
// Used by tests to inject a Runner via direct field assignment.
func NewOllamaClientWithConfig(namingCfg config.NamingConfig) *OllamaClient {
	return &OllamaClient{cfg: namingCfg}
}

// GenerateChatName generates a concise, descriptive name for a conversation.
// Configuration is controlled via NamingConfig (Temperature, MaxTokens).
func (c *OllamaClient) GenerateChatName(ctx context.Context, conversationSummary string) (string, error) {
	prompt := fmt.Sprintf(`Generate a very short, descriptive title (3-6 words max) for this conversation.
Return ONLY the title, no quotes, no explanation, no punctuation at the end.

Examples of good titles:
- Code Review Discussion
- Bug Fix for Login
- API Design Questions
- Database Migration Help
- React Component Tutorial

Conversation:
%s

Title:`, conversationSummary)

	out, err := c.generate(ctx, c.cfg.Role, prompt)
	if err != nil {
		return "", err
	}

	name := strings.TrimSpace(out)
	name = strings.Trim(name, `"'`)
	name = strings.TrimRight(name, ".!?,;:")

	maxLen := 50
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	if name == "" {
		name = c.cfg.FallbackName
	}
	return name, nil
}

// GenerateText sends a general-purpose text generation request.
// The maxTokens parameter is currently advisory — gateway generate does not
// expose a num_predict flag yet.
func (c *OllamaClient) GenerateText(ctx context.Context, role, prompt string, _maxTokens int) (string, error) {
	out, err := c.generate(ctx, role, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (c *OllamaClient) generate(ctx context.Context, role, prompt string) (string, error) {
	args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
	out, err := c.run(ctx, args, prompt)
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

func (c *OllamaClient) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if c.Runner != nil {
		return c.Runner(ctx, args, stdin)
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

// FallbackName returns the configured fallback name for when generation fails.
func (c *OllamaClient) FallbackName() string {
	return c.cfg.FallbackName
}

// Config returns the naming configuration for inspection/logging.
func (c *OllamaClient) Config() config.NamingConfig {
	return c.cfg
}

// SummaryLimits returns the configured limits for conversation summary building.
// Returns (maxMessages, maxContentLen).
func (c *OllamaClient) SummaryLimits() (int, int) {
	return c.cfg.SummaryMessageLimit, c.cfg.SummaryContentLimit
}

// IsAvailable checks if the Ollama daemon is accessible via the resource CLI.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "resource-ollama", "status")
	return cmd.Run() == nil
}
