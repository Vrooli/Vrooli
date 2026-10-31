package bundles

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func TestUpdateBundlePriceRetainsPartialUpdatePresence(t *testing.T) {
	var received UpdatePriceInput
	handler := NewConnectHandler(ConnectDependencies{
		ListCatalog: func(context.Context) ([]*lpbsv1.BundleCatalogEntry, error) { return nil, nil },
		UpdatePrice: func(_ context.Context, bundle, price string, input UpdatePriceInput) (*sharedv1.PlanOption, error) {
			if bundle != "business" || price != "price_1" {
				t.Fatalf("target = %q/%q", bundle, price)
			}
			received = input
			return &sharedv1.PlanOption{StripePriceId: "price_2"}, nil
		},
	})

	nextID, name, weight, enabled := "price_2", "Updated", int32(0), false
	_, err := handler.UpdateBundlePrice(context.Background(), connect.NewRequest(&lpbsv1.UpdateBundlePriceRequest{
		BundleKey: "business", PriceId: "price_1", StripePriceId: &nextID, PlanName: &name,
		DisplayWeight: &weight, DisplayEnabled: &enabled, FeaturesPresent: ptr(true), Features: []string{},
	}))
	if err != nil {
		t.Fatalf("UpdateBundlePrice() error = %v", err)
	}
	if received.StripePriceID == nil || *received.StripePriceID != "price_2" || received.PlanName == nil || *received.PlanName != "Updated" || received.DisplayWeight == nil || *received.DisplayWeight != 0 || received.DisplayEnabled == nil || *received.DisplayEnabled || received.Features == nil || len(*received.Features) != 0 {
		t.Fatalf("partial input = %#v", received)
	}
}

func TestUpdateBundlePriceRejectsMissingTarget(t *testing.T) {
	handler := NewConnectHandler(ConnectDependencies{})
	_, err := handler.UpdateBundlePrice(context.Background(), connect.NewRequest(&lpbsv1.UpdateBundlePriceRequest{}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid argument", got)
	}
}

func TestConnectRoutesServeGeneratedProceduresAndClassifyFailures(t *testing.T) {
	router := mux.NewRouter()
	RegisterConnectRoutes(router, ConnectDependencies{
		ListCatalog: func(context.Context) ([]*lpbsv1.BundleCatalogEntry, error) {
			return []*lpbsv1.BundleCatalogEntry{{Bundle: &sharedv1.Bundle{BundleKey: "business"}}}, nil
		},
		UpdatePrice: func(context.Context, string, string, UpdatePriceInput) (*sharedv1.PlanOption, error) {
			return nil, errors.New("missing")
		},
		Classify: func(error) connect.Code { return connect.CodeNotFound },
	}, func(next http.HandlerFunc) http.HandlerFunc { return next })
	server := httptest.NewServer(router)
	defer server.Close()
	client := lpbsconnect.NewBundleAdminServiceClient(server.Client(), server.URL)
	listed, err := client.ListBundleCatalog(context.Background(), connect.NewRequest(&lpbsv1.ListBundleCatalogRequest{}))
	if err != nil || listed.Msg.GetBundles()[0].GetBundle().GetBundleKey() != "business" {
		t.Fatalf("list = %#v, err = %v", listed, err)
	}
	_, err = client.UpdateBundlePrice(context.Background(), connect.NewRequest(&lpbsv1.UpdateBundlePriceRequest{BundleKey: "business", PriceId: "missing"}))
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("code = %v, want not found", got)
	}
}

func ptr(value bool) *bool { return &value }
