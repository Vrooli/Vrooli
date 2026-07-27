package main

import (
	"context"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
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

// StripeCouponService handles coupon CRUD operations.
type StripeCouponService interface {
	// ListCoupons returns all coupons from Stripe.
	ListCoupons(ctx context.Context) ([]StripeCoupon, error)

	// GetCoupon retrieves a single coupon by ID.
	GetCoupon(ctx context.Context, couponID string) (*StripeCoupon, error)

	// CreateCoupon creates a new coupon in Stripe.
	CreateCoupon(ctx context.Context, req CreateCouponRequest) (*StripeCoupon, error)

	// UpdateCoupon updates an existing coupon's metadata.
	UpdateCoupon(ctx context.Context, couponID string, req UpdateCouponRequest) (*StripeCoupon, error)

	// DeleteCoupon removes a coupon from Stripe.
	DeleteCoupon(ctx context.Context, couponID string) error

	// GetCouponImportPreview returns a preview of coupons available for import.
	GetCouponImportPreview(ctx context.Context) (*CouponImportPreview, error)

	// GetIntroCouponMap returns the mapping of plan tiers to intro coupon IDs.
	GetIntroCouponMap() map[string]string
}

// StripeAdminService handles administrative Stripe operations.
type StripeAdminService interface {
	// ListStripeProductsWithPrices lists all products and their prices from Stripe.
	ListStripeProductsWithPrices(ctx context.Context, planStore *PlanStore) (*StripeImportPreview, error)

	// FetchStripePriceDetails fetches detailed price info from Stripe.
	FetchStripePriceDetails(ctx context.Context, priceID string) (*StripePriceImport, error)

	// ConfigSnapshot returns the current Stripe configuration.
	ConfigSnapshot() *landing_page_business_suite_v1.StripeConfigSnapshot

	// RefreshConfig reloads the Stripe configuration.
	RefreshConfig(ctx context.Context) error

	// GetSecretValue retrieves a secret value by field name.
	GetSecretValue(field string) (string, bool)
}

// StripeCreditService handles credit wallet operations.
type StripeCreditService interface {
	// AddCredits adds credits to a user's wallet with idempotency protection.
	AddCredits(email string, amount int64, txnType, eventID string, metadata map[string]interface{}) error

	// ConsumeCredits deducts credits from a user's wallet for a given reason.
	ConsumeCredits(ctx context.Context, email string, amount int64, reason string, metadata map[string]interface{}) error

	// GetBalance returns the current credit balance for a user.
	GetBalance(email string) (int64, error)
}

// StripeAccountLinkService handles linking users to Stripe customers.
type StripeAccountLinkService interface {
	// LinkUserToStripeCustomer associates a user email with a Stripe customer ID.
	LinkUserToStripeCustomer(email, customerID string) error

	// LookupCustomerID finds the Stripe customer ID for a user (by email or customer ID).
	LookupCustomerID(userIdentity string) string

	// MigrateCustomerEmail updates all tables when a customer's email changes.
	MigrateCustomerEmail(ctx context.Context, oldEmail, newEmail, customerID string) error
}

// StripeService is the aggregate interface combining all public Stripe functionality.
// Most consumers should use the specific sub-interfaces above for better testability.
type StripeServiceInterface interface {
	StripeCheckoutService
	StripeSubscriptionService
	StripeWebhookService
	StripeCouponService
	StripeAdminService
	StripeCreditService
	StripeAccountLinkService
}

// Compile-time verification that StripeService implements all interfaces.
var (
	_ StripeCheckoutService     = (*StripeService)(nil)
	_ StripeSubscriptionService = (*StripeService)(nil)
	_ StripeWebhookService      = (*StripeService)(nil)
	_ StripeCouponService       = (*StripeService)(nil)
	_ StripeAdminService        = (*StripeService)(nil)
	_ StripeCreditService       = (*StripeService)(nil)
	_ StripeAccountLinkService  = (*StripeService)(nil)
	_ StripeServiceInterface    = (*StripeService)(nil)
)
