package pricing

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestGetPricingReturnsGeneratedPricingResponse(t *testing.T) {
	handler := NewHandler(func() (*sharedv1.PricingOverview, error) {
		return &sharedv1.PricingOverview{Bundle: &sharedv1.Bundle{BundleKey: "starter"}}, nil
	})
	response, err := handler.GetPricing(context.Background(), connect.NewRequest(&lpbsv1.GetPricingRequest{}))
	if err != nil {
		t.Fatalf("GetPricing() error = %v", err)
	}
	if got := response.Msg.GetPricing().GetBundle().GetBundleKey(); got != "starter" {
		t.Fatalf("bundle key = %q, want starter", got)
	}
}

func TestGetPricingRejectsUnsupportedOrUnknownPublicCatalogRequests(t *testing.T) {
	handler := NewHandler(func() (*sharedv1.PricingOverview, error) {
		return &sharedv1.PricingOverview{Bundle: &sharedv1.Bundle{BundleKey: "starter"}}, nil
	})
	for _, request := range []*lpbsv1.GetPricingRequest{{IncludeHidden: true}, {BundleKey: "other"}} {
		_, err := handler.GetPricing(context.Background(), connect.NewRequest(request))
		if err == nil {
			t.Fatal("GetPricing() error = nil, want Connect error")
		}
		if got := connect.CodeOf(err); got != connect.CodeInvalidArgument && got != connect.CodeNotFound {
			t.Fatalf("GetPricing() code = %v", got)
		}
	}
}

func TestGetPricingReturnsInternalErrorWhenCatalogLoadFails(t *testing.T) {
	handler := NewHandler(func() (*sharedv1.PricingOverview, error) { return nil, errors.New("store unavailable") })
	_, err := handler.GetPricing(context.Background(), connect.NewRequest(&lpbsv1.GetPricingRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("GetPricing() code = %v, want %v", got, connect.CodeInternal)
	}
}

func TestRegisterRoutesServesGeneratedConnectProcedure(t *testing.T) {
	router := mux.NewRouter()
	RegisterRoutes(router, func() (*sharedv1.PricingOverview, error) {
		return &sharedv1.PricingOverview{Bundle: &sharedv1.Bundle{BundleKey: "starter"}}, nil
	})
	server := httptest.NewServer(router)
	defer server.Close()

	client := lpbsconnect.NewPricingServiceClient(server.Client(), server.URL)
	response, err := client.GetPricing(context.Background(), connect.NewRequest(&lpbsv1.GetPricingRequest{}))
	if err != nil {
		t.Fatalf("generated Connect client GetPricing() error = %v", err)
	}
	if got := response.Msg.GetPricing().GetBundle().GetBundleKey(); got != "starter" {
		t.Fatalf("bundle key = %q, want starter", got)
	}
}
