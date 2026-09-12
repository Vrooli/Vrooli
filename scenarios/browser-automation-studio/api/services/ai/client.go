package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/logutil"
)

// OpenRouterClient provides typed helpers for invoking resource-openrouter.
type OpenRouterClient struct {
	log   *logrus.Logger
	model string
}

const openRouterCommand = "resource-openrouter"

// NewOpenRouterClient initializes an OpenRouter client instance.
//
// No concrete model slug is baked in: the client's model is left unset and the
// effective model is resolved per call via the OpenRouter resource policy (role
// based) unless an explicit model override is configured on the client.
func NewOpenRouterClient(log *logrus.Logger) *OpenRouterClient {
	return &OpenRouterClient{
		log: log,
	}
}

// ExecutePrompt sends a prompt through resource-openrouter and returns the raw response text.
//
// The model is resolved in order: explicit client model override → role-based
// resolution via the OpenRouter resource policy. If neither yields a model the
// call fails — there is no hard-coded fallback slug.
func (c *OpenRouterClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is required")
	}

	model := strings.TrimSpace(c.model)
	if model == "" {
		resolved, err := resolveRoleModel(ctx, openRouterRole())
		if err != nil {
			return "", err
		}
		model = resolved
	}
	return c.executePromptWithModel(ctx, model, prompt)
}

// ExecutePromptWithModel sends a prompt through resource-openrouter using an
// explicit, already-resolved model slug. The caller is responsible for supplying
// a non-empty model (typically resolved through the OpenRouter resource policy).
func (c *OpenRouterClient) ExecutePromptWithModel(ctx context.Context, model, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is required")
	}
	if strings.TrimSpace(model) == "" {
		return "", errors.New("model is required")
	}
	return c.executePromptWithModel(ctx, strings.TrimSpace(model), prompt)
}

func (c *OpenRouterClient) executePromptWithModel(ctx context.Context, model, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, openRouterCommand, "generate", "--model", model)
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	cmd.Stdin = strings.NewReader(prompt)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	fields := logrus.Fields{
		"model":    model,
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
