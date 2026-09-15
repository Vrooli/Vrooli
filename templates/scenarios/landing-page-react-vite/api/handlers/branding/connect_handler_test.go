package branding_test

import (
	"context"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	brandingH "landing-page-react-vite-api/handlers/branding"
	internalbranding "landing-page-react-vite-api/internal/branding"
)

func setup(t *testing.T) *internalbranding.Service {
	t.Helper()
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, internalbranding.Schema)
	// Reset the singleton to a known default so assertions are deterministic.
	_, err := db.Exec(`DELETE FROM site_branding`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO site_branding (id, site_name) VALUES (1, 'My Landing')`)
	require.NoError(t, err)
	return internalbranding.NewService(db)
}

func TestGetBranding(t *testing.T) {
	h := brandingH.NewConnectHandler(brandingH.Deps{Service: setup(t)})
	resp, err := h.GetBranding(context.Background(), connect.NewRequest(&landingv1.GetBrandingRequest{}))
	require.NoError(t, err)
	require.Equal(t, "My Landing", resp.Msg.Branding.SiteName)
}

func TestUpdateBranding(t *testing.T) {
	h := brandingH.NewConnectHandler(brandingH.Deps{Service: setup(t)})
	name := "Acme"
	logo := "https://cdn.example.com/logo.png"
	resp, err := h.UpdateBranding(context.Background(), connect.NewRequest(&landingv1.UpdateBrandingRequest{
		SiteName: &name,
		LogoUrl:  &logo,
	}))
	require.NoError(t, err)
	require.Equal(t, "Acme", resp.Msg.Branding.SiteName)
	require.NotNil(t, resp.Msg.Branding.LogoUrl)
	require.Equal(t, logo, *resp.Msg.Branding.LogoUrl)

	// Partial update preserves the previously-set logo.
	tagline := "Ship faster"
	resp2, err := h.UpdateBranding(context.Background(), connect.NewRequest(&landingv1.UpdateBrandingRequest{
		Tagline: &tagline,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp2.Msg.Branding.LogoUrl)
	require.Equal(t, logo, *resp2.Msg.Branding.LogoUrl)
	require.Equal(t, "Acme", resp2.Msg.Branding.SiteName)
}

func TestGetPublicBranding(t *testing.T) {
	svc := setup(t)
	h := brandingH.NewConnectHandler(brandingH.Deps{Service: svc})
	name := "Acme"
	color := "#112233"
	_, err := h.UpdateBranding(context.Background(), connect.NewRequest(&landingv1.UpdateBrandingRequest{
		SiteName:          &name,
		ThemePrimaryColor: &color,
	}))
	require.NoError(t, err)

	resp, err := h.GetPublicBranding(context.Background(), connect.NewRequest(&landingv1.GetPublicBrandingRequest{}))
	require.NoError(t, err)
	require.Equal(t, "Acme", resp.Msg.Branding.SiteName)
	require.Equal(t, color, resp.Msg.Branding.ThemePrimaryColor)
}

func TestClearBrandingField(t *testing.T) {
	h := brandingH.NewConnectHandler(brandingH.Deps{Service: setup(t)})
	tagline := "temporary"
	_, err := h.UpdateBranding(context.Background(), connect.NewRequest(&landingv1.UpdateBrandingRequest{Tagline: &tagline}))
	require.NoError(t, err)

	resp, err := h.ClearBrandingField(context.Background(), connect.NewRequest(&landingv1.ClearBrandingFieldRequest{Field: "tagline"}))
	require.NoError(t, err)
	require.Nil(t, resp.Msg.Branding.Tagline)
}
