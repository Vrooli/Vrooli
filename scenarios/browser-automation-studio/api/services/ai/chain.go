package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/credits"
)

// AIProviderChain implements the AI provider fallback chain:
// BYOK → Vrooli API → Dev mode → Block
//
// The chain tries providers in order until one succeeds or all fail.
// Credits are only charged when using the Vrooli API provider.
type AIProviderChain struct {
	log           *logrus.Logger
	creditService credits.CreditService

	// Configuration
	enableBYOK    bool
	enableVrooli  bool
	enableDevMode bool
	vrooliAPIURL  string

	// defaultModel is an OPTIONAL explicit operator override (BAS_AI_DEFAULT_MODEL).
	// When empty, the effective model is resolved through the OpenRouter resource
	// policy using role. There is no baked-in default slug.
	defaultModel string

	// role is the OpenRouter policy role used to resolve the default model when no
	// explicit override is supplied.
	role string

	// Pre-initialized dev provider (always available if resource-openrouter exists)
	devProvider *DevProvider
}

// AIProviderChainOptions configures the provider chain.
type AIProviderChainOptions struct {
	Logger        *logrus.Logger
	CreditService credits.CreditService

	// Enable/disable providers
	EnableBYOK    bool
	EnableVrooli  bool
	EnableDevMode bool

	// Provider configuration
	VrooliAPIURL string

	// DefaultModel is an OPTIONAL explicit operator override. Leave empty to resolve
	// the model from the OpenRouter resource policy via Role.
	DefaultModel string

	// Role is the OpenRouter policy role used to resolve the default model. When
	// empty it falls back to the package default role (chat.default).
	Role string
}

// NewAIProviderChain creates a new provider chain.
func NewAIProviderChain(opts AIProviderChainOptions) *AIProviderChain {
	chain := &AIProviderChain{
		log:           opts.Logger,
		creditService: opts.CreditService,
		enableBYOK:    opts.EnableBYOK,
		enableVrooli:  opts.EnableVrooli,
		enableDevMode: opts.EnableDevMode,
		vrooliAPIURL:  opts.VrooliAPIURL,
		defaultModel:  opts.DefaultModel,
		role:          opts.Role,
	}

	// Pre-initialize dev provider since it doesn't require per-request config
	if opts.EnableDevMode {
		chain.devProvider = NewDevProvider(DevProviderOptions{
			Logger: opts.Logger,
			Model:  opts.DefaultModel,
		})
	}

	return chain
}

// Execute runs a prompt through the provider chain.
// Returns the result including which provider was used.
func (c *AIProviderChain) Execute(ctx context.Context, req ProviderRequest) (*ProviderResult, error) {
	model, err := c.resolveModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	var lastErr error

	// Try BYOK first
	if c.enableBYOK && req.BYOKKey != "" {
		provider := NewBYOKProvider(BYOKProviderOptions{
			Logger: c.log,
			APIKey: req.BYOKKey,
			Model:  model,
		})

		if provider.IsAvailable(ctx) {
			c.log.WithFields(logrus.Fields{
				"provider": "byok",
				"model":    model,
			}).Debug("Trying BYOK provider")

			response, err := provider.ExecutePrompt(ctx, req.Prompt)
			if err == nil {
				return &ProviderResult{
					Response:       response,
					Provider:       ProviderTypeBYOK,
					Model:          model,
					ChargedCredits: false, // BYOK doesn't charge credits
				}, nil
			}

			lastErr = err
			c.log.WithError(err).Debug("BYOK provider failed, trying next")
		}
	}

	// Try Vrooli API (charges credits through LPBS gateway)
	// NOTE: Credit checking and charging is handled atomically by LPBS.
	// BAS no longer needs to pre-check or charge credits for Vrooli provider requests.
	// LPBS returns ErrInsufficientCredits if the user doesn't have enough credits.
	if c.enableVrooli && req.LPBSAuthToken != "" {
		provider := NewVrooliProvider(VrooliProviderOptions{
			Logger:    c.log,
			APIURL:    c.vrooliAPIURL,
			Model:     model,
			AuthToken: req.LPBSAuthToken,
		})

		if provider.IsAvailable(ctx) {
			c.log.WithFields(logrus.Fields{
				"provider": "vrooli",
				"model":    model,
			}).Debug("Trying Vrooli API provider (credits handled by LPBS)")

			response, err := provider.ExecutePrompt(ctx, req.Prompt)
			if err == nil {
				return &ProviderResult{
					Response:       response,
					Provider:       ProviderTypeVrooli,
					Model:          model,
					ChargedCredits: true, // LPBS charges credits atomically
				}, nil
			}

			// Check if it's an insufficient credits error
			if err == ErrInsufficientCredits {
				c.log.WithFields(logrus.Fields{
					"user": req.UserIdentity,
				}).Debug("Insufficient credits for Vrooli provider")
				// Don't try fallback providers for credit errors - user needs to upgrade
				return nil, err
			}

			lastErr = err
			c.log.WithError(err).Debug("Vrooli provider failed, trying next")
		}
	}

	// Try Dev mode (local resource-openrouter)
	if c.enableDevMode && c.devProvider != nil {
		if c.devProvider.IsAvailable(ctx) {
			c.log.WithFields(logrus.Fields{
				"provider": "dev",
				"model":    model,
			}).Debug("Trying dev mode provider")

			response, err := c.devProvider.ExecutePromptWithModel(ctx, model, req.Prompt)
			if err == nil {
				return &ProviderResult{
					Response:       response,
					Provider:       ProviderTypeDev,
					Model:          model,
					ChargedCredits: false, // Dev mode doesn't charge credits
				}, nil
			}

			lastErr = err
			c.log.WithError(err).Debug("Dev mode provider failed")
		}
	}

	// All providers failed
	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrAllProvidersUnavailable, lastErr)
	}
	return nil, ErrAllProvidersUnavailable
}

// resolveModel determines the effective OpenRouter model for a request.
//
// Resolution order (no concrete slug is ever baked into source):
//  1. An explicit per-request model override (honored verbatim).
//  2. An explicit operator override (BAS_AI_DEFAULT_MODEL, via c.defaultModel).
//  3. Role-based resolution through the OpenRouter resource policy.
//
// If steps 1-2 are empty and policy resolution fails, this returns an error
// rather than falling back to a hard-coded model slug.
func (c *AIProviderChain) resolveModel(ctx context.Context, requestedModel string) (string, error) {
	if model := strings.TrimSpace(requestedModel); model != "" {
		return model, nil
	}
	if model := strings.TrimSpace(c.defaultModel); model != "" {
		return model, nil
	}

	role := strings.TrimSpace(c.role)
	if role == "" {
		role = openRouterRole()
	}

	model, err := resolveRoleModel(ctx, role)
	if err != nil {
		return "", fmt.Errorf("no explicit model override supplied and OpenRouter policy resolution failed: %w", err)
	}
	return model, nil
}

// GetAvailableProviders returns which providers are currently available.
// Useful for debugging and status endpoints.
func (c *AIProviderChain) GetAvailableProviders(ctx context.Context) []ProviderType {
	var available []ProviderType

	// Note: BYOK availability depends on per-request key, so we skip it here

	if c.enableVrooli {
		provider := NewVrooliProvider(VrooliProviderOptions{
			Logger: c.log,
			APIURL: c.vrooliAPIURL,
		})
		if provider.IsAvailable(ctx) {
			available = append(available, ProviderTypeVrooli)
		}
	}

	if c.enableDevMode && c.devProvider != nil && c.devProvider.IsAvailable(ctx) {
		available = append(available, ProviderTypeDev)
	}

	return available
}

// IsAnyProviderAvailable checks if at least one provider can handle requests.
// Note: This doesn't check BYOK since that requires a per-request key.
func (c *AIProviderChain) IsAnyProviderAvailable(ctx context.Context) bool {
	return len(c.GetAvailableProviders(ctx)) > 0
}
