package config_test

import (
	"context"
	"database/sql"
	"landing-page-react-vite-api/internal/branding"
	"landing-page-react-vite-api/internal/content"
	"landing-page-react-vite-api/internal/download"
	"landing-page-react-vite-api/internal/landingconfig"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"landing-page-react-vite-api/internal/variant"
	"landing-page-react-vite-api/internal/variantspace"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	configH "landing-page-react-vite-api/handlers/config"
)

func handler(t *testing.T, db *sql.DB) *configH.Deps {
	t.Helper()
	space := variantspace.Load()
	contentSvc := content.NewService(db)
	variantSvc := variant.NewService(db, space, contentSvc)
	svc := landingconfig.NewService(variantSvc, contentSvc, plan.NewService(db), download.NewService(db), branding.NewService(db))
	return &configH.Deps{Service: svc}
}

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, variant.Schema, content.Schema, plan.Schema, download.Schema, branding.Schema)
	for _, table := range []string{"content_sections", "variant_axes", "variants", "download_assets", "download_apps", "bundle_prices", "bundle_products", "site_branding"} {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	return db
}

func TestFallbackWhenEmpty(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "business_suite")
	t.Setenv("BUNDLE_ENVIRONMENT", "production")
	db := newDB(t)

	h := configH.NewConnectHandler(*handler(t, db))
	resp, err := h.GetLandingConfig(context.Background(), connect.NewRequest(&landingv1.GetLandingConfigRequest{}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Fallback, "empty database should yield the fallback payload")
	require.NotEmpty(t, resp.Msg.Sections)
	require.Equal(t, "hero", resp.Msg.Sections[0].SectionType)
}

func TestLivePayload(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "business_suite")
	t.Setenv("BUNDLE_ENVIRONMENT", "production")
	db := newDB(t)

	productID := billingfix.UpsertBundleProduct(t, db, "business_suite", "Business Suite", "prod_bs", "production", 1_000_000, 0.001, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_m", "Monthly", "pro", "month", "usd",
		4900, false, "none", 0, 0, "k", 1_000_000, 0, 1, 20, "none", "subscription", nil)

	var variantID int64
	require.NoError(t, db.QueryRow(`INSERT INTO variants (slug, name, description, weight, status) VALUES ('control','Control','',50,'active') RETURNING id`).Scan(&variantID))
	_, err := db.Exec(`INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES ($1, 'hero', '{"headline":"Hi"}'::jsonb, 0, TRUE)`, variantID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO download_apps (bundle_key, app_key, name) VALUES ('business_suite','desktop','Desktop')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO site_branding (id, site_name) VALUES (1, 'My Landing') ON CONFLICT (id) DO NOTHING`)
	require.NoError(t, err)

	h := configH.NewConnectHandler(*handler(t, db))
	resp, err := h.GetLandingConfig(context.Background(), connect.NewRequest(&landingv1.GetLandingConfigRequest{VariantSlug: "control"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.Fallback, "seeded database should yield a live payload")
	require.Equal(t, "control", resp.Msg.Variant.Slug)
	require.NotEmpty(t, resp.Msg.Sections)
	require.NotNil(t, resp.Msg.Pricing)
	require.Len(t, resp.Msg.Pricing.Monthly, 1)
	require.Len(t, resp.Msg.Downloads, 1)
	require.NotNil(t, resp.Msg.Header)
	require.NotNil(t, resp.Msg.Branding)
	require.Equal(t, "My Landing", resp.Msg.Branding.SiteName)
}
