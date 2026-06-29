package generation

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// TextRequest describes a text generation request to a provider.
type TextRequest struct {
	Prompt string
	Model  string // optional role/model override
}

// TextResponse holds the result of a text generation call.
type TextResponse struct {
	Text     string
	Provider string
	Model    string
}

// Provider is the interface every TEXT AI backend implements. The chain tries
// providers in order until one succeeds. Image generation is NOT a provider
// concern — images run through image-tools (the ImageBackend seam), never this
// chain.
type Provider interface {
	// Name returns the provider identifier (e.g. "ollama", "openrouter").
	Name() string
	// Available reports whether the provider is currently reachable/configured.
	Available(ctx context.Context) bool
	// GenerateText sends a prompt and returns the text response.
	GenerateText(ctx context.Context, req TextRequest) (TextResponse, error)
}

// Providers is the seam Service depends on for TEXT facet generation. The
// production Chain satisfies it; service unit tests substitute a fake. Keeping
// the service behind this interface (rather than *Chain) means tests never reach
// out to Ollama or OpenRouter.
type Providers interface {
	// Available reports whether at least one provider in the chain is reachable.
	Available(ctx context.Context) bool
	// Statuses reports each provider's name and current availability, in chain
	// order.
	Statuses(ctx context.Context) []ProviderStatus
	GenerateText(ctx context.Context, req TextRequest) (TextResponse, error)
}

// ProviderStatus is one provider's reported reachability.
type ProviderStatus struct {
	Name      string
	Available bool
}

// Chain tries providers in order until one succeeds. Ported from aigen.Chain.
type Chain struct {
	providers []Provider
}

// NewChain creates a provider chain with the given providers (tried in order).
func NewChain(providers ...Provider) *Chain {
	return &Chain{providers: providers}
}

// Compile-time guarantee the chain satisfies the service seam.
var _ Providers = (*Chain)(nil)

// NewChainFromEnv builds the production TEXT provider chain from the
// environment:
//
//   - Ollama is always added (local, free, routed through the resource-ollama
//     gateway CLI). Its role defaults to "chat.default" unless OLLAMA_ROLE is
//     set.
//   - OpenRouter is added only when OPENROUTER_API_KEY is set. Its text model
//     defaults unless OPENROUTER_TEXT_MODEL is set.
//
// The chain is never empty (Ollama is always present); whether it is *available*
// depends on the resource-ollama daemon being reachable at call time.
func NewChainFromEnv() *Chain {
	providers := []Provider{NewOllamaProvider(strings.TrimSpace(os.Getenv("OLLAMA_ROLE")))}
	if key := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); key != "" {
		providers = append(providers, NewOpenRouterProvider(
			key,
			strings.TrimSpace(os.Getenv("OPENROUTER_TEXT_MODEL")),
		))
	}
	return NewChain(providers...)
}

// Available returns true if at least one provider is available.
func (c *Chain) Available(ctx context.Context) bool {
	for _, p := range c.providers {
		if p.Available(ctx) {
			return true
		}
	}
	return false
}

// Statuses reports each provider's availability in chain order.
func (c *Chain) Statuses(ctx context.Context) []ProviderStatus {
	out := make([]ProviderStatus, 0, len(c.providers))
	for _, p := range c.providers {
		out = append(out, ProviderStatus{Name: p.Name(), Available: p.Available(ctx)})
	}
	return out
}

// GenerateText tries each available provider in order for text generation.
func (c *Chain) GenerateText(ctx context.Context, req TextRequest) (TextResponse, error) {
	var errs []string
	for _, p := range c.providers {
		if !p.Available(ctx) {
			errs = append(errs, fmt.Sprintf("%s: unavailable", p.Name()))
			continue
		}
		resp, err := p.GenerateText(ctx, req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.Name(), err))
			continue
		}
		return resp, nil
	}
	return TextResponse{}, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
}
