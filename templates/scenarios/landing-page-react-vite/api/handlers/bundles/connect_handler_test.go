package bundles_test

import (
	"context"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	bundlesH "landing-page-react-vite-api/handlers/bundles"
)

func TestListCatalogAndUpdatePrice(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "catalog_bundle")
	t.Setenv("BUNDLE_ENVIRONMENT", "production")

	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema)
	_, err := db.Exec(`DELETE FROM bundle_products`)
	require.NoError(t, err)

	productID := billingfix.UpsertBundleProduct(t, db, "catalog_bundle", "Catalog Bundle", "prod_catalog", "production", 1_000_000, 0.01, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_catalog", "Catalog Plan", "pro", "month", "usd",
		4999, false, "none", 0, 0, "k", 1_000_000, 0, 1, 10, "none", "subscription", nil)

	h := bundlesH.NewConnectHandler(bundlesH.Deps{Plan: plan.NewService(db)})
	ctx := context.Background()

	list, err := h.ListBundleCatalog(ctx, connect.NewRequest(&landingv1.ListBundleCatalogRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Bundles, 1)
	require.Equal(t, "catalog_bundle", list.Msg.Bundles[0].Bundle.BundleKey)
	require.Len(t, list.Msg.Bundles[0].Prices, 1)

	newName := "Renamed Plan"
	highlight := true
	updated, err := h.UpdateBundlePrice(ctx, connect.NewRequest(&landingv1.UpdateBundlePriceRequest{
		BundleKey: "catalog_bundle", PriceId: "price_catalog",
		PlanName: &newName, Highlight: &highlight, Features: []string{"A", "B"},
	}))
	require.NoError(t, err)
	require.Equal(t, "Renamed Plan", updated.Msg.Price.PlanName)
	require.Contains(t, updated.Msg.Price.Metadata, "features")
	require.Contains(t, updated.Msg.Price.Metadata, "highlight")
}
