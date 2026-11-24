package byok

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	"audio-tools/internal/ai/summarizechain"

	"github.com/vrooli/api-core/schedule"
)

const (
	openRouterCommand      = "resource-openrouter"
	openRouterOutputLimit  = 1 << 20
	openRouterMaxOutTokens = 512
)

type openRouterCommandRunner func(ctx context.Context, command string, args, env []string, payload []byte, outputLimit int) ([]byte, error)

// OpenRouterSummarize routes BYOK generation through the resource-owned CLI.
// The request key is sent only in a typed one-shot stdin payload to the
// resource owner; it never appears in arguments, environment, logs, or errors.
type OpenRouterSummarize struct {
	ProviderID string
	Role       string
	Clock      schedule.Clock

	runCommand openRouterCommandRunner
}

func NewOpenRouterSummarize() *OpenRouterSummarize {
	return &OpenRouterSummarize{Role: "summarize.default", Clock: schedule.System()}
}

func (a *OpenRouterSummarize) ID() string {
	if a != nil && a.ProviderID != "" {
		return a.ProviderID
	}
	return "openrouter"
}

// Model reports the policy role used when the request does not select a
// concrete model. The model itself remains owned by resource-openrouter.
func (a *OpenRouterSummarize) Model() string {
	if a == nil || strings.TrimSpace(a.Role) == "" {
		return "summarize.default"
	}
	return strings.TrimSpace(a.Role)
}

func (a *OpenRouterSummarize) IsAvailable(_ context.Context, key string) bool {
	return strings.TrimSpace(key) != ""
}

func summarizationSystemPrompt(level string) string {
	switch level {
	case "light":
		return "Summarize in one sentence. No preamble. Just the summary."
	case "heavy":
		return "Summarize in 1–2 short sentences. Aggressive compression. No preamble."
	default:
		return "Summarize in 2–4 sentences. No preamble. Just the summary."
	}
}

func (a *OpenRouterSummarize) Summarize(ctx context.Context, key string, req summarizechain.Request) (*summarizechain.Result, error) {
	if ctx == nil {
		return nil, errors.New("openrouter: request failed")
	}
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("openrouter: missing API key")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	args := []string{"generate", "--byok-json-stdin", "--json"}
	model := strings.TrimSpace(req.Model)
	role := a.Model()
	if model != "" {
		role = ""
	}
	payload, _ := json.Marshal(struct {
		APIKey    string `json:"api_key"`
		Role      string `json:"role,omitempty"`
		Model     string `json:"model,omitempty"`
		Prompt    string `json:"prompt"`
		MaxTokens int    `json:"max_tokens"`
	}{key, role, model, summarizationSystemPrompt(req.Level) + "\n\n" + req.Text, openRouterMaxOutTokens})
	command := openRouterCommand
	runner := runOpenRouterCommand
	if a != nil && a.runCommand != nil {
		runner = a.runCommand
	}
	start := schedule.System().Now()
	if a != nil && a.Clock != nil {
		start = a.Clock.Now()
	}
	stdout, err := runner(ctx, command, args, openRouterChildEnv(os.Environ()), payload, openRouterOutputLimit)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("openrouter: generation failed")
	}
	if len(stdout) > openRouterOutputLimit {
		return nil, errors.New("openrouter: generation returned an oversized response")
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(stdout, &out); err != nil || len(out.Choices) == 0 {
		return nil, errors.New("openrouter: generation returned an invalid response")
	}
	clk := schedule.System()
	if a != nil && a.Clock != nil {
		clk = a.Clock
	}
	return &summarizechain.Result{
		Text:         out.Choices[0].Message.Content,
		PromptTokens: out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		ModelID:      out.Model,
		Latency:      clk.Now().Sub(start),
	}, nil
}

// openRouterChildEnv forwards only runtime configuration needed by the
// resource CLI. Inherited credentials and unrelated environment values are
// dropped; the one-shot credential travels in the typed stdin request.
func openRouterChildEnv(inherited []string) []string {
	allowed := map[string]bool{
		"PATH": true, "HOME": true, "XDG_DATA_HOME": true, "XDG_CONFIG_HOME": true, "XDG_STATE_HOME": true,
		"RESOURCE_DATA_DIR": true, "RESOURCE_CONFIG_DIR": true, "RESOURCE_STATE_DIR": true,
		"OPENROUTER_DATA_DIR": true, "OPENROUTER_CONFIG_DIR": true, "OPENROUTER_STATE_DIR": true,
		"OPENROUTER_CREDENTIALS_FILE": true, "OPENROUTER_MANUAL_MODELS_FILE": true,
		"OPENROUTER_API_BASE": true, "OPENROUTER_DEFAULT_ROLE": true,
		"OPENROUTER_TIMEOUT": true, "OPENROUTER_HEALTH_CHECK_TIMEOUT": true,
	}
	values := make(map[string]string, len(allowed))
	for _, entry := range inherited {
		name, value, ok := strings.Cut(entry, "=")
		if ok && allowed[name] {
			values[name] = value
		}
	}
	env := make([]string, 0, len(values))
	for name, value := range values {
		env = append(env, name+"="+value)
	}
	return env
}

type outputLimitWriter struct {
	buffer bytes.Buffer
	limit  int
}

func (w *outputLimitWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.buffer.Len()
	if remaining <= 0 {
		return 0, errors.New("output limit exceeded")
	}
	if len(p) > remaining {
		_, _ = w.buffer.Write(p[:remaining])
		return remaining, errors.New("output limit exceeded")
	}
	return w.buffer.Write(p)
}

func runOpenRouterCommand(ctx context.Context, command string, args, env []string, payload []byte, outputLimit int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(payload)
	stdout := &outputLimitWriter{limit: outputLimit}
	cmd.Stdout = stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if strings.Contains(err.Error(), "output limit exceeded") {
			return nil, errors.New("resource-openrouter output exceeds limit")
		}
		return nil, errors.New("resource-openrouter command failed")
	}
	return stdout.buffer.Bytes(), nil
}
