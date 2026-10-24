package commerce

import (
	"context"
	"io"
	"testing"
)

type stripeImportRequesterFunc func(context.Context, string, string, io.Reader, string) ([]byte, error)

func (f stripeImportRequesterFunc) Request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	return f(ctx, method, path, body, contentType)
}

func TestStripeImportProviderBuildsCatalogPreview(t *testing.T) {
	responses := map[string]string{
		"/v1/products?active=true&limit=100":        `{"data":[{"id":"prod_current","name":"Current","active":true},{"id":"prod_other","name":"Other","active":true}]}`,
		"/v1/prices?limit=100&product=prod_current": `{"data":[{"id":"price_current","lookup_key":"current","currency":"usd","unit_amount":1200,"active":true,"recurring":{"interval":"month"}}]}`,
		"/v1/prices?limit=100&product=prod_other":   `{"data":[{"id":"price_other","currency":"usd","unit_amount":500,"active":true}]}`,
	}
	provider := NewStripeImportProvider(stripeImportRequesterFunc(func(_ context.Context, method, path string, _ io.Reader, _ string) ([]byte, error) {
		if method != "GET" {
			t.Fatalf("method = %q, want GET", method)
		}
		return []byte(responses[path]), nil
	}), nil, nil)

	preview, err := provider.ListProductsWithPrices(context.Background())
	if err != nil {
		t.Fatalf("ListProductsWithPrices() error = %v", err)
	}
	if preview.TotalPrices != 2 || preview.NewCount != 2 || preview.ConflictCount != 0 {
		t.Fatalf("preview counts = %#v", preview)
	}
	if preview.Products[0].Prices[0].Interval != "month" || preview.Products[1].Prices[0].Interval != "one_time" {
		t.Fatalf("interval normalization = %#v", preview.Products)
	}
}

func TestStripeImportProviderFetchPriceRejectsBlankID(t *testing.T) {
	provider := NewStripeImportProvider(nil, nil, nil)
	if _, err := provider.FetchPrice(context.Background(), " "); err == nil {
		t.Fatal("FetchPrice() error = nil, want blank price rejection")
	}
}
