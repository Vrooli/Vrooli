package download

import (
	"context"
	"errors"
	"landing-page-react-vite-api/internal/plan"
	"log"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internaldownload "landing-page-react-vite-api/internal/download"
)

// userHeader carries the caller identity for gated downloads (X-User-Email).
const userHeader = "X-User-Email"

// Deps wires the Download Connect handler over the download service, its
// entitlement-gating authorizer, and the plan service (for the bundle key).
type Deps struct {
	Service    *internaldownload.Service
	Authorizer *internaldownload.Authorizer
	Plan       *plan.Service
	Logger     *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the DownloadService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) AuthorizeDownload(_ context.Context, req *connect.Request[landingv1.AuthorizeDownloadRequest]) (*connect.Response[landingv1.AuthorizeDownloadResponse], error) {
	user := req.Header().Get(userHeader)
	asset, err := h.deps.Authorizer.Authorize(req.Msg.App, req.Msg.Platform, user)
	if err != nil {
		return nil, authorizeError(err)
	}
	return connect.NewResponse(&landingv1.AuthorizeDownloadResponse{Asset: assetToProto(asset)}), nil
}

func (h *connectHandler) ListDownloadApps(_ context.Context, _ *connect.Request[landingv1.ListDownloadAppsRequest]) (*connect.Response[landingv1.ListDownloadAppsResponse], error) {
	apps, err := h.deps.Service.ListApps(h.deps.Plan.BundleKey())
	if err != nil {
		h.deps.Logger.Printf("download.ListDownloadApps: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*landingv1.DownloadApp, 0, len(apps))
	for i := range apps {
		out = append(out, appToProto(&apps[i]))
	}
	return connect.NewResponse(&landingv1.ListDownloadAppsResponse{Apps: out}), nil
}

func (h *connectHandler) CreateDownloadApp(_ context.Context, req *connect.Request[landingv1.CreateDownloadAppRequest]) (*connect.Response[landingv1.DownloadAppResponse], error) {
	return h.upsert(req.Msg.App, "")
}

func (h *connectHandler) SaveDownloadApp(_ context.Context, req *connect.Request[landingv1.SaveDownloadAppRequest]) (*connect.Response[landingv1.DownloadAppResponse], error) {
	return h.upsert(req.Msg.App, req.Msg.AppKey)
}

func (h *connectHandler) upsert(proto *landingv1.DownloadApp, overrideKey string) (*connect.Response[landingv1.DownloadAppResponse], error) {
	app, err := appFromProto(proto, h.deps.Plan.BundleKey(), overrideKey)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	saved, err := h.deps.Service.UpsertDownloadApp(app)
	if err != nil {
		h.deps.Logger.Printf("download.upsert: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&landingv1.DownloadAppResponse{App: appToProto(saved)}), nil
}

func authorizeError(err error) error {
	switch {
	case errors.Is(err, internaldownload.ErrNotFound), errors.Is(err, internaldownload.ErrAppNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, internaldownload.ErrRequiresActive):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, internaldownload.ErrIdentityRequired), errors.Is(err, internaldownload.ErrPlatformRequired):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internaldownload.ErrEntitlementsUnavail):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func appFromProto(p *landingv1.DownloadApp, bundleKey, overrideKey string) (internaldownload.App, error) {
	if p == nil {
		return internaldownload.App{}, errors.New("app payload is required")
	}
	appKey := strings.TrimSpace(overrideKey)
	if appKey == "" {
		appKey = strings.TrimSpace(p.AppKey)
	}
	if appKey == "" {
		return internaldownload.App{}, errors.New("app_key is required")
	}

	app := internaldownload.App{
		BundleKey:       bundleKey,
		AppKey:          appKey,
		Name:            strings.TrimSpace(p.Name),
		Tagline:         strings.TrimSpace(p.Tagline),
		Description:     strings.TrimSpace(p.Description),
		InstallOverview: strings.TrimSpace(p.InstallOverview),
		InstallSteps:    filterStrings(p.InstallSteps),
		Metadata:        structToMap(p.Metadata),
		DisplayOrder:    int(p.DisplayOrder),
	}
	if app.Name == "" {
		return internaldownload.App{}, errors.New("name is required")
	}
	for _, sf := range p.Storefronts {
		if strings.TrimSpace(sf.Url) == "" {
			return internaldownload.App{}, errors.New("storefront url is required when storefront entries are provided")
		}
		app.Storefronts = append(app.Storefronts, internaldownload.Storefront{Store: sf.Store, Label: sf.Label, URL: sf.Url, Badge: sf.Badge})
	}
	for _, platform := range p.Platforms {
		if strings.TrimSpace(platform.Platform) == "" {
			return internaldownload.App{}, errors.New("platform is required for all installers")
		}
		if strings.TrimSpace(platform.ArtifactUrl) == "" {
			return internaldownload.App{}, errors.New("artifact_url is required for platform " + platform.Platform)
		}
		if strings.TrimSpace(platform.ReleaseVersion) == "" {
			return internaldownload.App{}, errors.New("release_version is required for platform " + platform.Platform)
		}
		app.Platforms = append(app.Platforms, internaldownload.Asset{
			BundleKey:           bundleKey,
			AppKey:              appKey,
			Platform:            strings.TrimSpace(platform.Platform),
			ArtifactURL:         strings.TrimSpace(platform.ArtifactUrl),
			ReleaseVersion:      strings.TrimSpace(platform.ReleaseVersion),
			ReleaseNotes:        strings.TrimSpace(platform.ReleaseNotes),
			Checksum:            strings.TrimSpace(platform.Checksum),
			RequiresEntitlement: platform.RequiresEntitlement,
			Metadata:            structToMap(platform.Metadata),
		})
	}
	return app, nil
}

func appToProto(a *internaldownload.App) *landingv1.DownloadApp {
	out := &landingv1.DownloadApp{
		BundleKey:       a.BundleKey,
		AppKey:          a.AppKey,
		Name:            a.Name,
		Tagline:         a.Tagline,
		Description:     a.Description,
		InstallOverview: a.InstallOverview,
		InstallSteps:    a.InstallSteps,
		Metadata:        mapToStruct(a.Metadata),
		DisplayOrder:    int32(a.DisplayOrder),
	}
	for _, sf := range a.Storefronts {
		out.Storefronts = append(out.Storefronts, &landingv1.DownloadStorefront{Store: sf.Store, Label: sf.Label, Url: sf.URL, Badge: sf.Badge})
	}
	for i := range a.Platforms {
		out.Platforms = append(out.Platforms, assetToProto(&a.Platforms[i]))
	}
	return out
}

func assetToProto(a *internaldownload.Asset) *landingv1.DownloadAsset {
	return &landingv1.DownloadAsset{
		Id:                  a.ID,
		BundleKey:           a.BundleKey,
		AppKey:              a.AppKey,
		Platform:            a.Platform,
		ArtifactUrl:         a.ArtifactURL,
		ReleaseVersion:      a.ReleaseVersion,
		ReleaseNotes:        a.ReleaseNotes,
		Checksum:            a.Checksum,
		RequiresEntitlement: a.RequiresEntitlement,
		Metadata:            mapToStruct(a.Metadata),
	}
}

func filterStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func structToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func mapToStruct(m map[string]interface{}) *structpb.Struct {
	if len(m) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}
