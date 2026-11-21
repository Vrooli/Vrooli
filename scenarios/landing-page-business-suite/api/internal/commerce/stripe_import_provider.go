package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// StripeImportPreview is the provider-neutral result used by admin transport
// to review Stripe catalog changes before importing them.
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
	ProductID       string              `json:"product_id"`
	ProductName     string              `json:"product_name"`
	IsCurrentBundle bool                `json:"is_current_bundle"`
	Prices          []StripePriceImport `json:"prices"`
}

// ListStripeProductsWithPrices discovers the provider catalog through the
// authenticated requester and returns the commerce-owned reconciliation
// projection. API transport code supplies the requester and plan store.
func ListStripeProductsWithPrices(ctx context.Context, requester StripeRequester, planStore *PlanStore, logf func(string, map[string]interface{})) (*StripeImportPreview, error) {
	preview, err := NewStripeImportProvider(requester, planStore, logf).ListProductsWithPrices(ctx)
	if err != nil {
		return nil, err
	}
	return preview, nil
}

// FetchStripePriceDetails fetches one provider price through the commerce
// requester boundary.
func FetchStripePriceDetails(ctx context.Context, requester StripeRequester, priceID string, planStore *PlanStore, logf func(string, map[string]interface{})) (*StripePriceImport, error) {
	return NewStripeImportProvider(requester, planStore, logf).FetchPrice(ctx, priceID)
}

// StripeImportProvider owns provider catalog discovery and reconciliation
// against the local plan catalog. API-root code supplies only authenticated
// requests and observability.
type StripeImportProvider struct {
	requester StripeRequester
	plans     *PlanStore
	logf      func(string, map[string]interface{})
}

func NewStripeImportProvider(requester StripeRequester, plans *PlanStore, logf func(string, map[string]interface{})) *StripeImportProvider {
	return &StripeImportProvider{requester: requester, plans: plans, logf: logf}
}

type stripeImportProduct struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type stripeImportPrice struct {
	ID         string `json:"id"`
	LookupKey  string `json:"lookup_key"`
	Currency   string `json:"currency"`
	UnitAmount int64  `json:"unit_amount"`
	Active     bool   `json:"active"`
	Recurring  *struct {
		Interval string `json:"interval"`
	} `json:"recurring"`
}

func (p *StripeImportProvider) ListProductsWithPrices(ctx context.Context) (*StripeImportPreview, error) {
	products, err := p.products(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch stripe products: %w", err)
	}
	bundleKey, bundleProductID, planCount, existing := p.catalogSnapshot()
	preview := &StripeImportPreview{BundleKey: bundleKey, BundleProductID: bundleProductID, BundlePlanCount: planCount, Products: make([]StripeProductWithPrices, 0, len(products))}
	for _, product := range products {
		prices, err := p.prices(ctx, product.ID)
		if err != nil {
			p.log("stripe_price_fetch_failed", map[string]interface{}{"product_id": product.ID, "error": err.Error()})
			continue
		}
		if len(prices) == 0 {
			continue
		}
		isCurrent := bundleProductID != "" && product.ID == bundleProductID
		if isCurrent {
			preview.BundleProductFound = true
		}
		entry := StripeProductWithPrices{ProductID: product.ID, ProductName: product.Name, IsCurrentBundle: isCurrent, Prices: make([]StripePriceImport, 0, len(prices))}
		for _, price := range prices {
			_, exists := existing[price.ID]
			entry.Prices = append(entry.Prices, StripePriceImport{PriceID: price.ID, LookupKey: price.LookupKey, Currency: price.Currency, AmountCents: price.UnitAmount, Interval: price.interval(), ProductID: product.ID, ProductName: product.Name, Active: price.Active, ExistsLocally: exists})
			preview.TotalPrices++
			if exists {
				preview.ConflictCount++
			} else {
				preview.NewCount++
			}
		}
		preview.Products = append(preview.Products, entry)
	}
	return preview, nil
}

func (p *StripeImportProvider) FetchPrice(ctx context.Context, priceID string) (*StripePriceImport, error) {
	if strings.TrimSpace(priceID) == "" {
		return nil, errors.New("stripe price id is required")
	}
	body, err := p.request(ctx, http.MethodGet, "/v1/prices/"+url.PathEscape(priceID)+"?expand[]=product")
	if err != nil {
		return nil, err
	}
	var price struct {
		stripeImportPrice
		Product struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"product"`
	}
	if err := json.Unmarshal(body, &price); err != nil {
		return nil, fmt.Errorf("decode stripe price: %w", err)
	}
	return &StripePriceImport{PriceID: price.ID, LookupKey: price.LookupKey, Currency: price.Currency, AmountCents: price.UnitAmount, Interval: price.interval(), ProductID: price.Product.ID, ProductName: price.Product.Name, Active: price.Active}, nil
}

func (p *StripeImportProvider) products(ctx context.Context) ([]stripeImportProduct, error) {
	values := url.Values{"active": {"true"}, "limit": {"100"}}
	body, err := p.request(ctx, http.MethodGet, "/v1/products?"+values.Encode())
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []stripeImportProduct `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode stripe products: %w", err)
	}
	return response.Data, nil
}

func (p *StripeImportProvider) prices(ctx context.Context, productID string) ([]stripeImportPrice, error) {
	values := url.Values{"product": {productID}, "limit": {"100"}}
	body, err := p.request(ctx, http.MethodGet, "/v1/prices?"+values.Encode())
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []stripeImportPrice `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode stripe prices: %w", err)
	}
	return response.Data, nil
}

func (p *StripeImportProvider) catalogSnapshot() (string, string, int, map[string]struct{}) {
	existing := make(map[string]struct{})
	if p.plans == nil {
		return "", "", 0, existing
	}
	bundleKey, productID := "", ""
	if bundle := p.plans.GetBundle(); bundle != nil {
		bundleKey, productID = bundle.BundleKey, strings.TrimSpace(bundle.StripeProductId)
	}
	plans := p.plans.GetPlans()
	for _, plan := range plans {
		if plan.StripePriceId != "" {
			existing[plan.StripePriceId] = struct{}{}
		}
	}
	return bundleKey, productID, len(plans), existing
}

func (p *StripeImportProvider) request(ctx context.Context, method, path string) ([]byte, error) {
	if p.requester == nil {
		return nil, errors.New("stripe requester unavailable")
	}
	return p.requester.Request(ctx, method, path, nil, "")
}

func (p *StripeImportProvider) log(event string, fields map[string]interface{}) {
	if p.logf != nil {
		p.logf(event, fields)
	}
}

func (p stripeImportPrice) interval() string {
	if p.Recurring != nil && strings.TrimSpace(p.Recurring.Interval) != "" {
		return p.Recurring.Interval
	}
	return "one_time"
}
