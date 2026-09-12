package delivery

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	internal "landing-page-business-suite-api/internal/delivery"
)

type connectCatalogStub struct {
	apps      []internal.App
	deleteErr error
	saved     internal.App
}

func (s *connectCatalogStub) ListApps(string) ([]internal.App, error) { return s.apps, nil }
func (s *connectCatalogStub) UpsertApp(app internal.App) (*internal.App, error) {
	s.saved = app
	return &app, nil
}
func (s *connectCatalogStub) DeleteApp(string, string) error { return s.deleteErr }

func TestConnectListDownloadAppsReturnsGeneratedValues(t *testing.T) {
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{apps: []internal.App{{AppKey: "desktop", BundleKey: "bundle", Name: "Desktop", Platforms: []internal.Asset{}}}})
	response, err := h.ListDownloadApps(context.Background(), connect.NewRequest(&lpbsv1.ListDownloadAppsRequest{}))
	if err != nil || len(response.Msg.Apps) != 1 || response.Msg.Apps[0].GetAppKey() != "desktop" {
		t.Fatalf("response=%v err=%v", response, err)
	}
}

func TestConnectSaveDownloadAppRejectsMissingName(t *testing.T) {
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{})
	_, err := h.SaveDownloadApp(context.Background(), connect.NewRequest(&lpbsv1.SaveDownloadAppRequest{AppKey: "desktop", App: &shared.DownloadApp{}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestConnectDeleteDownloadAppMapsNotFound(t *testing.T) {
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{deleteErr: internal.ErrAppNotFound})
	_, err := h.DeleteDownloadApp(context.Background(), connect.NewRequest(&lpbsv1.DeleteDownloadAppRequest{AppKey: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound || !errors.Is(err, internal.ErrAppNotFound) {
		t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestConnectAuthorizeDownloadValidatesBeforeAuthorization(t *testing.T) {
	calls := 0
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			calls++
			return nil, nil
		},
	})

	_, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{Platform: "mac"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || calls != 0 {
		t.Fatalf("code=%s calls=%d err=%v", connect.CodeOf(err), calls, err)
	}
}

func TestConnectAuthorizeDownloadMapsSubscriptionDenial(t *testing.T) {
	denied := errors.New("subscription required")
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail:     func(context.Context) string { return "member@example.com" },
		Authorize:     func(context.Context, string, string, string) (*internal.Asset, error) { return nil, denied },
		ClassifyError: func(error) ErrorKind { return ErrorSubscriptionRequired },
		Log:           func(string, map[string]any) {},
	})

	_, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "mac"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied || !errors.Is(err, denied) {
		t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestConnectAuthorizeDownloadReturnsGeneratedAsset(t *testing.T) {
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			return &internal.Asset{AppKey: "desktop", Platform: "mac", ArtifactURL: "https://downloads.example.test/app.dmg", ArtifactSource: "direct"}, nil
		},
		ClassifyError: func(error) ErrorKind { return "" },
		Log:           func(string, map[string]any) {},
	})

	response, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "mac"}))
	if err != nil || response.Msg.GetAsset().GetArtifactUrl() != "https://downloads.example.test/app.dmg" {
		t.Fatalf("response=%v err=%v", response, err)
	}
}

func TestGeneratedDeliveryProjectionRejectsInt32Overflow(t *testing.T) {
	if _, err := appProto(internal.App{DisplayOrder: 1 << 31}); err == nil {
		t.Fatal("appProto() accepted overflowing display order")
	}
	if _, err := assetProto(internal.Asset{ArtifactCount: 1 << 31}); err == nil {
		t.Fatal("assetProto() accepted overflowing artifact count")
	}
}
