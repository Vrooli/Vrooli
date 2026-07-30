package content

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
	contentdomain "landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/contracts"
)

// SEOConnectHandler adapts the SEO domain to generated Connect procedures.
// Sitemap and robots remain crawler-document HTTP endpoints rather than RPCs.
type SEOConnectHandler struct {
	service *contentdomain.SEOService
	now     func() time.Time
}

func NewSEOConnectHandler(service *contentdomain.SEOService) *SEOConnectHandler {
	return &SEOConnectHandler{service: service, now: time.Now}
}

func (h *SEOConnectHandler) GetVariantSEO(_ context.Context, request *connect.Request[lpbsv1.GetVariantSEORequest]) (*connect.Response[lpbsv1.SEOResponse], error) {
	slug := strings.TrimSpace(request.Msg.GetSlug())
	if slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("variant slug required"))
	}
	seo, err := h.service.VariantSEO(slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("variant %q not found", slug))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load variant SEO: %w", err))
	}
	response, err := SEOResponseProto(seo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode variant SEO: %w", err))
	}
	return connect.NewResponse(response), nil
}

func (h *SEOConnectHandler) UpdateVariantSEO(_ context.Context, request *connect.Request[lpbsv1.UpdateVariantSEORequest]) (*connect.Response[lpbsv1.UpdateVariantSEOResponse], error) {
	slug := strings.TrimSpace(request.Msg.GetSlug())
	if slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("variant slug required"))
	}
	if err := h.service.UpdateVariantSEO(slug, variantSEOConfigFromProto(request.Msg.GetConfig())); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("variant %q not found", slug))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save variant SEO: %w", err))
	}
	return connect.NewResponse(&lpbsv1.UpdateVariantSEOResponse{Success: true, UpdatedAt: h.now().UTC().Format(time.RFC3339)}), nil
}

// SEOResponseProto maps domain SEO data to its generated public contract.
func SEOResponseProto(response *contentdomain.SEOResponse) (*lpbsv1.SEOResponse, error) {
	result := &lpbsv1.SEOResponse{SiteName: response.SiteName, Title: response.Title, Description: response.Description, OgTitle: response.OGTitle, OgDescription: response.OGDescription, OgImageUrl: response.OGImageURL, TwitterCard: response.TwitterCard, CanonicalUrl: response.CanonicalURL, FaviconUrl: response.FaviconURL, AppleTouchIconUrl: response.AppleTouchIconURL, ThemePrimaryColor: response.ThemePrimaryColor, Noindex: response.NoIndex}
	if response.StructuredData == nil || len(*response.StructuredData) == 0 {
		return result, nil
	}
	var values map[string]any
	if err := json.Unmarshal(*response.StructuredData, &values); err != nil {
		return nil, fmt.Errorf("decode structured data: %w", err)
	}
	structuredData, err := structpb.NewStruct(values)
	if err != nil {
		return nil, fmt.Errorf("build structured data: %w", err)
	}
	result.StructuredData = structuredData
	return result, nil
}

func variantSEOConfigFromProto(config *sharedv1.VariantSEOConfig) contracts.VariantSEOConfig {
	if config == nil {
		return contracts.VariantSEOConfig{}
	}
	result := contracts.VariantSEOConfig{Title: config.GetTitle(), Description: config.GetDescription(), OGTitle: config.GetOgTitle(), OGDescription: config.GetOgDescription(), OGImageURL: config.GetOgImageUrl(), TwitterCard: config.GetTwitterCard(), CanonicalPath: config.GetCanonicalPath(), NoIndex: config.GetNoindex()}
	if structuredData := config.GetStructuredData(); structuredData != nil {
		result.StructuredData = structuredData.AsMap()
	}
	return result
}

// RegisterSEOConnectRoutes mounts public reads and administrator updates.
func RegisterSEOConnectRoutes(router *mux.Router, service *contentdomain.SEOService, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewSeoServiceHandler(NewSEOConnectHandler(service))
	router.Handle(lpbsconnect.SeoServiceGetVariantSEOProcedure, generated).Methods(http.MethodPost)
	router.Handle(lpbsconnect.SeoServiceUpdateVariantSEOProcedure, requireAdmin(generated.ServeHTTP)).Methods(http.MethodPost)
}

var _ lpbsconnect.SeoServiceHandler = (*SEOConnectHandler)(nil)
