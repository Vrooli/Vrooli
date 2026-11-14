package seo_test

import (
	"context"
	"database/sql"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"landing-page-react-vite-api/internal/variantspace"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	seoH "landing-page-react-vite-api/handlers/seo"
	internalbranding "landing-page-react-vite-api/internal/branding"
	internalseo "landing-page-react-vite-api/internal/seo"

	internalvariant "landing-page-react-vite-api/internal/variant"
)

func setup(t *testing.T) (*seoH.Deps, *internalseo.Service, *sql.DB) {
	t.Helper()
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, internalbranding.Schema, internalvariant.Schema)
	_, err := db.Exec(`DELETE FROM site_branding`)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM variants`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO site_branding (id, site_name, default_title, default_description, canonical_base_url)
		VALUES (1, 'My Landing', 'Default Title', 'Default Desc', 'https://example.com')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO variants (slug, name, weight, status) VALUES ('control', 'Control', 50, 'active')`)
	require.NoError(t, err)

	brandingSvc := internalbranding.NewService(db)
	variantSvc := internalvariant.NewService(db, variantspace.Default(), nil)
	svc := internalseo.NewService(brandingSvc, variantSvc)
	return &seoH.Deps{Service: svc}, svc, db
}

func TestGetVariantSEOMergesBrandingDefaults(t *testing.T) {
	deps, _, _ := setup(t)
	h := seoH.NewConnectHandler(*deps)
	resp, err := h.GetVariantSEO(context.Background(), connect.NewRequest(&landingv1.GetVariantSEORequest{Slug: "control"}))
	require.NoError(t, err)
	require.Equal(t, "My Landing", resp.Msg.SiteName)
	require.Equal(t, "Default Title", resp.Msg.Title)
	require.Equal(t, "Default Desc", resp.Msg.Description)
	require.Equal(t, "summary_large_image", resp.Msg.TwitterCard)
	require.Equal(t, "https://example.com/", resp.Msg.CanonicalUrl)
}

func TestUpdateVariantSEOOverridesResolve(t *testing.T) {
	deps, _, _ := setup(t)
	h := seoH.NewConnectHandler(*deps)
	up, err := h.UpdateVariantSEO(context.Background(), connect.NewRequest(&landingv1.UpdateVariantSEORequest{
		Slug: "control",
		Config: &landingv1.VariantSEOConfig{
			Title:         "Override Title",
			CanonicalPath: "/pricing",
			Noindex:       true,
		},
	}))
	require.NoError(t, err)
	require.True(t, up.Msg.Success)
	require.NotEmpty(t, up.Msg.UpdatedAt)

	resp, err := h.GetVariantSEO(context.Background(), connect.NewRequest(&landingv1.GetVariantSEORequest{Slug: "control"}))
	require.NoError(t, err)
	require.Equal(t, "Override Title", resp.Msg.Title)
	require.Equal(t, "https://example.com/pricing", resp.Msg.CanonicalUrl)
	require.True(t, resp.Msg.Noindex)
}

func TestSitemapAndRobots(t *testing.T) {
	_, svc, db := setup(t)
	// Give control an indexable canonical path.
	_, err := db.Exec(`UPDATE variants SET seo_config = $1::jsonb WHERE slug = 'control'`,
		`{"canonical_path":"/pricing"}`)
	require.NoError(t, err)

	xml, err := svc.SitemapXML(context.Background(), "http://localhost")
	require.NoError(t, err)
	require.Contains(t, xml, "https://example.com/")
	require.Contains(t, xml, "https://example.com/pricing")

	robots := svc.RobotsTXT(context.Background())
	require.Contains(t, robots, "Sitemap: https://example.com/sitemap.xml")
}
