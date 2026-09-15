package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	deploymenthttp "landing-page-business-suite-api/handlers/deployment"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
)

const stripeWebhookPath = "/api/v1/webhooks/stripe"

// stripeReadiness is the application-side half of the deployment contract.
// It reports configuration and provider-route gaps without ever returning a
// credential value. The public probe deliberately uses GET: the webhook
// endpoint should reject GET with 405, which proves that the edge reached the
// application without requiring a signed or mutating Stripe event.
func (s *Server) stripeReadiness(ctx context.Context) deploymenthttp.Gate {
	gate := deploymenthttp.Gate{Name: "stripe_commerce"}
	mode := strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE")))
	if mode != "test" && mode != "live" {
		gate.Message = "STRIPE_MODE must be explicitly set to test or live"
		return gate
	}

	settings, err := s.paymentSettings.GetStripeSettings(ctx)
	if err != nil {
		gate.Message = fmt.Sprintf("read active Stripe credentials: %v", err)
		return gate
	}
	missing := make([]string, 0, 3)
	if settings == nil || strings.TrimSpace(settings.GetPublishableKey()) == "" {
		missing = append(missing, "stripe-"+mode+"-publishable-key")
	}
	if settings == nil || strings.TrimSpace(settings.GetSecretKey()) == "" {
		missing = append(missing, "stripe-"+mode+"-secret-key")
	}
	if settings == nil || strings.TrimSpace(settings.GetWebhookSecret()) == "" {
		missing = append(missing, "stripe-"+mode+"-webhook-secret")
	}
	if len(missing) > 0 {
		gate.Message = "missing active Stripe credential fields: " + strings.Join(missing, ", ")
		return gate
	}
	if err := commerce.ValidateStripeKeyModePair(settings.GetPublishableKey(), settings.GetSecretKey()); err != nil {
		gate.Message = err.Error()
		return gate
	}

	overview, err := s.planService.GetPricingOverview()
	if err != nil {
		gate.Message = fmt.Sprintf("load Stripe catalog: %v", err)
		return gate
	}
	if overview == nil || overview.GetBundle() == nil {
		gate.Message = "Stripe catalog has no bundle"
		return gate
	}
	if len(overview.GetMonthly()) == 0 && len(overview.GetYearly()) == 0 {
		gate.Message = "Stripe catalog has no subscription prices"
		return gate
	}
	if len(overview.GetCreditTopups()) == 0 {
		gate.Message = "Stripe catalog has no fixed credit top-up prices"
		return gate
	}

	base := strings.TrimSpace(envx.Get("PUBLIC_BASE_URL"))
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		gate.Message = "PUBLIC_BASE_URL must be an absolute HTTPS URL"
		return gate
	}
	probeURL := strings.TrimRight(base, "/") + stripeWebhookPath
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, probeURL, nil)
	if err != nil {
		gate.Message = fmt.Sprintf("build public webhook probe: %v", err)
		return gate
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		gate.Message = fmt.Sprintf("public webhook route unreachable: %v", err)
		return gate
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		gate.Message = fmt.Sprintf("public webhook route returned HTTP %d; expected 405 from the application for GET", response.StatusCode)
		return gate
	}

	gate.Ready = true
	gate.Message = fmt.Sprintf("Stripe %s credentials, catalog, HTTPS origin, and webhook route are ready", mode)
	return gate
}
