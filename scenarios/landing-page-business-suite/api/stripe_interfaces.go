package main

import (
	"context"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// StripeCheckoutService handles checkout session creation and price verification.
type StripeCheckoutService interface {
	// CreateCheckoutSession creates a new Stripe checkout session for the given price.
	CreateCheckoutSession(priceID, successURL, cancelURL, customerEmail string) (*landing_page_business_suite_v1.CheckoutSession, error)

	// VerifyStripePrice verifies a price exists and returns its details as a map.
	VerifyStripePrice(key string) (map[string]interface{}, error)

	// VerifyStripePriceTyped verifies a price and returns typed price info.
	VerifyStripePriceTyped(key string) (*StripePriceInfo, error)
}

// StripeSubscriptionService handles subscription verification and management.
type StripeSubscriptionService interface {
	// VerifySubscription checks the subscription status for a user (by email or customer ID).
	VerifySubscription(userIdentity string) (*shared.SubscriptionStatus, error)

	// CancelSubscription cancels the subscription for a user.
	CancelSubscription(userIdentity string) (*landing_page_business_suite_v1.CancelSubscriptionResponse, error)

	// CreateBillingPortalSession creates a Stripe billing portal session for subscription management.
	CreateBillingPortalSession(ctx context.Context, userIdentity, returnURL string) (*landing_page_business_suite_v1.BillingPortalResponse, error)
}

// StripeWebhookService handles incoming Stripe webhooks.
type StripeWebhookService interface {
	// HandleWebhook processes a webhook payload after signature verification.
	HandleWebhook(body []byte, signature string) error

	// VerifyWebhookSignature verifies the Stripe webhook signature.
	VerifyWebhookSignature(payload []byte, signature string) bool
}

// StripeAdminService handles administrative Stripe operations.
type StripeAdminService interface {
	// ListStripeProductsWithPrices lists all products and their prices from Stripe.
	ListStripeProductsWithPrices(ctx context.Context, planStore *commerce.PlanStore) (*StripeImportPreview, error)

	// FetchStripePriceDetails fetches detailed price info from Stripe.
	FetchStripePriceDetails(ctx context.Context, priceID string) (*commerce.StripePriceImport, error)

	// ConfigSnapshot returns the current Stripe configuration.
	ConfigSnapshot() *landing_page_business_suite_v1.StripeConfigSnapshot

	// RefreshConfig reloads the Stripe configuration.
	RefreshConfig(ctx context.Context) error

	// GetSecretValue retrieves a secret value by field name.
	GetSecretValue(field string) (string, bool)
}

// StripeService is the aggregate interface combining all public Stripe functionality.
// Most consumers should use the specific sub-interfaces above for better testability.
type StripeServiceInterface interface {
	StripeCheckoutService
	StripeSubscriptionService
	StripeWebhookService
	commerce.CouponService
	StripeAdminService
}

// Compile-time verification that StripeService implements all interfaces.
var (
	_ StripeCheckoutService     = (*StripeService)(nil)
	_ StripeSubscriptionService = (*StripeService)(nil)
	_ StripeWebhookService      = (*StripeService)(nil)
	_ commerce.CouponService    = (*StripeService)(nil)
	_ StripeAdminService        = (*StripeService)(nil)
	_ StripeServiceInterface    = (*StripeService)(nil)
)
