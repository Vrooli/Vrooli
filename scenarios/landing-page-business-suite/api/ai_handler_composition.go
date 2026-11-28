package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/lpbsidentity"
	aihandler "landing-page-business-suite-api/handlers/intelligence"
	"landing-page-business-suite-api/internal/businessaccount"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/intelligence"
	"landing-page-business-suite-api/internal/logx"
)

// newMeteredInferenceHandler is the API composition point for the AI HTTP boundary.
// Rate-limit lifecycle, authentication context, response envelopes, and logging
// are application concerns; handlers/intelligence only translates the request.
func newMeteredInferenceDependencies(service intelligence.MeteredInferenceProvider, usage *commerce.UsageService, account *commerce.Service, accounts businessaccount.Repository, callerVerifier ...authn.TokenVerifier) aihandler.Dependencies {
	userLimiter := NewRateLimiter(60, time.Minute)
	userLimiter.StartCleanup(5*time.Minute, 10*time.Minute)
	ipLimiter := NewRateLimiter(120, time.Minute)
	ipLimiter.StartCleanup(5*time.Minute, 10*time.Minute)

	deps := aihandler.Dependencies{
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
		UserID:          getUserID,
		UserIdentity:    getUserEmail,
		ResolveBusinessAccount: func(ctx context.Context, userID, userEmail, requestedID string) (string, error) {
			resolved, err := accounts.ResolveForUser(ctx, userID, userEmail, requestedID)
			if err != nil {
				return "", err
			}
			return resolved.ID, nil
		},
		WriteJSONError: func(w http.ResponseWriter, status int, message, errorType string) {
			writeJSONError(w, status, message, errorType)
		},
		Log:      logx.Info,
		LogError: logx.Error,
	}
	if len(callerVerifier) > 0 && callerVerifier[0] != nil {
		verifier := callerVerifier[0]
		deps.ResolveCreditCallerScenario = func(ctx context.Context, rawAuthorization string) (string, error) {
			rawAuthorization = strings.TrimSpace(rawAuthorization)
			token, found := strings.CutPrefix(rawAuthorization, "Bearer ")
			if !found || strings.TrimSpace(token) == "" || strings.ContainsAny(token, " \t\r\n") {
				return "", errors.New("caller service bearer is required")
			}
			principal, err := verifier.Verify(ctx, strings.TrimSpace(token))
			if err != nil {
				return "", err
			}
			return lpbsidentity.GatewayCreditInference.Source(principal)
		}
	}
	return deps
}
