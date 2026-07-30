package main

import (
	"context"
	"net/http"
	"time"

	aihandler "landing-page-business-suite-api/handlers/intelligence"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/intelligence"
)

// newAIGatewayHandler is the API composition point for the AI HTTP boundary.
// Rate-limit lifecycle, authentication context, response envelopes, and logging
// are application concerns; handlers/intelligence only translates the request.
func newAIGatewayDependencies(service intelligence.Gateway, usage *commerce.UsageService, account *commerce.Service) aihandler.Dependencies {
	userLimiter := NewRateLimiter(60, time.Minute)
	userLimiter.StartCleanup(5*time.Minute, 10*time.Minute)
	ipLimiter := NewRateLimiter(120, time.Minute)
	ipLimiter.StartCleanup(5*time.Minute, 10*time.Minute)

	return aihandler.Dependencies{
		Service: service,
		Usage: func(ctx context.Context, identity, tier string) (aihandler.UsageSummary, error) {
			summary, err := usage.GetUsageSummary(ctx, identity, tier)
			if err != nil {
				return aihandler.UsageSummary{}, err
			}
			return aihandler.UsageSummary{BillingPeriod: summary.BillingPeriod, ResetDate: summary.ResetDate, Usage: summary.Usage, Limits: summary.Limits, Remaining: summary.Remaining}, nil
		},
		SubscriptionTier: func(ctx context.Context, identity string) (string, error) {
			subscription, err := account.GetSubscriptionContext(ctx, identity)
			if err != nil || subscription == nil || subscription.PlanTier == nil {
				return "", err
			}
			return *subscription.PlanTier, nil
		},
		UserRateLimiter: userLimiter,
		IPRateLimiter:   ipLimiter,
		IPKeyFunc:       IPKeyFunc(),
		UserIdentity:    getUserEmail,
		WriteJSONError: func(w http.ResponseWriter, status int, message, errorType string) {
			writeJSONError(w, status, message, errorType)
		},
		Log:      logStructured,
		LogError: logStructuredError,
	}
}
