package main

import (
	"context"
	"net/http/httptest"
	"testing"
)

// StripeTestConfig holds test-specific Stripe configuration.
// Use this instead of os.Setenv to enable parallel-safe tests.
type StripeTestConfig struct {
	PublishableKey string
	SecretKey      string
	WebhookSecret  string
	APIBase        string
	IntroCoupon    *IntroCouponConfig
}

// DefaultStripeTestConfig returns sensible test defaults.
func DefaultStripeTestConfig() StripeTestConfig {
	return StripeTestConfig{
		PublishableKey: "stripe-test-publishable",
		SecretKey:      "stripe-test-secret",
		WebhookSecret:  "stripe-test-webhook",
	}
}

// WithMockServer returns config with APIBase set to the mock server URL.
func (c StripeTestConfig) WithMockServer(server *httptest.Server) StripeTestConfig {
	c.APIBase = server.URL
	return c
}

// WithIntroCoupon returns config with intro coupon settings.
func (c StripeTestConfig) WithIntroCoupon(enabled bool, couponMap map[string]string) StripeTestConfig {
	c.IntroCoupon = &IntroCouponConfig{
		Enabled:   enabled,
		CouponMap: couponMap,
	}
	return c
}

// WithKeys returns config with custom API keys.
func (c StripeTestConfig) WithKeys(publishable, secret, webhook string) StripeTestConfig {
	c.PublishableKey = publishable
	c.SecretKey = secret
	c.WebhookSecret = webhook
	return c
}

// ConfigureStripeService creates a StripeService with injected test config.
// This bypasses environment variables entirely, enabling parallel tests.
//
// Usage:
//
//	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), server)
//
// Or with intro coupon:
//
//	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{"pro": "coupon_pro"})
//	service := ConfigureStripeService(t, db, cfg, server)
func ConfigureStripeService(t *testing.T, db StripeTestStore, cfg StripeTestConfig, server *httptest.Server) *StripeService {
	t.Helper()

	service := requireTestStripeService(t, db)

	if server != nil {
		cfg.APIBase = server.URL
		service.UseHTTPClient(server.Client())
	}

	// Inject config loader - bypasses os.Getenv entirely
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		apiBase := cfg.APIBase
		if apiBase == "" {
			apiBase = "https://api.stripe.com"
		}
		return stripeRuntimeConfig{
			publishableKey: cfg.PublishableKey,
			secretKey:      cfg.SecretKey,
			webhookSecret:  cfg.WebhookSecret,
			apiBase:        apiBase,
			hasPublishable: cfg.PublishableKey != "",
			hasSecret:      cfg.SecretKey != "",
			hasWebhook:     cfg.WebhookSecret != "",
			source:         "test",
		}, nil
	})

	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("ConfigureStripeService: RefreshConfig failed: %v", err)
	}

	// Inject intro coupon config if provided
	if cfg.IntroCoupon != nil {
		service.UseIntroCouponConfig(*cfg.IntroCoupon)
	}

	return service
}

// ConfigureStripeServiceSimple creates a StripeService without a mock server.
// Use this for tests that don't make actual HTTP calls.
func ConfigureStripeServiceSimple(t *testing.T, db StripeTestStore) *StripeService {
	t.Helper()
	return ConfigureStripeService(t, db, DefaultStripeTestConfig(), nil)
}
