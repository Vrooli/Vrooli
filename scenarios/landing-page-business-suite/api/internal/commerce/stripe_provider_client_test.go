package commerce

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type providerRequesterFunc func(context.Context, string, string, io.Reader, string) ([]byte, error)

func (f providerRequesterFunc) Request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	return f(ctx, method, path, body, contentType)
}

func TestStripeProviderClientSubscriptionAndCustomerLookups(t *testing.T) {
	var paths []string
	client := NewStripeProviderClient(providerRequesterFunc(func(_ context.Context, method, path string, _ io.Reader, _ string) ([]byte, error) {
		if method != "GET" {
			t.Fatalf("method = %s, want GET", method)
		}
		paths = append(paths, path)
		switch {
		case strings.HasPrefix(path, "/v1/subscriptions/sub_123"):
			return []byte(`{"id":"sub_123","customer":"cus_123","items":{"data":[{"price":{"id":"price_monthly"}}]}}`), nil
		case strings.HasPrefix(path, "/v1/customers/search?"):
			return []byte(`{"data":[{"id":"cus_123","email":"buyer@example.com"}]}`), nil
		case strings.HasPrefix(path, "/v1/subscriptions?"):
			return []byte(`{"data":[{"id":"sub_latest","customer":"cus_123"}]}`), nil
		default:
			return nil, errors.New("unexpected path")
		}
	}))

	sub, err := client.FetchSubscription(context.Background(), "sub_123")
	if err != nil || sub.ID != "sub_123" || sub.Items.Data[0].Price.ID != "price_monthly" {
		t.Fatalf("subscription = %#v, err = %v", sub, err)
	}
	customer, err := client.FindCustomerByEmail(context.Background(), "buyer@example.com")
	if err != nil || customer == nil || customer.ID != "cus_123" {
		t.Fatalf("customer = %#v, err = %v", customer, err)
	}
	latest, err := client.LatestSubscriptionForCustomer(context.Background(), "cus_123")
	if err != nil || latest == nil || latest.ID != "sub_latest" {
		t.Fatalf("latest = %#v, err = %v", latest, err)
	}
	if len(paths) != 3 {
		t.Fatalf("requests = %d, want 3", len(paths))
	}
}

func TestStripeProviderClientRejectsUnavailableRequester(t *testing.T) {
	_, err := NewStripeProviderClient(nil).FetchSubscription(context.Background(), "sub_123")
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("err = %v", err)
	}
}
