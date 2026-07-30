package variant

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/experimentation"
)

// BrandingConnectHandler translates generated branding procedures into the
// experimentation configuration domain without API-root composition.
type BrandingConnectHandler struct{ store *experimentation.ConfigStore }

func NewBrandingConnectHandler(store *experimentation.ConfigStore) BrandingConnectHandler {
	return BrandingConnectHandler{store: store}
}

// BrandingProto maps branding configuration to its generated public contract.
func BrandingProto(value *experimentation.SiteBranding) *lpbsv1.SiteBranding {
	if value == nil {
		return &lpbsv1.SiteBranding{}
	}
	port := func(v *int) *int32 {
		if v == nil || *v < -1<<31 || *v > 1<<31-1 {
			return nil
		}
		result := int32(*v)
		return &result
	}
	return &lpbsv1.SiteBranding{Id: value.ID, SiteName: value.SiteName, Tagline: value.Tagline, LogoUrl: value.LogoURL, LogoIconUrl: value.LogoIconURL, FaviconUrl: value.FaviconURL, AppleTouchIconUrl: value.AppleTouchIconURL, DefaultTitle: value.DefaultTitle, DefaultDescription: value.DefaultDescription, DefaultOgImageUrl: value.DefaultOGImageURL, ThemePrimaryColor: value.ThemePrimaryColor, ThemeBackgroundColor: value.ThemeBackgroundColor, CanonicalBaseUrl: value.CanonicalBaseURL, GoogleSiteVerification: value.GoogleSiteVerification, RobotsTxt: value.RobotsTxt, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt), SupportChatUrl: value.SupportChatURL, SupportEmail: value.SupportEmail, SmtpHost: value.SMTPHost, SmtpPort: port(value.SMTPPort), SmtpUsername: value.SMTPUsername, SmtpPassword: value.SMTPPassword, SmtpFrom: value.SMTPFrom, ComingSoonEnabled: value.ComingSoonEnabled, ComingSoonMessage: value.ComingSoonMessage}
}

func brandingUpdate(input *lpbsv1.UpdateBrandingRequest) *experimentation.BrandingUpdateRequest {
	port := func(v *int32) *int {
		if v == nil {
			return nil
		}
		result := int(*v)
		return &result
	}
	return &experimentation.BrandingUpdateRequest{SiteName: input.SiteName, Tagline: input.Tagline, LogoURL: input.LogoUrl, LogoIconURL: input.LogoIconUrl, FaviconURL: input.FaviconUrl, AppleTouchIconURL: input.AppleTouchIconUrl, DefaultTitle: input.DefaultTitle, DefaultDescription: input.DefaultDescription, DefaultOGImageURL: input.DefaultOgImageUrl, ThemePrimaryColor: input.ThemePrimaryColor, ThemeBackgroundColor: input.ThemeBackgroundColor, CanonicalBaseURL: input.CanonicalBaseUrl, GoogleSiteVerification: input.GoogleSiteVerification, RobotsTxt: input.RobotsTxt, SupportChatURL: input.SupportChatUrl, SupportEmail: input.SupportEmail, SMTPHost: input.SmtpHost, SMTPPort: port(input.SmtpPort), SMTPUsername: input.SmtpUsername, SMTPPassword: input.SmtpPassword, SMTPFrom: input.SmtpFrom, ComingSoonEnabled: input.ComingSoonEnabled, ComingSoonMessage: input.ComingSoonMessage}
}

func (h BrandingConnectHandler) GetBranding(context.Context, *connect.Request[lpbsv1.GetBrandingRequest]) (*connect.Response[lpbsv1.BrandingResponse], error) {
	return connect.NewResponse(&lpbsv1.BrandingResponse{Branding: BrandingProto(h.store.GetBranding())}), nil
}

func (h BrandingConnectHandler) UpdateBranding(_ context.Context, request *connect.Request[lpbsv1.UpdateBrandingRequest]) (*connect.Response[lpbsv1.BrandingResponse], error) {
	updated, err := h.store.UpdateBranding(brandingUpdate(request.Msg))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update branding: %w", err))
	}
	return connect.NewResponse(&lpbsv1.BrandingResponse{Branding: BrandingProto(updated)}), nil
}

func (h BrandingConnectHandler) ClearBrandingField(_ context.Context, request *connect.Request[lpbsv1.ClearBrandingFieldRequest]) (*connect.Response[lpbsv1.BrandingResponse], error) {
	if request.Msg.GetField() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("field is required"))
	}
	if err := h.store.ClearBrandingField(request.Msg.GetField()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("clear branding field: %w", err))
	}
	return connect.NewResponse(&lpbsv1.BrandingResponse{Branding: BrandingProto(h.store.GetBranding())}), nil
}

func (h BrandingConnectHandler) GetPublicBranding(context.Context, *connect.Request[lpbsv1.GetPublicBrandingRequest]) (*connect.Response[lpbsv1.PublicBrandingResponse], error) {
	b := h.store.GetBranding()
	if b == nil {
		return connect.NewResponse(&lpbsv1.PublicBrandingResponse{Branding: &lpbsv1.PublicBranding{}}), nil
	}
	return connect.NewResponse(&lpbsv1.PublicBrandingResponse{Branding: &lpbsv1.PublicBranding{SiteName: b.SiteName, Tagline: derefString(b.Tagline), LogoUrl: derefString(b.LogoURL), LogoIconUrl: derefString(b.LogoIconURL), FaviconUrl: derefString(b.FaviconURL), ThemePrimaryColor: derefString(b.ThemePrimaryColor), ThemeBackgroundColor: derefString(b.ThemeBackgroundColor), SupportChatUrl: derefString(b.SupportChatURL), ComingSoonEnabled: derefBool(b.ComingSoonEnabled), ComingSoonMessage: derefString(b.ComingSoonMessage)}}), nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func derefBool(value *bool) bool { return value != nil && *value }

// RegisterBrandingConnectRoutes mounts public and administrator branding procedures.
func RegisterBrandingConnectRoutes(router *mux.Router, store *experimentation.ConfigStore, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewBrandingServiceHandler(NewBrandingConnectHandler(store))
	mount := func(path string, admin bool) {
		handler := http.HandlerFunc(service.ServeHTTP)
		if admin {
			handler = requireAdmin(handler)
		}
		router.Handle(path, handler).Methods(http.MethodPost)
	}
	mount(lpbsconnect.BrandingServiceGetBrandingProcedure, true)
	mount(lpbsconnect.BrandingServiceUpdateBrandingProcedure, true)
	mount(lpbsconnect.BrandingServiceClearBrandingFieldProcedure, true)
	mount(lpbsconnect.BrandingServiceGetPublicBrandingProcedure, false)
}

var _ lpbsconnect.BrandingServiceHandler = BrandingConnectHandler{}
