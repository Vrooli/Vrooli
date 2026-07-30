package main

import (
	"context"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/commerce"
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
	if cfg.source == "database" {
		source = landing_page_business_suite_v1.ConfigSource_CONFIG_SOURCE_DATABASE
	}
	preview := ""
	if cfg.hasPublishable {
		preview = maskValue(cfg.publishableKey)
	}
	return &landing_page_business_suite_v1.StripeConfigSnapshot{PublishableKeyPreview: preview, PublishableKeySet: cfg.hasPublishable, SecretKeySet: cfg.hasSecret, WebhookSecretSet: cfg.hasWebhook, Source: source}
}

// StripeImportPreview is the API-owned transport projection of the
// commerce-owned catalog reconciliation result.
type StripeImportPreview struct {
	BundleKey          string                    `json:"bundle_key,omitempty"`
	BundleProductID    string                    `json:"bundle_product_id,omitempty"`
	BundleProductFound bool                      `json:"bundle_product_found"`
	BundlePlanCount    int                       `json:"bundle_plan_count"`
	Products           []StripeProductWithPrices `json:"products"`
	TotalPrices        int                       `json:"total_prices"`
	ConflictCount      int                       `json:"conflict_count"`
	NewCount           int                       `json:"new_count"`
}

type StripeProductWithPrices struct {
	ProductID       string                       `json:"product_id"`
	ProductName     string                       `json:"product_name"`
	IsCurrentBundle bool                         `json:"is_current_bundle"`
	Prices          []commerce.StripePriceImport `json:"prices"`
}

func (s *StripeService) importProvider(planStore *commerce.PlanStore) *commerce.StripeImportProvider {
	return commerce.NewStripeImportProvider(stripeCouponRequester{service: s}, planStore, logStructuredError)
}

func (s *StripeService) ListStripeProductsWithPrices(ctx context.Context, planStore *commerce.PlanStore) (*StripeImportPreview, error) {
	preview, err := s.importProvider(planStore).ListProductsWithPrices(ctx)
	if err != nil {
		return nil, err
	}
	return stripeImportPreviewForAPI(preview), nil
}

func (s *StripeService) FetchStripePriceDetails(ctx context.Context, priceID string) (*commerce.StripePriceImport, error) {
	return s.importProvider(nil).FetchPrice(ctx, priceID)
}

func stripeImportPreviewForAPI(source *commerce.StripeImportPreview) *StripeImportPreview {
	if source == nil {
		return nil
	}
	result := &StripeImportPreview{BundleKey: source.BundleKey, BundleProductID: source.BundleProductID, BundleProductFound: source.BundleProductFound, BundlePlanCount: source.BundlePlanCount, TotalPrices: source.TotalPrices, ConflictCount: source.ConflictCount, NewCount: source.NewCount, Products: make([]StripeProductWithPrices, 0, len(source.Products))}
	for _, product := range source.Products {
		result.Products = append(result.Products, StripeProductWithPrices{ProductID: product.ProductID, ProductName: product.ProductName, IsCurrentBundle: product.IsCurrentBundle, Prices: product.Prices})
	}
	return result
}
