package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/commerce"
)

// --- StripeAdminService Interface Implementation ---
// This file contains admin-only operations: product/price listing, config snapshot, and secret access.

// GetSecretValue returns a specific configuration secret value.
// Allowed fields: "publishable_key", "secret_key", "webhook_secret"
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

// ConfigSnapshot returns a redacted view of the active Stripe configuration.
// Note: PublishableKeyPreview is only set when hasPublishable is true to avoid
// leaking placeholder values that could be mistaken for real configuration.
func (s *StripeService) ConfigSnapshot() *landing_page_business_suite_v1.StripeConfigSnapshot {
	cfg := s.getConfig()
	source := landing_page_business_suite_v1.ConfigSource_CONFIG_SOURCE_ENV
	if cfg.source == "database" {
		source = landing_page_business_suite_v1.ConfigSource_CONFIG_SOURCE_DATABASE
	}

	// Only show preview when a real key is configured (not the placeholder)
	var publishablePreview string
	if cfg.hasPublishable {
		publishablePreview = maskValue(cfg.publishableKey)
	}

	return &landing_page_business_suite_v1.StripeConfigSnapshot{
		PublishableKeyPreview: publishablePreview,
		PublishableKeySet:     cfg.hasPublishable,
		SecretKeySet:          cfg.hasSecret,
		WebhookSecretSet:      cfg.hasWebhook,
		Source:                source,
	}
}

// StripeImportPreview provides a preview of products/prices available for import from Stripe.
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

// StripeProductWithPrices groups a Stripe product with its prices.
type StripeProductWithPrices struct {
	ProductID       string                       `json:"product_id"`
	ProductName     string                       `json:"product_name"`
	IsCurrentBundle bool                         `json:"is_current_bundle"`
	Prices          []commerce.StripePriceImport `json:"prices"`
}

type stripeProduct struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type stripePrice struct {
	ID         string `json:"id"`
	LookupKey  string `json:"lookup_key"`
	Currency   string `json:"currency"`
	UnitAmount int64  `json:"unit_amount"`
	Active     bool   `json:"active"`
	Interval   string
	ProductID  string `json:"product"`
}

// ListStripeProductsWithPrices fetches all products and prices from Stripe for import preview.
func (s *StripeService) ListStripeProductsWithPrices(ctx context.Context, planStore *commerce.PlanStore) (*StripeImportPreview, error) {
	// Fetch all active products from Stripe
	products, err := s.fetchStripeProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch stripe products: %w", err)
	}

	bundleKey := ""
	bundleProductID := ""
	bundlePlanCount := 0
	bundleProductFound := false
	// Get existing price IDs from plan store
	existingPriceIDs := make(map[string]bool)
	if planStore != nil {
		if bundle := planStore.GetBundle(); bundle != nil {
			bundleKey = bundle.BundleKey
			bundleProductID = strings.TrimSpace(bundle.StripeProductId)
		}
		plans := planStore.GetPlans()
		bundlePlanCount = len(plans)
		for _, plan := range plans {
			if plan.StripePriceId != "" {
				existingPriceIDs[plan.StripePriceId] = true
			}
		}
	}

	preview := &StripeImportPreview{
		BundleKey:          bundleKey,
		BundleProductID:    bundleProductID,
		BundleProductFound: false,
		BundlePlanCount:    bundlePlanCount,
		Products:           make([]StripeProductWithPrices, 0, len(products)),
	}

	for _, product := range products {
		isCurrentBundle := bundleProductID != "" && product.ID == bundleProductID
		if isCurrentBundle {
			bundleProductFound = true
		}
		// Fetch prices for this product
		prices, err := s.fetchStripePricesForProduct(ctx, product.ID)
		if err != nil {
			logStructuredError("stripe_price_fetch_failed", map[string]interface{}{
				"product_id": product.ID,
				"error":      err.Error(),
			})
			continue
		}

		if len(prices) == 0 {
			continue
		}

		productWithPrices := StripeProductWithPrices{
			ProductID:       product.ID,
			ProductName:     product.Name,
			IsCurrentBundle: isCurrentBundle,
			Prices:          make([]commerce.StripePriceImport, 0, len(prices)),
		}

		for _, price := range prices {
			existsLocally := existingPriceIDs[price.ID]
			priceImport := commerce.StripePriceImport{
				PriceID:       price.ID,
				LookupKey:     price.LookupKey,
				Currency:      price.Currency,
				AmountCents:   price.UnitAmount,
				Interval:      price.Interval,
				ProductID:     product.ID,
				ProductName:   product.Name,
				Active:        price.Active,
				ExistsLocally: existsLocally,
			}
			productWithPrices.Prices = append(productWithPrices.Prices, priceImport)

			preview.TotalPrices++
			if existsLocally {
				preview.ConflictCount++
			} else {
				preview.NewCount++
			}
		}

		preview.Products = append(preview.Products, productWithPrices)
	}

	preview.BundleProductFound = bundleProductFound
	return preview, nil
}

func (s *StripeService) fetchStripeProducts(ctx context.Context) ([]stripeProduct, error) {
	values := url.Values{}
	values.Set("active", "true")
	values.Set("limit", "100")
	path := "/v1/products?" + values.Encode()

	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []stripeProduct `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe products: %w", err)
	}

	return resp.Data, nil
}

func (s *StripeService) fetchStripePricesForProduct(ctx context.Context, productID string) ([]stripePrice, error) {
	values := url.Values{}
	values.Set("product", productID)
	values.Set("limit", "100")
	path := "/v1/prices?" + values.Encode()

	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			ID         string `json:"id"`
			LookupKey  string `json:"lookup_key"`
			Currency   string `json:"currency"`
			UnitAmount int64  `json:"unit_amount"`
			Active     bool   `json:"active"`
			Recurring  *struct {
				Interval string `json:"interval"`
			} `json:"recurring"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe prices: %w", err)
	}

	prices := make([]stripePrice, 0, len(resp.Data))
	for _, p := range resp.Data {
		price := stripePrice{
			ID:         p.ID,
			LookupKey:  p.LookupKey,
			Currency:   p.Currency,
			UnitAmount: p.UnitAmount,
			Active:     p.Active,
			ProductID:  productID,
		}
		if p.Recurring != nil {
			price.Interval = p.Recurring.Interval
		} else {
			price.Interval = "one_time"
		}
		prices = append(prices, price)
	}

	return prices, nil
}

// FetchStripePriceDetails fetches full details for a single price from Stripe.
func (s *StripeService) FetchStripePriceDetails(ctx context.Context, priceID string) (*commerce.StripePriceImport, error) {
	path := "/v1/prices/" + url.PathEscape(priceID) + "?expand[]=product"
	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var price struct {
		ID         string `json:"id"`
		LookupKey  string `json:"lookup_key"`
		Currency   string `json:"currency"`
		UnitAmount int64  `json:"unit_amount"`
		Active     bool   `json:"active"`
		Recurring  *struct {
			Interval string `json:"interval"`
		} `json:"recurring"`
		Product struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"product"`
	}
	if err := json.Unmarshal(body, &price); err != nil {
		return nil, fmt.Errorf("decode stripe price: %w", err)
	}

	interval := "one_time"
	if price.Recurring != nil {
		interval = price.Recurring.Interval
	}

	return &commerce.StripePriceImport{
		PriceID:     price.ID,
		LookupKey:   price.LookupKey,
		Currency:    price.Currency,
		AmountCents: price.UnitAmount,
		Interval:    interval,
		ProductID:   price.Product.ID,
		ProductName: price.Product.Name,
		Active:      price.Active,
	}, nil
}
