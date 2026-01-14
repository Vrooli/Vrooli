package ai

import (
	"context"
	"fmt"

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
	defaultModel  string

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
	DefaultModel string
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
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	if model == "" {
		model = byokDefaultModel
	}

	// Default operation type if not specified
	opType := req.OperationType
	if opType == "" {
		opType = credits.OpAIWorkflowGenerate
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

	// Try Vrooli API (charges credits)
	if c.enableVrooli {
		provider := NewVrooliProvider(VrooliProviderOptions{
			Logger: c.log,
			APIURL: c.vrooliAPIURL,
			Model:  model,
		})

		if provider.IsAvailable(ctx) {
			// Check credits before attempting
			if c.creditService != nil && c.creditService.IsEnabled() {
				canCharge, remaining, err := c.creditService.CanCharge(ctx, req.UserIdentity, opType)
				if err != nil {
					c.log.WithError(err).Warn("Failed to check credits, skipping Vrooli provider")
				} else if !canCharge {
					c.log.WithFields(logrus.Fields{
						"user":      req.UserIdentity,
						"remaining": remaining,
					}).Debug("Insufficient credits for Vrooli provider")
				} else {
					c.log.WithFields(logrus.Fields{
						"provider": "vrooli",
						"model":    model,
					}).Debug("Trying Vrooli API provider")

					response, err := provider.ExecutePrompt(ctx, req.Prompt)
					if err == nil {
						// Charge credits on success
						_, chargeErr := c.creditService.Charge(ctx, credits.ChargeRequest{
							UserIdentity: req.UserIdentity,
							Operation:    opType,
							Metadata: credits.ChargeMetadata{
								Model: model,
							},
						})
						if chargeErr != nil {
							c.log.WithError(chargeErr).Warn("Failed to charge credits for Vrooli API request")
						}

						return &ProviderResult{
							Response:       response,
							Provider:       ProviderTypeVrooli,
							Model:          model,
							ChargedCredits: chargeErr == nil,
						}, nil
					}

					lastErr = err
					c.log.WithError(err).Debug("Vrooli provider failed, trying next")
				}
			}
		}
	}

	// Try Dev mode (local resource-openrouter)
	if c.enableDevMode && c.devProvider != nil {
		if c.devProvider.IsAvailable(ctx) {
			c.log.WithFields(logrus.Fields{
				"provider": "dev",
				"model":    c.devProvider.Model(),
			}).Debug("Trying dev mode provider")

			response, err := c.devProvider.ExecutePrompt(ctx, req.Prompt)
			if err == nil {
				return &ProviderResult{
					Response:       response,
					Provider:       ProviderTypeDev,
					Model:          c.devProvider.Model(),
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
