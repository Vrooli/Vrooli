package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/health"

	deploymenthttp "landing-page-business-suite-api/handlers/deployment"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"
)

const stripeWebhookPath = "/api/v1/webhooks/stripe"

// stripePaymentsConfigured reports the locally checkable half of Stripe
// readiness: an explicit mode, present credential fields, a key pair whose
// mode matches, and a catalog with something to sell. It performs no network
// call, so it is safe to evaluate on every health request.
//
// It exists so a deployment whose payments are misconfigured says so in its
// own /health body. Without it, the only signal was the deployment readiness
// gate, which nothing reads after a deploy: a live-mode site carrying test
// keys served a cheerful 200.
func (s *Server) stripePaymentsConfigured(ctx context.Context) error {
	mode := strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE")))
	if mode != "test" && mode != "live" {
		return fmt.Errorf("STRIPE_MODE must be explicitly set to test or live")
	}
	settings, err := s.paymentSettings.GetStripeSettings(ctx)
	if err != nil {
		return fmt.Errorf("read active Stripe credentials: %w", err)
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
		return fmt.Errorf("missing active Stripe credential fields: %s", strings.Join(missing, ", "))
	}
	if err := commerce.ValidateStripeKeyModePair(settings.GetPublishableKey(), settings.GetSecretKey()); err != nil {
		return err
	}
	overview, err := s.planService.GetPricingOverview()
	if err != nil {
		return fmt.Errorf("load Stripe catalog: %w", err)
	}
	switch {
	case overview == nil || overview.GetBundle() == nil:
		return fmt.Errorf("Stripe catalog has no bundle")
	case len(overview.GetMonthly()) == 0 && len(overview.GetYearly()) == 0:
		return fmt.Errorf("Stripe catalog has no subscription prices")
	case len(overview.GetCreditTopups()) == 0:
		return fmt.Errorf("Stripe catalog has no fixed credit top-up prices")
	}
	return nil
}

// paymentsCheck publishes stripePaymentsConfigured as an optional health
// dependency named "payments". Optional is deliberate: a scenario with no
// commerce configured still serves its site, so this degrades health and
// warns the operator rather than failing the deployment.
func (s *Server) paymentsCheck() health.Checker {
	return health.Func("payments", func(ctx context.Context) error {
		if s.paymentSettings == nil || s.planService == nil {
			return nil
		}
		if strings.TrimSpace(envx.Get("STRIPE_MODE")) == "" {
			// Commerce was never configured for this deployment. Silence is
			// correct: nothing promised payments.
			return nil
		}
		return s.stripePaymentsConfigured(ctx)
	})
}

// stripeReadiness is the application-side half of the deployment contract.
// It reports configuration and provider-route gaps without ever returning a
// credential value. The public probe deliberately uses GET: the webhook
// endpoint should reject GET with 405, which proves that the edge reached the
// application without requiring a signed or mutating Stripe event.
func (s *Server) stripeReadiness(ctx context.Context) deploymenthttp.Gate {
	gate := deploymenthttp.Gate{Name: "stripe_commerce"}
	mode := strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE")))
	// The locally checkable half is shared with the "payments" health
	// dependency so the gate and the health body can never disagree.
	if err := s.stripePaymentsConfigured(ctx); err != nil {
		gate.Message = err.Error()
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
