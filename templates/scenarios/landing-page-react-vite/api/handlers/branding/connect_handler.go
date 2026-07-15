package branding

import (
	"context"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalbranding "landing-page-react-vite-api/internal/branding"
)

// Deps wires the branding Connect handler.
type Deps struct {
	Service *internalbranding.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the BrandingService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetBranding(ctx context.Context, _ *connect.Request[landingv1.GetBrandingRequest]) (*connect.Response[landingv1.BrandingResponse], error) {
	b, err := h.deps.Service.Get(ctx)
	if err != nil {
		h.deps.Logger.Printf("branding.GetBranding: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.BrandingResponse{Branding: domainToProto(b)}), nil
}

func (h *connectHandler) UpdateBranding(ctx context.Context, req *connect.Request[landingv1.UpdateBrandingRequest]) (*connect.Response[landingv1.BrandingResponse], error) {
	m := req.Msg
	b, err := h.deps.Service.Update(ctx, internalbranding.UpdateRequest{
		SiteName:               m.SiteName,
		Tagline:                m.Tagline,
		LogoURL:                m.LogoUrl,
		LogoIconURL:            m.LogoIconUrl,
		FaviconURL:             m.FaviconUrl,
		AppleTouchIconURL:      m.AppleTouchIconUrl,
		DefaultTitle:           m.DefaultTitle,
		DefaultDescription:     m.DefaultDescription,
		DefaultOGImageURL:      m.DefaultOgImageUrl,
		ThemePrimaryColor:      m.ThemePrimaryColor,
		ThemeBackgroundColor:   m.ThemeBackgroundColor,
		CanonicalBaseURL:       m.CanonicalBaseUrl,
		GoogleSiteVerification: m.GoogleSiteVerification,
		RobotsTxt:              m.RobotsTxt,
	})
	if err != nil {
		h.deps.Logger.Printf("branding.UpdateBranding: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.BrandingResponse{Branding: domainToProto(b)}), nil
}

func (h *connectHandler) ClearBrandingField(ctx context.Context, req *connect.Request[landingv1.ClearBrandingFieldRequest]) (*connect.Response[landingv1.BrandingResponse], error) {
	if req.Msg.Field == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errFieldRequired)
	}
	if err := h.deps.Service.ClearField(ctx, req.Msg.Field); err != nil {
		h.deps.Logger.Printf("branding.ClearBrandingField(%q): %v", req.Msg.Field, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	b, err := h.deps.Service.Get(ctx)
	if err != nil {
		h.deps.Logger.Printf("branding.ClearBrandingField get: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.BrandingResponse{Branding: domainToProto(b)}), nil
}

func (h *connectHandler) GetPublicBranding(ctx context.Context, _ *connect.Request[landingv1.GetPublicBrandingRequest]) (*connect.Response[landingv1.PublicBrandingResponse], error) {
	b, err := h.deps.Service.Get(ctx)
	if err != nil {
		h.deps.Logger.Printf("branding.GetPublicBranding: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.PublicBrandingResponse{Branding: &landingv1.PublicBranding{
		SiteName:             b.SiteName,
		Tagline:              deref(b.Tagline),
		LogoUrl:              deref(b.LogoURL),
		LogoIconUrl:          deref(b.LogoIconURL),
		FaviconUrl:           deref(b.FaviconURL),
		ThemePrimaryColor:    deref(b.ThemePrimaryColor),
		ThemeBackgroundColor: deref(b.ThemeBackgroundColor),
	}}), nil
}

func domainToProto(b *internalbranding.SiteBranding) *landingv1.SiteBranding {
	return &landingv1.SiteBranding{
		Id:                     int64(b.ID),
		SiteName:               b.SiteName,
		Tagline:                b.Tagline,
		LogoUrl:                b.LogoURL,
		LogoIconUrl:            b.LogoIconURL,
		FaviconUrl:             b.FaviconURL,
		AppleTouchIconUrl:      b.AppleTouchIconURL,
		DefaultTitle:           b.DefaultTitle,
		DefaultDescription:     b.DefaultDescription,
		DefaultOgImageUrl:      b.DefaultOGImageURL,
		ThemePrimaryColor:      b.ThemePrimaryColor,
		ThemeBackgroundColor:   b.ThemeBackgroundColor,
		CanonicalBaseUrl:       b.CanonicalBaseURL,
		GoogleSiteVerification: b.GoogleSiteVerification,
		RobotsTxt:              b.RobotsTxt,
		CreatedAt:              timestamppb.New(b.CreatedAt),
		UpdatedAt:              timestamppb.New(b.UpdatedAt),
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
