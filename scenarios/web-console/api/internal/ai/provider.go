package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

type consumerTokenContextKey struct{}

// ErrCreditsRequired is preserved through the provider chain so transports
// can expose a typed wallet refusal instead of flattening it into a generic
// provider outage.
var ErrCreditsRequired = errors.New("credits required")

func WithConsumerToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, consumerTokenContextKey{}, strings.TrimSpace(token))
}

func consumerToken(ctx context.Context) string {
	token, _ := ctx.Value(consumerTokenContextKey{}).(string)
	return strings.TrimSpace(token)
}

// Provider abstracts one source in the command-generation chain. The active
// production source implementations are AI Gateway roles, which own provider
// consent, privacy eligibility, and resource selection.
type Provider interface {
	Name() string
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// Chain tries each explicitly configured source in order until one succeeds.
type Chain struct {
	providers []Provider
}

func NewChain(providers ...Provider) *Chain { return &Chain{providers: providers} }

func (c *Chain) Providers() []Provider { return c.providers }

func (c *Chain) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, string, error) {
	var lastErr error
	for _, provider := range c.providers {
		result, err := provider.Generate(ctx, systemPrompt, userPrompt)
		if err != nil {
			log.Printf("ai-generate: provider %s failed: %v", provider.Name(), err)
			lastErr = err
			continue
		}
		return result, provider.Name(), nil
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return "", "", fmt.Errorf("no providers configured")
}
