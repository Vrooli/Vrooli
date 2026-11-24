package generation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/ai-provider-go/openai"
)

// OpenRouterProvider connects to the OpenRouter API for TEXT facet generation
// (the cloud fallback after Ollama). Image generation is NOT an OpenRouter
// concern in brand-manager — images run through image-tools.
//
// Greenfield: this provider selects NO concrete model. It resolves an OpenRouter
// policy ROLE (default chat.default) to a concrete slug through
// `resource-openrouter policy resolve`; resource-openrouter is the single source
// of truth. A per-request req.Model is honored only as an explicit advanced
// operator override.
type OpenRouterProvider struct {
	APIKey     string
	BaseURL    string // defaults to "https://openrouter.ai/api/v1"
	Role       string
	HTTPClient *http.Client

	// Runner is an optional seam for tests; production callers leave it nil and
	// the real resource-openrouter binary is exec'd to resolve the role.
	Runner func(ctx context.Context, args []string) ([]byte, error)
}

// NewOpenRouterProvider creates a provider for the given API key and role. An
// empty role defaults to "chat.default".
func NewOpenRouterProvider(apiKey, role string) *OpenRouterProvider {
	if role == "" {
		role = "chat.default"
	}
	return &OpenRouterProvider{
		APIKey:     apiKey,
		BaseURL:    "https://openrouter.ai/api/v1",
		Role:       role,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns "openrouter".
func (o *OpenRouterProvider) Name() string { return "openrouter" }

// Available checks if an API key is configured.
func (o *OpenRouterProvider) Available(_ context.Context) bool {
	return o.APIKey != ""
}

// GenerateText calls OpenRouter's chat completion endpoint, resolving the role
// to a concrete model first.
func (o *OpenRouterProvider) GenerateText(ctx context.Context, req TextRequest) (TextResponse, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		resolved, err := o.resolveModel(ctx, o.Role)
		if err != nil {
			return TextResponse{}, err
		}
		model = resolved
	}

	client := openai.New(openai.Config{
		ProviderName: "openrouter",
		BaseURL:      o.BaseURL,
		APIKey:       o.APIKey,
		HTTPClient:   o.HTTPClient,
	})
	result, err := client.Chat(ctx, openai.ChatRequest{
		Model:    model,
		Messages: []openai.Message{{Role: "user", Content: req.Prompt}},
	})
	if err != nil {
		return TextResponse{}, fmt.Errorf("openrouter chat: %w", err)
	}

	return TextResponse{
		Text:     result.Text,
		Provider: "openrouter",
		Model:    result.Model,
	}, nil
}

// resolveModel turns a policy role into a concrete OpenRouter model slug via
// `resource-openrouter policy resolve`. The resource is the authority; this
// provider never reads the policy file directly.
func (o *OpenRouterProvider) resolveModel(ctx context.Context, role string) (string, error) {
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
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

var _ Provider = (*OpenRouterProvider)(nil)
