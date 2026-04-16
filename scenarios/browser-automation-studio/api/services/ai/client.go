package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/logutil"
)

// OpenRouterClient provides typed helpers for invoking resource-openrouter.
type OpenRouterClient struct {
	log   *logrus.Logger
	model string
}

const (
	openRouterDefaultModel = "openai/gpt-4o-mini"
	openRouterCommand      = "resource-openrouter"
)

// NewOpenRouterClient initializes an OpenRouter client instance.
func NewOpenRouterClient(log *logrus.Logger) *OpenRouterClient {
	model := os.Getenv("BAS_OPENROUTER_MODEL")
	if strings.TrimSpace(model) == "" {
		model = openRouterDefaultModel
	}

	return &OpenRouterClient{
		log:   log,
		model: model,
	}
}

// ExecutePrompt sends a prompt through resource-openrouter and returns the raw response text.
func (c *OpenRouterClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is required")
	}
	cmd := exec.CommandContext(ctx, openRouterCommand, "generate", "--model", c.model)
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(prompt)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	fields := logrus.Fields{
		"model":    c.model,
		"duration": duration.Milliseconds(),
		"cmd":      strings.Join(cmd.Args, " "),
	}

	if err != nil {
		stderr := strings.TrimSpace(string(output))
		fields["exit_error"] = err.Error()
		if stderr != "" {
			fields["stderr"] = stderr
		}
		c.log.WithFields(fields).Error("OpenRouter prompt execution failed")
		if stderr != "" {
			return "", fmt.Errorf("resource-openrouter execution failed: %s", stderr)
		}
		return "", fmt.Errorf("resource-openrouter execution failed: %w", err)
	}

	response := strings.TrimSpace(string(output))
	if response == "" {
		return "", errors.New("resource-openrouter returned an empty response; verify the selected model produces textual completions")
	}
	fields["response_preview"] = logutil.TruncateForLog(response, 400)
	c.log.WithFields(fields).Debug("OpenRouter prompt executed successfully")
	return response, nil
}

// Model returns the configured AI model name.
func (c *OpenRouterClient) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}
