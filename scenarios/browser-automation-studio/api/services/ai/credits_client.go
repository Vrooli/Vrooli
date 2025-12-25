package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// ErrNoAICredits indicates the user has no AI credits remaining.
var ErrNoAICredits = errors.New("no AI credits remaining")

// ErrNoAIAccess indicates the user's tier doesn't include AI access.
var ErrNoAIAccess = errors.New("your subscription tier does not include AI access")

// CreditsClient wraps an AIClient to enforce credit limits and track usage.
type CreditsClient struct {
	inner            AIClient
	entitlementSvc   *entitlement.Service
	aiCreditsTracker *entitlement.AICreditsTracker
	log              *logrus.Logger

	// userIdentityFn retrieves the current user identity for credit checks.
	// This is typically set by request context.
	userIdentityFn func(ctx context.Context) string
}

// CreditsClientOptions configures the credits client.
type CreditsClientOptions struct {
	Inner            AIClient
	EntitlementSvc   *entitlement.Service
	AICreditsTracker *entitlement.AICreditsTracker
	Logger           *logrus.Logger
	UserIdentityFn   func(ctx context.Context) string
}

// NewCreditsClient creates a new credits-aware AI client wrapper.
func NewCreditsClient(opts CreditsClientOptions) *CreditsClient {
	return &CreditsClient{
		inner:            opts.Inner,
		entitlementSvc:   opts.EntitlementSvc,
		aiCreditsTracker: opts.AICreditsTracker,
		log:              opts.Logger,
		userIdentityFn:   opts.UserIdentityFn,
	}
}

// ExecutePrompt executes an AI prompt, checking and charging credits.
func (c *CreditsClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	return c.ExecutePromptWithType(ctx, prompt, "workflow_generate")
}

// ExecutePromptWithType executes an AI prompt with a specific request type for logging.
func (c *CreditsClient) ExecutePromptWithType(ctx context.Context, prompt, requestType string) (string, error) {
	userIdentity := ""
	if c.userIdentityFn != nil {
		userIdentity = c.userIdentityFn(ctx)
	}

	// Skip credit checks if entitlements are disabled
	if c.entitlementSvc != nil && !c.entitlementSvc.IsEnabled() {
		return c.executeAndLog(ctx, prompt, userIdentity, requestType)
	}

	// Check if user has AI credits available
	if c.entitlementSvc != nil && c.aiCreditsTracker != nil && userIdentity != "" {
		ent, err := c.entitlementSvc.GetEntitlement(ctx, userIdentity)
		if err != nil {
			c.log.WithError(err).Warn("Failed to get entitlement for AI credits check, allowing request")
			// Fail open - allow the request
			return c.executeAndLog(ctx, prompt, userIdentity, requestType)
		}

		creditsLimit := c.entitlementSvc.GetAICreditsLimit(ent.Tier)

		// Check if tier has AI access
		if creditsLimit == 0 {
			return "", ErrNoAIAccess
		}

		// Check if unlimited
		if creditsLimit < 0 {
			// Unlimited - proceed without checking remaining credits
			return c.executeAndLog(ctx, prompt, userIdentity, requestType)
		}

		// Check remaining credits
		canUse, remaining, err := c.aiCreditsTracker.CanUseCredits(ctx, userIdentity, creditsLimit)
		if err != nil {
			c.log.WithError(err).Warn("Failed to check AI credits, allowing request")
			// Fail open - allow the request
			return c.executeAndLog(ctx, prompt, userIdentity, requestType)
		}

		if !canUse {
			return "", fmt.Errorf("%w: you have %d of %d credits remaining", ErrNoAICredits, remaining, creditsLimit)
		}
	}

	return c.executeAndLog(ctx, prompt, userIdentity, requestType)
}

// executeAndLog executes the prompt and logs/charges the usage.
func (c *CreditsClient) executeAndLog(ctx context.Context, prompt, userIdentity, requestType string) (string, error) {
	start := time.Now()
	response, err := c.inner.ExecutePrompt(ctx, prompt)
	duration := time.Since(start)

	// Charge credits on successful execution
	if err == nil && c.aiCreditsTracker != nil && userIdentity != "" {
		creditsToCharge := 1 // Each AI request costs 1 credit

		if chargeErr := c.aiCreditsTracker.ChargeCredits(ctx, userIdentity, creditsToCharge, requestType, c.inner.Model()); chargeErr != nil {
			c.log.WithError(chargeErr).WithFields(logrus.Fields{
				"user":    userIdentity,
				"credits": creditsToCharge,
			}).Warn("Failed to charge AI credits")
			// Don't fail the request - just log the error
		}
	} else if err != nil && c.aiCreditsTracker != nil && userIdentity != "" {
		// Log failed request (don't charge credits)
		logErr := c.aiCreditsTracker.LogRequest(ctx, &entitlement.AIRequestLog{
			UserIdentity:   userIdentity,
			RequestType:    requestType,
			Model:          c.inner.Model(),
			CreditsCharged: 0,
			Success:        false,
			ErrorMessage:   err.Error(),
			DurationMs:     int(duration.Milliseconds()),
		})
		if logErr != nil {
			c.log.WithError(logErr).Warn("Failed to log failed AI request")
		}
	}

	return response, err
}

// Model returns the configured AI model name.
func (c *CreditsClient) Model() string {
	return c.inner.Model()
}

// Ensure CreditsClient implements AIClient.
var _ AIClient = (*CreditsClient)(nil)
