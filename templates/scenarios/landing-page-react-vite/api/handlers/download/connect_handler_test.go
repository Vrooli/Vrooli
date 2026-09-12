package download_test

import (
	"context"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	downloadH "landing-page-react-vite-api/handlers/download"
	internaldownload "landing-page-react-vite-api/internal/download"
)

type stubEntitlements struct{ status string }

func (s *stubEntitlements) GetEntitlements(string) (*internaldownload.Entitlements, error) {
	return &internaldownload.Entitlements{Status: s.status}, nil
}

func TestDownloadCatalogAndAuthorize(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "business_suite")

	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, internaldownload.Schema)
	for _, table := range []string{"download_assets", "download_apps"} {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}

	svc := internaldownload.NewService(db)
	ent := &stubEntitlements{status: "active"}
	authorizer := internaldownload.NewAuthorizer(svc, ent, "business_suite")
	planSvc := plan.NewService(db)
	h := downloadH.NewConnectHandler(downloadH.Deps{Service: svc, Authorizer: authorizer, Plan: planSvc})
	ctx := context.Background()

	_, err := h.CreateDownloadApp(ctx, connect.NewRequest(&landingv1.CreateDownloadAppRequest{App: &landingv1.DownloadApp{
		AppKey: "app", Name: "App",
		Platforms: []*landingv1.DownloadAsset{
			{Platform: "mac", ArtifactUrl: "https://cdn/app.dmg", ReleaseVersion: "1.0", RequiresEntitlement: false},
			{Platform: "windows", ArtifactUrl: "https://cdn/app.exe", ReleaseVersion: "1.0", RequiresEntitlement: true},
		},
	}}))
	require.NoError(t, err)

	list, err := h.ListDownloadApps(ctx, connect.NewRequest(&landingv1.ListDownloadAppsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Apps, 1)
	require.Len(t, list.Msg.Apps[0].Platforms, 2)

	// Ungated asset: no identity needed.
	ungated, err := h.AuthorizeDownload(ctx, connect.NewRequest(&landingv1.AuthorizeDownloadRequest{App: "app", Platform: "mac"}))
	require.NoError(t, err)
	require.Equal(t, "mac", ungated.Msg.Asset.Platform)

	// Gated asset without identity -> InvalidArgument.
	_, err = h.AuthorizeDownload(ctx, connect.NewRequest(&landingv1.AuthorizeDownloadRequest{App: "app", Platform: "windows"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	// Gated asset with active entitlement -> allowed.
	gatedReq := connect.NewRequest(&landingv1.AuthorizeDownloadRequest{App: "app", Platform: "windows"})
	gatedReq.Header().Set("X-User-Email", "user@example.com")
	gated, err := h.AuthorizeDownload(ctx, gatedReq)
	require.NoError(t, err)
	require.Equal(t, "windows", gated.Msg.Asset.Platform)

	// Gated asset with inactive entitlement -> PermissionDenied.
	ent.status = "inactive"
	deniedReq := connect.NewRequest(&landingv1.AuthorizeDownloadRequest{App: "app", Platform: "windows"})
	deniedReq.Header().Set("X-User-Email", "user@example.com")
	_, err = h.AuthorizeDownload(ctx, deniedReq)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}
