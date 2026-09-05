package delivery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
	internal "landing-page-business-suite-api/internal/delivery"
)

// ConnectCatalog is the app-catalog portion of delivery's persistence seam.
type ConnectCatalog interface {
	ListApps(string) ([]internal.App, error)
	UpsertApp(internal.App) (*internal.App, error)
	DeleteApp(string, string) error
}

type ConnectHandler struct {
	lpbsconnect.UnimplementedDownloadServiceHandler
	bundleKey     func() string
	catalog       ConnectCatalog
	authorization *ConnectAuthorizationDependencies
}

func NewConnectHandler(bundleKey func() string, catalog ConnectCatalog) *ConnectHandler {
	return &ConnectHandler{bundleKey: bundleKey, catalog: catalog}
}

// ConnectAuthorizationDependencies isolates entitlement authorization from
// catalog administration while keeping both operations on DownloadService.
type ConnectAuthorizationDependencies struct {
	UserEmail      func(context.Context) string
	Authorize      func(context.Context, string, string, string) (*internal.Asset, error)
	ClassifyError  func(error) ErrorKind
	ResolveManaged func(context.Context, int64) (string, bool, error)
	Log            func(string, map[string]any)
}

// WithAuthorization attaches the user-scoped delivery authorization seam.
func (h *ConnectHandler) WithAuthorization(deps ConnectAuthorizationDependencies) *ConnectHandler {
	h.authorization = &deps
	return h
}

func (h *ConnectHandler) AuthorizeDownload(ctx context.Context, request *connect.Request[lpbsv1.AuthorizeDownloadRequest]) (*connect.Response[lpbsv1.AuthorizeDownloadResponse], error) {
	if h.authorization == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("download authorization is not configured"))
	}
	appKey := strings.TrimSpace(request.Msg.GetApp())
	if appKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("app is required"))
	}
	platform := strings.TrimSpace(request.Msg.GetPlatform())
	if platform == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("platform is required"))
	}
	user := h.authorization.UserEmail(ctx)
	if user == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	asset, err := h.authorization.Authorize(ctx, appKey, platform, user)
	if err != nil {
		h.authorization.Log("download_authorization_failed", map[string]any{"app_key": appKey, "platform": platform, "user": user, "error": err.Error()})
		return nil, connectAuthorizationError(h.authorization.ClassifyError(err), err)
	}
	if asset == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("authorize download returned no asset"))
	}
	if asset.ArtifactSource == "managed" && asset.ArtifactID != nil {
		url, found, resolveErr := h.authorization.ResolveManaged(ctx, *asset.ArtifactID)
		if resolveErr != nil {
			h.authorization.Log("download_artifact_resolution_failed", map[string]any{"app_key": appKey, "platform": platform, "artifact_id": *asset.ArtifactID, "error": resolveErr.Error()})
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("resolve managed artifact: %w", resolveErr))
		}
		if !found {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("download artifact not found"))
		}
		asset.ArtifactURL = url
	}
	result, err := assetProto(*asset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.AuthorizeDownloadResponse{Asset: result}), nil
}

func connectAuthorizationError(kind ErrorKind, err error) error {
	switch kind {
	case ErrorNotFound, ErrorAppNotFound:
		return connect.NewError(connect.CodeNotFound, err)
	case ErrorSubscriptionRequired:
		return connect.NewError(connect.CodePermissionDenied, err)
	case ErrorIdentityRequired, ErrorPlatformRequired:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case ErrorEntitlementsUnavailable:
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func (h *ConnectHandler) ListDownloadApps(_ context.Context, _ *connect.Request[lpbsv1.ListDownloadAppsRequest]) (*connect.Response[lpbsv1.ListDownloadAppsResponse], error) {
	apps, err := h.catalog.ListApps(h.bundleKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list download apps: %w", err))
	}
	result, err := appsProto(apps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.ListDownloadAppsResponse{Apps: result}), nil
}

func (h *ConnectHandler) CreateDownloadApp(_ context.Context, request *connect.Request[lpbsv1.CreateDownloadAppRequest]) (*connect.Response[lpbsv1.DownloadAppResponse], error) {
	return h.save(request.Msg.GetApp(), "")
}

func (h *ConnectHandler) SaveDownloadApp(_ context.Context, request *connect.Request[lpbsv1.SaveDownloadAppRequest]) (*connect.Response[lpbsv1.DownloadAppResponse], error) {
	return h.save(request.Msg.GetApp(), request.Msg.GetAppKey())
}

func (h *ConnectHandler) DeleteDownloadApp(_ context.Context, request *connect.Request[lpbsv1.DeleteDownloadAppRequest]) (*connect.Response[lpbsv1.DeleteDownloadAppResponse], error) {
	key := request.Msg.GetAppKey()
	if key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("app_key is required"))
	}
	if err := h.catalog.DeleteApp(h.bundleKey(), key); err != nil {
		if errors.Is(err, internal.ErrAppNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete download app: %w", err))
	}
	return connect.NewResponse(&lpbsv1.DeleteDownloadAppResponse{}), nil
}

func (h *ConnectHandler) save(message *shared.DownloadApp, overrideKey string) (*connect.Response[lpbsv1.DownloadAppResponse], error) {
	payload := appRequestFromProto(message)
	app, err := BuildAppFromPayload(payload, h.bundleKey(), overrideKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	saved, err := h.catalog.UpsertApp(app)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save download app: %w", err))
	}
	result, err := appProto(*saved)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.DownloadAppResponse{App: result}), nil
}

func appRequestFromProto(app *shared.DownloadApp) AppRequest {
	if app == nil {
		return AppRequest{}
	}
	result := AppRequest{AppKey: app.GetAppKey(), Name: app.GetName(), Tagline: app.GetTagline(), Description: app.GetDescription(), IconURL: app.GetIconUrl(), ScreenshotURL: app.GetScreenshotUrl(), InstallOverview: app.GetInstallOverview(), InstallSteps: app.GetInstallSteps()}
	if app.Metadata != nil {
		result.Metadata = app.Metadata.AsMap()
	}
	order := int(app.GetDisplayOrder())
	result.DisplayOrder = &order
	for _, store := range app.GetStorefronts() {
		result.Storefronts = append(result.Storefronts, internal.Storefront{Store: store.GetStore(), Label: store.GetLabel(), URL: store.GetUrl(), Badge: store.GetBadge()})
	}
	for _, asset := range app.GetPlatforms() {
		item := AssetRequest{Platform: asset.GetPlatform(), ArtifactURL: asset.GetArtifactUrl(), ArtifactSource: asset.GetArtifactSource(), ArtifactID: asset.ArtifactId, ReleaseVersion: asset.GetReleaseVersion(), ReleaseNotes: asset.GetReleaseNotes(), Checksum: asset.GetChecksum()}
		required := asset.GetRequiresEntitlement()
		item.RequiresEntitlement = &required
		if asset.Metadata != nil {
			item.Metadata = asset.Metadata.AsMap()
		}
		result.Platforms = append(result.Platforms, item)
	}
	return result
}

func appsProto(apps []internal.App) ([]*shared.DownloadApp, error) {
	result := make([]*shared.DownloadApp, 0, len(apps))
	for _, app := range apps {
		item, err := appProto(app)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func appProto(app internal.App) (*shared.DownloadApp, error) {
	metadata, err := structpb.NewStruct(app.Metadata)
	if err != nil {
		return nil, fmt.Errorf("app %q metadata: %w", app.AppKey, err)
	}
	policy, err := structpb.NewStruct(app.UpdatePolicy)
	if err != nil {
		return nil, fmt.Errorf("app %q update policy: %w", app.AppKey, err)
	}
	platforms := make([]*shared.DownloadAsset, 0, len(app.Platforms))
	for _, asset := range app.Platforms {
		item, err := assetProto(asset)
		if err != nil {
			return nil, err
		}
		platforms = append(platforms, item)
	}
	stores := make([]*shared.DownloadStorefront, 0, len(app.Storefronts))
	for _, store := range app.Storefronts {
		stores = append(stores, &shared.DownloadStorefront{Store: store.Store, Label: store.Label, Url: store.URL, Badge: store.Badge})
	}
	displayOrder, err := int32FromInt(app.DisplayOrder, "display order")
	if err != nil {
		return nil, err
	}
	return &shared.DownloadApp{Id: app.ID, BundleKey: app.BundleKey, AppKey: app.AppKey, Name: app.Name, Tagline: app.Tagline, Description: app.Description, IconUrl: app.IconURL, ScreenshotUrl: app.ScreenshotURL, InstallOverview: app.InstallOverview, InstallSteps: app.InstallSteps, Storefronts: stores, Metadata: metadata, DisplayOrder: displayOrder, UpdateApiKey: app.UpdateAPIKey, UpdatePolicy: policy, Platforms: platforms}, nil
}

func assetProto(asset internal.Asset) (*shared.DownloadAsset, error) {
	metadata, err := structpb.NewStruct(asset.Metadata)
	if err != nil {
		return nil, fmt.Errorf("asset %q metadata: %w", asset.Platform, err)
	}
	artifactCount, err := int32FromInt(asset.ArtifactCount, "artifact count")
	if err != nil {
		return nil, err
	}
	return &shared.DownloadAsset{Id: asset.ID, BundleKey: asset.BundleKey, AppKey: asset.AppKey, Platform: asset.Platform, ArtifactUrl: asset.ArtifactURL, ArtifactSource: asset.ArtifactSource, ArtifactId: asset.ArtifactID, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes, Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement, Metadata: metadata, VariantKey: asset.VariantKey, ArtifactFilename: asset.ArtifactFilename, ArtifactSizeBytes: asset.ArtifactSizeBytes, ArtifactCount: artifactCount}, nil
}

func int32FromInt(value int, field string) (int32, error) {
	if value < -1<<31 || value > 1<<31-1 {
		return 0, fmt.Errorf("%s %d exceeds int32 range", field, value)
	}
	return int32(value), nil // #nosec G115 -- bounds are verified immediately above.
}

// RegisterConnectAppRoutes mounts only the fully implemented app catalog RPCs.
func RegisterConnectAppRoutes(router *mux.Router, bundleKey func() string, catalog ConnectCatalog, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewDownloadServiceHandler(NewConnectHandler(bundleKey, catalog))
	for _, path := range []string{lpbsconnect.DownloadServiceListDownloadAppsProcedure, lpbsconnect.DownloadServiceCreateDownloadAppProcedure, lpbsconnect.DownloadServiceSaveDownloadAppProcedure, lpbsconnect.DownloadServiceDeleteDownloadAppProcedure} {
		router.Handle(path, requireAdmin(service.ServeHTTP)).Methods(http.MethodPost)
	}
}

// RegisterConnectAuthorizationRoute mounts the generated procedure with the
// same user authentication boundary as the legacy download endpoint.
func RegisterConnectAuthorizationRoute(router *mux.Router, bundleKey func() string, catalog ConnectCatalog, authorization ConnectAuthorizationDependencies, requireUser func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewDownloadServiceHandler(NewConnectHandler(bundleKey, catalog).WithAuthorization(authorization))
	router.Handle(lpbsconnect.DownloadServiceAuthorizeDownloadProcedure, requireUser(service.ServeHTTP)).Methods(http.MethodPost)
}
