package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

func TestBundleConnectDependenciesListCatalogMapsPlanServiceResult(t *testing.T) {
	service := createTestPlanService(t, testBundle("business", "production"), []planFileFormat{{
		StripePriceID:   "price_business_monthly",
		PlanName:        "Business monthly",
		PlanTier:        "business",
		BillingInterval: "month",
		AmountCents:     4900,
		Currency:        "usd",
		DisplayEnabled:  true,
	}})

	catalog, err := bundleConnectDependencies(service, nil).ListCatalog(context.Background())
	if err != nil {
		t.Fatalf("ListCatalog() error = %v", err)
	}
	if len(catalog) != 1 || catalog[0].GetBundle().GetBundleKey() != "business" {
		t.Fatalf("catalog = %#v, want business bundle", catalog)
	}
	if len(catalog[0].GetPrices()) != 1 || catalog[0].GetPrices()[0].GetStripePriceId() != "price_business_monthly" {
		t.Fatalf("catalog prices = %#v", catalog[0].GetPrices())
	}
}

func TestClassifyBundleConnectError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"not found message", errors.New("bundle not found"), connect.CodeNotFound},
		{"stripe not found", &StripeAPIError{Status: http.StatusNotFound}, connect.CodeNotFound},
		{"stripe gateway", &StripeAPIError{Status: http.StatusBadGateway}, connect.CodeUnavailable},
		{"stripe unavailable", &StripeAPIError{Status: http.StatusServiceUnavailable}, connect.CodeUnavailable},
		{"stripe unauthorized", &StripeAPIError{Status: http.StatusUnauthorized}, connect.CodeUnauthenticated},
		{"stripe forbidden", &StripeAPIError{Status: http.StatusForbidden}, connect.CodePermissionDenied},
		{"stripe invalid request", &StripeAPIError{Status: http.StatusTooManyRequests}, connect.CodeInvalidArgument},
		{"unknown", errors.New("invalid price"), connect.CodeInvalidArgument},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyBundleConnectError(test.err); got != test.want {
				t.Fatalf("classifyBundleConnectError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestRegisterBundleAdminConnectRoutesAppliesAdminGuard(t *testing.T) {
	router := mux.NewRouter()
	guarded := false
	registerBundleAdminConnectRoutes(router, nil, nil, func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			guarded = true
			http.Error(w, "admin authentication required", http.StatusUnauthorized)
		}
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, lpbsconnect.BundleAdminServiceListBundleCatalogProcedure, nil))
	if !guarded {
		t.Fatal("admin guard was not invoked")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
