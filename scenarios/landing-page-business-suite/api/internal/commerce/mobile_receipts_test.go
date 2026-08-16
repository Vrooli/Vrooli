package commerce

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGooglePlayDeveloperValidatorUsesServerVerificationAndBindsAccount(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || !strings.Contains(r.URL.Path, "/purchases/subscriptions/pro/tokens/purchase-token") {
			t.Fatalf("unexpected Play lookup: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer server-oauth-token" {
			t.Fatalf("missing server authorization")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"purchaseState":0,"orderId":"GPA.order","productId":"pro","purchaseToken":"purchase-token","obfuscatedExternalAccountId":"account-token","expiryTimeMillis":"4102444800000"}`)),
			Header:     make(http.Header),
		}, nil
	})}
	validator := GooglePlayDeveloperValidator{
		PackageName: "com.vrooli.app",
		ProductID:   "pro",
		OAuthToken: func(context.Context) (string, error) {
			return "server-oauth-token", nil
		},
		ResolveIdentity: func(context.Context, string) (string, error) {
			return "buyer@example.com", nil
		},
		Client: client,
		Now:    func() time.Time { return time.Unix(1700000000, 0) },
	}

	got, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "purchase-token", UserIdentity: "buyer@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ExternalSubscription != "purchase-token" || got.PlanTier != "pro" {
		t.Fatalf("normalized purchase = %+v", got)
	}
}

func TestGooglePlayDeveloperValidatorRejectsUnboundPurchase(t *testing.T) {
	validator := GooglePlayDeveloperValidator{
		PackageName: "com.vrooli.app",
		ProductID:   "pro",
		OAuthToken:  func(context.Context) (string, error) { return "token", nil },
		ResolveIdentity: func(context.Context, string) (string, error) {
			return "different@example.com", nil
		},
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"purchaseState":0,"orderId":"GPA.order","productId":"pro","purchaseToken":"purchase-token","obfuscatedExternalAccountId":"account-token","expiryTimeMillis":"4102444800000"}`)), Header: make(http.Header)}, nil
		})},
	}
	if _, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "purchase-token", UserIdentity: "buyer@example.com"}); err != ErrReceiptBound {
		t.Fatalf("unbound purchase error = %v, want ErrReceiptBound", err)
	}
}
