package variant

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/administration"
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
	// SMTP password is authority-backed write-only material. Keep the wire
	// field present for schema compatibility, but never project its value back
	// to a browser or API caller, including when a test/in-process store returns
	// a legacy populated model.
	return &lpbsv1.SiteBranding{Id: value.ID, SiteName: value.SiteName, Tagline: value.Tagline, LogoUrl: value.LogoURL, LogoIconUrl: value.LogoIconURL, FaviconUrl: value.FaviconURL, AppleTouchIconUrl: value.AppleTouchIconURL, DefaultTitle: value.DefaultTitle, DefaultDescription: value.DefaultDescription, DefaultOgImageUrl: value.DefaultOGImageURL, ThemePrimaryColor: value.ThemePrimaryColor, ThemeBackgroundColor: value.ThemeBackgroundColor, CanonicalBaseUrl: value.CanonicalBaseURL, GoogleSiteVerification: value.GoogleSiteVerification, RobotsTxt: value.RobotsTxt, CreatedAt: timestamppb.New(value.CreatedAt), UpdatedAt: timestamppb.New(value.UpdatedAt), SupportChatUrl: value.SupportChatURL, SupportEmail: value.SupportEmail, SmtpHost: value.SMTPHost, SmtpPort: port(value.SMTPPort), SmtpUsername: value.SMTPUsername, SmtpPassword: nil, SmtpFrom: value.SMTPFrom, ComingSoonEnabled: value.ComingSoonEnabled, ComingSoonMessage: value.ComingSoonMessage, LegalName: value.LegalName, ContactAddress: value.ContactAddress, PrivacyPolicyMarkdown: value.PrivacyPolicyMarkdown, TermsMarkdown: value.TermsMarkdown, PrivacyEffectiveDate: value.PrivacyEffectiveDate, TermsEffectiveDate: value.TermsEffectiveDate}
}

func brandingUpdate(input *lpbsv1.UpdateBrandingRequest) *experimentation.BrandingUpdateRequest {
	port := func(v *int32) *int {
		if v == nil {
			return nil
		}
		result := int(*v)
		return &result
	}
	return &experimentation.BrandingUpdateRequest{SiteName: input.SiteName, Tagline: input.Tagline, LogoURL: input.LogoUrl, LogoIconURL: input.LogoIconUrl, FaviconURL: input.FaviconUrl, AppleTouchIconURL: input.AppleTouchIconUrl, DefaultTitle: input.DefaultTitle, DefaultDescription: input.DefaultDescription, DefaultOGImageURL: input.DefaultOgImageUrl, ThemePrimaryColor: input.ThemePrimaryColor, ThemeBackgroundColor: input.ThemeBackgroundColor, CanonicalBaseURL: input.CanonicalBaseUrl, GoogleSiteVerification: input.GoogleSiteVerification, RobotsTxt: input.RobotsTxt, SupportChatURL: input.SupportChatUrl, SupportEmail: input.SupportEmail, SMTPHost: input.SmtpHost, SMTPPort: port(input.SmtpPort), SMTPUsername: input.SmtpUsername, SMTPPassword: input.SmtpPassword, SMTPFrom: input.SmtpFrom, ComingSoonEnabled: input.ComingSoonEnabled, ComingSoonMessage: input.ComingSoonMessage, LegalName: input.LegalName, ContactAddress: input.ContactAddress, PrivacyPolicyMarkdown: input.PrivacyPolicyMarkdown, TermsMarkdown: input.TermsMarkdown, PrivacyEffectiveDate: input.PrivacyEffectiveDate, TermsEffectiveDate: input.TermsEffectiveDate}
}

func (h BrandingConnectHandler) GetBranding(context.Context, *connect.Request[lpbsv1.GetBrandingRequest]) (*connect.Response[lpbsv1.BrandingResponse], error) {
	return connect.NewResponse(&lpbsv1.BrandingResponse{Branding: BrandingProto(h.store.GetBranding())}), nil
}

func (h BrandingConnectHandler) UpdateBranding(_ context.Context, request *connect.Request[lpbsv1.UpdateBrandingRequest]) (*connect.Response[lpbsv1.BrandingResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("branding update request is required"))
	}
	input := request.Msg
	if err := validateBusinessIdentity(input); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if input.SmtpPassword != nil {
		var err error
		if strings.TrimSpace(input.GetSmtpPassword()) == "" {
			err = administration.DeleteAuthorityCredential("SMTP_PASSWORD")
		} else {
			err = administration.PutAuthorityCredential("SMTP_PASSWORD", input.GetSmtpPassword())
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("store SMTP password in credential authority: %w", err))
		}
	}
	update := brandingUpdate(input)
	// The protected value is owned by the authority, never by ConfigStore.
	update.SMTPPassword = nil
	updated, err := h.store.UpdateBranding(update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update branding: %w", err))
	}
	return connect.NewResponse(&lpbsv1.BrandingResponse{Branding: BrandingProto(updated)}), nil
}

func (h BrandingConnectHandler) ClearBrandingField(_ context.Context, request *connect.Request[lpbsv1.ClearBrandingFieldRequest]) (*connect.Response[lpbsv1.BrandingResponse], error) {
	if request.Msg.GetField() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("field is required"))
	}
	if request.Msg.GetField() == "smtp_password" {
		if err := administration.DeleteAuthorityCredential("SMTP_PASSWORD"); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("clear SMTP password from credential authority: %w", err))
		}
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
	return connect.NewResponse(&lpbsv1.PublicBrandingResponse{Branding: &lpbsv1.PublicBranding{SiteName: b.SiteName, Tagline: derefString(b.Tagline), LogoUrl: derefString(b.LogoURL), LogoIconUrl: derefString(b.LogoIconURL), FaviconUrl: derefString(b.FaviconURL), ThemePrimaryColor: derefString(b.ThemePrimaryColor), ThemeBackgroundColor: derefString(b.ThemeBackgroundColor), SupportChatUrl: derefString(b.SupportChatURL), ComingSoonEnabled: derefBool(b.ComingSoonEnabled), ComingSoonMessage: derefString(b.ComingSoonMessage), CanonicalBaseUrl: derefString(b.CanonicalBaseURL), SupportEmail: derefString(b.SupportEmail), LegalName: derefString(b.LegalName), ContactAddress: derefString(b.ContactAddress), PrivacyPolicyMarkdown: derefString(b.PrivacyPolicyMarkdown), TermsMarkdown: derefString(b.TermsMarkdown), PrivacyEffectiveDate: derefString(b.PrivacyEffectiveDate), TermsEffectiveDate: derefString(b.TermsEffectiveDate)}}), nil
}

// maxLegalDocumentBytes bounds operator-authored legal Markdown so a public
// branding read can never balloon into an unbounded payload.
const maxLegalDocumentBytes = 100_000

// validateBusinessIdentity rejects malformed public identity fields before they
// reach the footer and legal pages every visitor sees. Empty strings are allowed
// and mean "not configured".
func validateBusinessIdentity(input *lpbsv1.UpdateBrandingRequest) error {
	for name, value := range map[string]*string{"privacy_effective_date": input.PrivacyEffectiveDate, "terms_effective_date": input.TermsEffectiveDate} {
		if value == nil || strings.TrimSpace(*value) == "" {
			continue
		}
		if _, err := time.Parse(time.DateOnly, strings.TrimSpace(*value)); err != nil {
			return fmt.Errorf("%s must be a date in YYYY-MM-DD form", name)
		}
	}
	if value := input.SupportEmail; value != nil && strings.TrimSpace(*value) != "" {
		if _, err := mail.ParseAddress(strings.TrimSpace(*value)); err != nil || strings.ContainsAny(strings.TrimSpace(*value), " <>") {
			return fmt.Errorf("support_email must be a plain email address")
		}
	}
	for name, value := range map[string]*string{"privacy_policy_markdown": input.PrivacyPolicyMarkdown, "terms_markdown": input.TermsMarkdown} {
		if value != nil && len(*value) > maxLegalDocumentBytes {
			return fmt.Errorf("%s exceeds %d bytes", name, maxLegalDocumentBytes)
		}
	}
	return nil
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
