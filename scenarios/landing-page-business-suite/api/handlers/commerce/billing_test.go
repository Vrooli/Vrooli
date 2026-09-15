package billing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

func TestCheckoutRequiresPriceBeforeCallingService(t *testing.T) {
	called, status := false, 0
	deps := testDependencies()
	deps.CreateCheckout = func(string, string, string, string) (*lpbsv1.CheckoutSession, error) { called = true; return nil, nil }
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	Checkout(deps, "checkout_failed", "checkout failed", true).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(`{"customer_email":"a@example.test"}`)))
	if status != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%t", status, called)
	}
}

func TestCheckoutUsesClassifiedProviderError(t *testing.T) {
	status, message := 0, ""
	deps := testDependencies()
	deps.CreateCheckout = func(string, string, string, string) (*lpbsv1.CheckoutSession, error) {
		return nil, errors.New("provider")
	}
	deps.ClassifyError = func(error) (int, string, string, bool) {
		return http.StatusBadGateway, "server_error", "provider unavailable", true
	}
	deps.WriteError = func(_ http.ResponseWriter, got int, gotMessage, _ string) { status, message = got, gotMessage }
	Checkout(deps, "checkout_failed", "checkout failed", true).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(`{"price_id":"price_1","customer_email":"a@example.test","success_url":"https://ok.test","cancel_url":"https://cancel.test"}`)))
	if status != http.StatusBadGateway || message != "provider unavailable" {
		t.Fatalf("status=%d message=%q", status, message)
	}
}

func TestPortalRequiresAuthenticatedUser(t *testing.T) {
	status := 0
	deps := testDependencies()
	deps.UserEmail = func(context.Context) string { return "" }
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	Portal(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/portal", nil))
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d", status)
	}
}

func testDependencies() Dependencies {
	return Dependencies{
		ValidateEmail: func(_ http.ResponseWriter, email string) (string, bool) { return email, true }, NormalizeRedirect: func(_ http.ResponseWriter, value, _ string) (string, bool) { return value, true }, ValidateOptionalURL: func(value string) (string, error) { return value, nil }, CreateCheckout: func(string, string, string, string) (*lpbsv1.CheckoutSession, error) {
			return &lpbsv1.CheckoutSession{}, nil
		}, CreatePortal: func(context.Context, string, string) (any, error) { return map[string]any{}, nil }, UserEmail: func(context.Context) string { return "a@example.test" }, ClassifyError: func(error) (int, string, string, bool) { return 0, "", "", false }, WriteJSON: func(http.ResponseWriter, any) {}, WriteError: func(http.ResponseWriter, int, string, string) {}, Log: func(string, map[string]any) {},
	}
}
