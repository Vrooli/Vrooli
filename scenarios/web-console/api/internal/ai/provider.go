package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// Provider abstracts a single AI provider for text generation.
// Providers are use-case-agnostic: the caller supplies both the system
// prompt (which sets the task context) and the user prompt.
type Provider interface {
	Name() string
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// Chain tries each provider in order until one succeeds.
type Chain struct {
	providers []Provider
}

// NewChain creates a chain with the given providers.
func NewChain(providers ...Provider) *Chain { return &Chain{providers: providers} }

// Providers returns the chain's providers in order. Used by Service to
// match enabled configs to live provider instances.
func (c *Chain) Providers() []Provider { return c.providers }

// Generate tries each provider in order, returning the first successful result.
func (c *Chain) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, string, error) {
	var lastErr error
	for _, p := range c.providers {
		res, err := p.Generate(ctx, systemPrompt, userPrompt)
		if err != nil {
			log.Printf("ai-generate: provider %s failed: %v", p.Name(), err)
			lastErr = err
			continue
		}
		return res, p.Name(), nil
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return "", "", fmt.Errorf("no providers configured")
}

// OllamaProvider calls the shared Ollama gateway using a model role.
type OllamaProvider struct {
	Role   string
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

// NewOllamaProvider creates an Ollama provider with env-configurable settings.
func NewOllamaProvider() *OllamaProvider {
	role := os.Getenv("WC_OLLAMA_ROLE")
	if role == "" {
		role = "chat.default"
	}
	return &OllamaProvider{
		Role: role,
	}
}

func (o *OllamaProvider) Name() string { return "ollama" }

func (o *OllamaProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	role := strings.TrimSpace(o.Role)
	if role == "" {
		role = "chat.default"
	}
	prompt := strings.TrimSpace(fmt.Sprintf("System:\n%s\n\nUser:\n%s", systemPrompt, userPrompt))
	args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
	out, err := o.run(ctx, args, prompt)
	if err != nil {
		return "", fmt.Errorf("resource-ollama gateway generate failed: %w", err)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &result); err != nil {
		return "", fmt.Errorf("decode gateway response: %w", err)
	}
	return result.Response, nil
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

// OpenRouterProvider calls the OpenRouter API as a fallback.
//
// Greenfield: this provider selects NO concrete model. It resolves an OpenRouter
// policy ROLE (default chat.default) to a concrete slug through
// `resource-openrouter policy resolve`; resource-openrouter is the single source
// of truth.
type OpenRouterProvider struct {
	APIKey string
	Role   string
	Client *http.Client

	// Runner is an optional seam for tests; production callers leave it nil and
	// the real resource-openrouter binary is exec'd to resolve the role.
	Runner func(ctx context.Context, args []string) ([]byte, error)
}

// NewOpenRouterProvider creates an OpenRouter provider with env-configurable settings.
func NewOpenRouterProvider() *OpenRouterProvider {
	role := os.Getenv("WC_OPENROUTER_ROLE")
	if role == "" {
		role = "chat.default"
	}
	return &OpenRouterProvider{
		APIKey: os.Getenv("OPENROUTER_API_KEY"),
		Role:   role,
		Client: &http.Client{Timeout: DefaultProviderTimeout},
	}
}

func (o *OpenRouterProvider) Name() string { return "openrouter" }

func (o *OpenRouterProvider) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if o.APIKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	model, err := o.resolveModel(ctx, o.Role)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkProviderResponse(resp, "openrouter"); err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return result.Choices[0].Message.Content, nil
}

// resolveModel turns a policy role into a concrete OpenRouter model slug via
// `resource-openrouter policy resolve`. The resource is the authority; this
// provider never reads the policy file or hard-codes a model slug.
func (o *OpenRouterProvider) resolveModel(ctx context.Context, role string) (string, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		role = "chat.default"
	}
	args := []string{"policy", "resolve", "--role", role, "--field", "model"}
	out, err := o.run(ctx, args)
	if err != nil {
		return "", fmt.Errorf("resource-openrouter policy resolve %q: %w", role, err)
	}
	model := strings.TrimSpace(string(out))
	if model == "" {
		return "", fmt.Errorf("resource-openrouter policy resolve %q returned no model", role)
	}
	return model, nil
}

func (o *OpenRouterProvider) run(ctx context.Context, args []string) ([]byte, error) {
	if o.Runner != nil {
		return o.Runner(ctx, args)
	}
	cmd := exec.CommandContext(ctx, "resource-openrouter", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func checkProviderResponse(resp *http.Response, providerName string) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s returned %d: %s", providerName, resp.StatusCode, string(body))
	}
	return nil
}
