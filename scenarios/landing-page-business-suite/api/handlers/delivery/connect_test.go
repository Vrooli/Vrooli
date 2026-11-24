package delivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	internal "landing-page-business-suite-api/internal/delivery"
)

type connectCatalogStub struct {
	apps      []internal.App
	deleteErr error
	saved     internal.App
	upserts   int
}

type connectAppLookupContextKey struct{}

func (s *connectCatalogStub) ListApps(string) ([]internal.App, error) { return s.apps, nil }
func (s *connectCatalogStub) UpsertApp(app internal.App) (*internal.App, error) {
	s.upserts++
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

func TestConnectSaveDownloadAppValidatesManagedArtifactBeforePersistence(t *testing.T) {
	artifactID := int64(91)
	checksum := strings.Repeat("a", 64)
	artifact := &internal.Artifact{BundleKey: "bundle", AppKey: "desktop", Platform: "windows", ReleaseVersion: "1.2.3", SHA256: checksum}
	app := &shared.DownloadApp{
		AppKey: "desktop", Name: "Desktop",
		Platforms: []*shared.DownloadAsset{{
			Platform: "windows", ArtifactSource: "managed", ArtifactId: &artifactID,
			ReleaseVersion: "1.2.3", Checksum: checksum,
		}},
	}

	t.Run("matching record saves", func(t *testing.T) {
		catalog := &connectCatalogStub{}
		lookupCalls := 0
		h := NewConnectHandler(func() string { return "bundle" }, catalog).WithArtifactLookup(
			func(ctx context.Context, bundle string, id int64) (*internal.Artifact, error) {
				lookupCalls++
				if bundle != "bundle" || id != artifactID {
					t.Fatalf("artifact lookup bundle=%q id=%d", bundle, id)
				}
				if ctx.Value(connectAppLookupContextKey{}) != "connect-request" {
					t.Fatalf("artifact lookup context value = %#v", ctx.Value(connectAppLookupContextKey{}))
				}
				return artifact, nil
			},
		)
		requestContext := context.WithValue(context.Background(), connectAppLookupContextKey{}, "connect-request")
		response, err := h.CreateDownloadApp(requestContext, connect.NewRequest(&lpbsv1.CreateDownloadAppRequest{App: app}))
		if err != nil || response == nil || lookupCalls != 1 || catalog.upserts != 1 {
			t.Fatalf("response=%v err=%v lookups=%d upserts=%d", response, err, lookupCalls, catalog.upserts)
		}
	})

	t.Run("mismatch refuses before persistence", func(t *testing.T) {
		catalog := &connectCatalogStub{}
		wrong := *artifact
		wrong.SHA256 = strings.Repeat("b", 64)
		h := NewConnectHandler(func() string { return "bundle" }, catalog).WithArtifactLookup(
			func(context.Context, string, int64) (*internal.Artifact, error) { return &wrong, nil },
		)
		_, err := h.CreateDownloadApp(context.Background(), connect.NewRequest(&lpbsv1.CreateDownloadAppRequest{App: app}))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition || catalog.upserts != 0 {
			t.Fatalf("code=%s err=%v upserts=%d", connect.CodeOf(err), err, catalog.upserts)
		}
	})

	t.Run("missing lookup fails closed", func(t *testing.T) {
		catalog := &connectCatalogStub{}
		h := NewConnectHandler(func() string { return "bundle" }, catalog)
		_, err := h.CreateDownloadApp(context.Background(), connect.NewRequest(&lpbsv1.CreateDownloadAppRequest{App: app}))
		if connect.CodeOf(err) != connect.CodeUnavailable || catalog.upserts != 0 {
			t.Fatalf("code=%s err=%v upserts=%d", connect.CodeOf(err), err, catalog.upserts)
		}
	})
}

func TestRegisterConnectAppRoutesBindsManagedArtifactLookup(t *testing.T) {
	artifactID := int64(42)
	checksum := strings.Repeat("c", 64)
	lookups, catalog := 0, &connectCatalogStub{}
	router := mux.NewRouter()
	RegisterConnectAppRoutes(router, func() string { return "bundle" }, catalog,
		func(ctx context.Context, bundle string, id int64) (*internal.Artifact, error) {
			lookups++
			if bundle != "bundle" || id != artifactID {
				t.Fatalf("lookup bundle=%q id=%d", bundle, id)
			}
			return &internal.Artifact{BundleKey: bundle, AppKey: "desktop", Platform: "windows", ReleaseVersion: "2.0", SHA256: checksum}, nil
		},
		func(next http.HandlerFunc) http.HandlerFunc { return next },
	)
	server := httptest.NewServer(router)
	defer server.Close()
	client := lpbsconnect.NewDownloadServiceClient(server.Client(), server.URL)
	_, err := client.CreateDownloadApp(context.Background(), connect.NewRequest(&lpbsv1.CreateDownloadAppRequest{App: &shared.DownloadApp{
		AppKey: "desktop", Name: "Desktop", Platforms: []*shared.DownloadAsset{{
			Platform: "windows", ArtifactSource: "managed", ArtifactId: &artifactID, ReleaseVersion: "2.0", Checksum: checksum,
		}},
	}}))
	if err != nil || lookups != 1 || catalog.upserts != 1 {
		t.Fatalf("err=%v lookups=%d upserts=%d", err, lookups, catalog.upserts)
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

func TestConnectAuthorizeManagedAssetWithoutResolverFailsClosed(t *testing.T) {
	recorded := false
	artifactID := int64(91)
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			return &internal.Asset{AppKey: "desktop", Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ArtifactURL: "https://stale.example/public.zip"}, nil
		},
		Log:         func(string, map[string]any) {},
		RecordEvent: func(context.Context, string, string, string) error { recorded = true; return nil },
	})
	_, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "linux"}))
	if connect.CodeOf(err) != connect.CodeUnavailable || recorded {
		t.Fatalf("code=%s recorded=%v err=%v", connect.CodeOf(err), recorded, err)
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

func TestConnectAuthorizeDownloadAllowsAnonymousUngatedAsset(t *testing.T) {
	var identity string
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "" },
		Authorize: func(_ context.Context, _, _, user string) (*internal.Asset, error) {
			identity = user
			return &internal.Asset{AppKey: "desktop", Platform: "mac", ArtifactURL: "https://downloads.example.test/app.dmg", ArtifactSource: "direct"}, nil
		},
		ClassifyError: func(error) ErrorKind { return "" },
		Log:           func(string, map[string]any) {},
	})

	response, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "mac"}))
	if err != nil || response.Msg.GetAsset().GetArtifactUrl() != "https://downloads.example.test/app.dmg" {
		t.Fatalf("response=%v err=%v", response, err)
	}
	if identity != "" {
		t.Fatalf("anonymous request must pass an empty identity, got %q", identity)
	}
}

func TestConnectAuthorizeDownloadMapsAnonymousGatedAssetToUnauthenticated(t *testing.T) {
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			return nil, internal.ErrIdentityRequired
		},
		ClassifyError: func(error) ErrorKind { return ErrorIdentityRequired },
		Log:           func(string, map[string]any) {},
	})

	_, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "mac"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated || !errors.Is(err, internal.ErrIdentityRequired) {
		t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestConnectAuthorizeDownloadSelectedUsesExactIDAndManagedResolutionContext(t *testing.T) {
	requestCtx := context.WithValue(context.Background(), struct{}{}, "request-context")
	var selectedID int64
	var resolveCtx context.Context
	var authorizeCalls int
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			authorizeCalls++
			return &internal.Asset{ID: 1, Platform: "linux", ArtifactURL: "caller-url"}, nil
		},
		AuthorizeSelected: func(ctx context.Context, app, platform, user string, id int64) (*internal.Asset, error) {
			if app != "desktop" || platform != "linux" || user != "member@example.com" {
				t.Fatalf("selected authorization inputs = %q/%q/%q", app, platform, user)
			}
			selectedID = id
			return &internal.Asset{ID: id, AppKey: app, Platform: platform, ArtifactURL: "https://caller.invalid/forbidden", ArtifactSource: "managed", ArtifactID: ptrInt64(9001)}, nil
		},
		ClassifyError: func(error) ErrorKind { return "" },
		ResolveManaged: func(ctx context.Context, id int64) (string, bool, error) {
			resolveCtx = ctx
			if id != 9001 {
				t.Fatalf("managed artifact ID = %d, want 9001", id)
			}
			return "https://provider.example/authorized", true, nil
		},
		Log: func(string, map[string]any) {},
	})

	assetID := int64(42)
	response, err := h.AuthorizeDownload(requestCtx, connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "linux", AssetId: &assetID}))
	if err != nil || response.Msg.GetAsset().GetId() != assetID || response.Msg.GetAsset().GetArtifactUrl() != "https://provider.example/authorized" {
		t.Fatalf("selected response = %v, error = %v", response, err)
	}
	if selectedID != assetID || authorizeCalls != 0 || resolveCtx != requestCtx {
		t.Fatalf("selected flow = id:%d legacyCalls:%d resolveCtxPreserved:%v", selectedID, authorizeCalls, resolveCtx == requestCtx)
	}
}

func TestConnectAuthorizeDownloadSelectedDoesNotFallbackWhenSelectorMissing(t *testing.T) {
	assetID := int64(42)
	legacyCalls := 0
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			legacyCalls++
			return &internal.Asset{ID: 1, Platform: "linux"}, nil
		},
		ClassifyError: func(error) ErrorKind { return "" },
		Log:           func(string, map[string]any) {},
	})
	_, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "linux", AssetId: &assetID}))
	if connect.CodeOf(err) != connect.CodeUnavailable || legacyCalls != 0 {
		t.Fatalf("missing selector = code:%s legacyCalls:%d error:%v", connect.CodeOf(err), legacyCalls, err)
	}
}

func TestConnectAuthorizeManagedAssetWithoutArtifactIDFailsClosed(t *testing.T) {
	resolved := false
	h := NewConnectHandler(func() string { return "bundle" }, &connectCatalogStub{}).WithAuthorization(ConnectAuthorizationDependencies{
		UserEmail: func(context.Context) string { return "member@example.com" },
		Authorize: func(context.Context, string, string, string) (*internal.Asset, error) {
			return &internal.Asset{AppKey: "desktop", Platform: "linux", ArtifactSource: "managed", ArtifactURL: "https://stale.example/public.zip"}, nil
		},
		ClassifyError:  func(error) ErrorKind { return "" },
		ResolveManaged: func(context.Context, int64) (string, bool, error) { resolved = true; return "", false, nil },
		Log:            func(string, map[string]any) {},
	})
	_, err := h.AuthorizeDownload(context.Background(), connect.NewRequest(&lpbsv1.AuthorizeDownloadRequest{App: "desktop", Platform: "linux"}))
	if connect.CodeOf(err) != connect.CodeUnavailable || resolved {
		t.Fatalf("code=%s resolved=%v err=%v", connect.CodeOf(err), resolved, err)
	}
}

func ptrInt64(value int64) *int64 { return &value }

func TestGeneratedDeliveryProjectionRejectsInt32Overflow(t *testing.T) {
	if _, err := appProto(internal.App{DisplayOrder: 1 << 31}); err == nil {
		t.Fatal("appProto() accepted overflowing display order")
	}
	if _, err := assetProto(internal.Asset{ArtifactCount: 1 << 31}); err == nil {
		t.Fatal("assetProto() accepted overflowing artifact count")
	}
}
