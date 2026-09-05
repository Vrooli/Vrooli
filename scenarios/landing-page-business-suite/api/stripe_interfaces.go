package main

import (
	"context"
	"database/sql"
	"time"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// StripeImportPreview and StripeProductWithPrices are aliases for the
// commerce-owned provider reconciliation model. The API composition layer
// exposes the same JSON projection without duplicating domain types.
type (
	StripeImportPreview     = commerce.StripeImportPreview
	StripeProductWithPrices = commerce.StripeProductWithPrices
)

// GetSecretValue returns a specific configuration secret value.
func (s *StripeService) GetSecretValue(field string) (string, bool) {
	cfg := s.getConfig()
	switch field {
	case "publishable_key":
		return cfg.publishableKey, cfg.hasPublishable
	case "secret_key":
		return cfg.secretKey, cfg.hasSecret
	case "webhook_secret":
		return cfg.webhookSecret, cfg.hasWebhook
	default:
		return "", false
	}
}

// ConfigSnapshot returns a redacted view of active Stripe configuration.
func (s *StripeService) ConfigSnapshot() *landing_page_business_suite_v1.StripeConfigSnapshot {
	cfg := s.getConfig()
	source := landing_page_business_suite_v1.ConfigSource_CONFIG_SOURCE_ENV
	if cfg.source == "database" || cfg.source == "authority" {
		source = landing_page_business_suite_v1.ConfigSource_CONFIG_SOURCE_DATABASE
	}
	preview := ""
	if cfg.hasPublishable {
		preview = maskValue(cfg.publishableKey)
	}
	return &landing_page_business_suite_v1.StripeConfigSnapshot{PublishableKeyPreview: preview, PublishableKeySet: cfg.hasPublishable, SecretKeySet: cfg.hasSecret, WebhookSecretSet: cfg.hasWebhook, Source: source}
}

func (s *StripeService) ListStripeProductsWithPrices(ctx context.Context, planStore *commerce.PlanStore) (*StripeImportPreview, error) {
	preview, err := commerce.ListStripeProductsWithPrices(ctx, stripeCouponRequester{service: s}, planStore, logx.Error)
	if err != nil {
		return nil, err
	}
	return preview, nil
}

func (s *StripeService) FetchStripePriceDetails(ctx context.Context, priceID string) (*commerce.StripePriceImport, error) {
	return commerce.FetchStripePriceDetails(ctx, stripeCouponRequester{service: s}, priceID, nil, logx.Error)
}

func (s *StripeService) webhookService() *commerce.StripeWebhookService {
	return commerce.NewStripeWebhookService(commerce.StripeWebhookOptions{
		DB:          s.db,
		PlanService: s.planService,
		Config: func() commerce.StripeWebhookConfig {
			cfg := s.getConfig()
			return commerce.StripeWebhookConfig{WebhookSecret: cfg.webhookSecret, HasWebhook: cfg.hasWebhook}
		},
		NormalizeEmail: NormalizeEmail,
		LoadCheckoutSession: func(id string) (*commerce.CheckoutSessionRecord, error) {
			record, err := s.loadCheckoutSession(id)
			if err != nil {
				return nil, err
			}
			return &commerce.CheckoutSessionRecord{
				SessionID: record.SessionID, Status: record.Status, PriceID: record.PriceID,
				SessionType: record.SessionType.String, AmountCents: record.AmountCents,
				ScheduleID: record.ScheduleID, CustomerID: record.CustomerID,
				CustomerEmail: record.CustomerEmail, SubscriptionID: record.SubscriptionID,
			}, nil
		},
		ExtractAmount: func(obj map[string]interface{}, record *commerce.CheckoutSessionRecord) int64 {
			var legacy *checkoutSessionRecord
			if record != nil {
				legacy = &checkoutSessionRecord{
					SessionID: record.SessionID, Status: record.Status, PriceID: record.PriceID,
					SessionType: sql.NullString{String: record.SessionType, Valid: record.SessionType != ""}, AmountCents: record.AmountCents,
					ScheduleID: record.ScheduleID, CustomerID: record.CustomerID,
					CustomerEmail: record.CustomerEmail, SubscriptionID: record.SubscriptionID,
				}
			}
			return s.extractAmount(obj, legacy)
		},
		HandleCreditTopup: func(email string, amount int64, plan *commerce.PlanOption, eventID string, metadata map[string]interface{}) error {
			return s.handleCreditTopup(email, amount, plan, eventID, metadata)
		},
		RefreshSubscription:   s.refreshSubscriptionFromStripe,
		PersistSubscription:   s.persistSubscriptionFromStripe,
		CheckIntroEligibility: s.checkIntroEligibility,
		MarkIntroUsed:         s.markIntroUsed,
		ExtractIntroCoupon:    s.extractIntroCouponFromInvoice,
		LogIntroAnomaly:       s.logIntroAnomaly,
		Log:                   logx.Info,
		LogError:              logx.Error,
	})
}

func (s *StripeService) HandleWebhook(body []byte, signature string) error {
	return s.webhookService().HandleWebhook(body, signature)
}

func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) bool {
	return s.webhookService().VerifyWebhookSignature(payload, signature)
}

func (s *StripeService) handleCustomerUpdated(obj map[string]interface{}) error {
	return s.webhookService().HandleCustomerUpdated(obj)
}

func (s *StripeService) persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, status string) error {
	return s.webhookService().PersistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, status)
}

func (s *StripeService) billingIntervalDuration(interval shared.BillingInterval) time.Duration {
	return s.webhookService().BillingIntervalDuration(interval)
}

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
