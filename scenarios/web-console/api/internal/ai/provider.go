package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

type consumerTokenContextKey struct{}
type rootOperationIDContextKey struct{}
type privacyConsentGrantContextKey struct{}
type privacyConsentSourceContextKey struct{}

// ErrCreditsRequired is preserved through the provider chain so transports
// can expose a typed wallet refusal instead of flattening it into a generic
// provider outage.
var ErrCreditsRequired = errors.New("credits required")

func WithConsumerToken(ctx context.Context, token string) context.Context {
	token = strings.TrimSpace(token)
	if scheme, value, ok := strings.Cut(token, " "); ok && strings.EqualFold(scheme, "Bearer") {
		token = strings.TrimSpace(value)
	}
	return context.WithValue(ctx, consumerTokenContextKey{}, token)
}

func consumerToken(ctx context.Context) string {
	token, _ := ctx.Value(consumerTokenContextKey{}).(string)
	return strings.TrimSpace(token)
}

// WithPrivacyConsentGrant carries the one-time consent receipt for the exact
// selected remote source. The shared Gateway verifies it on every dispatch.
func WithPrivacyConsentGrant(ctx context.Context, receipt, source string) context.Context {
	ctx = context.WithValue(ctx, privacyConsentGrantContextKey{}, strings.TrimSpace(receipt))
	return context.WithValue(ctx, privacyConsentSourceContextKey{}, strings.ToLower(strings.TrimSpace(source)))
}

func privacyConsentGrant(ctx context.Context, selectedSource string) string {
	receipt, _ := ctx.Value(privacyConsentGrantContextKey{}).(string)
	source, _ := ctx.Value(privacyConsentSourceContextKey{}).(string)
	if source != strings.ToLower(strings.TrimSpace(selectedSource)) {
		return ""
	}
	return strings.TrimSpace(receipt)
}

// WithRootOperationID carries one high-entropy AI operation identity through
// source fallback and into AI Gateway's accounting ledger. A caller may reuse
// the same ID when it retries an uncertain request.
func WithRootOperationID(ctx context.Context, id string) context.Context {
	id = strings.TrimSpace(id)
	if id == "" {
		id = uuid.NewString()
	}
	return context.WithValue(ctx, rootOperationIDContextKey{}, id)
}

func rootOperationID(ctx context.Context) string {
	id, _ := ctx.Value(rootOperationIDContextKey{}).(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return uuid.NewString()
	}
	return id
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
