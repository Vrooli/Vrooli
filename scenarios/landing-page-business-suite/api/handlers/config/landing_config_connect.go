// Package landing owns the generated public landing-configuration transport.
package landing

import (
	"context"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	"landing-page-business-suite-api/internal/landing"
)

// LandingConfigConnectHandler exposes the config.proto public landing payload via
// its generated contract without depending on API-root composition.
type LandingConfigConnectHandler struct{ service *landing.LandingConfigService }

func NewLandingConfigConnectHandler(service *landing.LandingConfigService) LandingConfigConnectHandler {
	return LandingConfigConnectHandler{service: service}
}

func (h LandingConfigConnectHandler) GetLandingConfig(ctx context.Context, request *connect.Request[lpbsv1.GetLandingConfigRequest]) (*connect.Response[lpbsv1.LandingConfigResponse], error) {
	response, err := h.service.GetLandingConfig(ctx, request.Msg.GetVariantSlug(), request.Msg.GetVisitorId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load landing configuration: %w", err))
	}
	message, err := LandingConfigProto(response)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode landing configuration: %w", err))
	}
	return connect.NewResponse(message), nil
}

// LandingConfigProto converts domain payloads to their generated public wire contract.
func LandingConfigProto(response *landing.LandingConfigResponse) (*lpbsv1.LandingConfigResponse, error) {
	sections := make([]*lpbsv1.LandingSection, 0, len(response.Sections))
	for index, section := range response.Sections {
		content, err := structpb.NewStruct(section.Content)
		if err != nil {
			return nil, fmt.Errorf("section %d (%q) content: %w", index, section.SectionType, err)
		}
		order, err := LandingConfigInt32(section.Order, fmt.Sprintf("section %d (%q) order", index, section.SectionType))
		if err != nil {
			return nil, err
		}
		sections = append(sections, &lpbsv1.LandingSection{SectionKey: section.Key, SectionType: section.SectionType, Content: content, Order: order, Enabled: section.Enabled})
	}
	downloads, err := landing.ProtoDownloads(response.Downloads)
	if err != nil {
		return nil, err
	}
	header, err := varianthttp.HeaderProto(response.Header)
	if err != nil {
		return nil, err
	}
	offers, err := introOffersProto(response.IntroOffers)
	if err != nil {
		return nil, err
	}
	return &lpbsv1.LandingConfigResponse{Variant: &lpbsv1.LandingVariantSummary{Id: int64(response.Variant.ID), Slug: response.Variant.Slug, Name: response.Variant.Name, Description: response.Variant.Description, Axes: response.Variant.Axes}, Sections: sections, Pricing: response.Pricing, Downloads: downloads, Header: header, Branding: brandingProto(response.Branding), Fallback: response.Fallback, CouponMappings: response.CouponMappings, IntroOffers: offers}, nil
}

func brandingProto(branding *landing.LandingBranding) *lpbsv1.LandingBranding {
	if branding == nil {
		return nil
	}
	return &lpbsv1.LandingBranding{SiteName: branding.SiteName, Tagline: branding.Tagline, LogoUrl: branding.LogoURL, LogoIconUrl: branding.LogoIconURL, FaviconUrl: branding.FaviconURL, ThemePrimaryColor: branding.ThemePrimaryColor, ThemeBackgroundColor: branding.ThemeBackgroundColor, SupportChatUrl: branding.SupportChatURL, SupportEmail: branding.SupportEmail, ComingSoonEnabled: branding.ComingSoonEnabled, ComingSoonMessage: branding.ComingSoonMessage}
}

func introOffersProto(offers []landing.IntroOffer) ([]*lpbsv1.IntroOffer, error) {
	result := make([]*lpbsv1.IntroOffer, 0, len(offers))
	for _, offer := range offers {
		var months, max *int32
		if offer.DurationInMonths != nil {
			value, err := LandingConfigInt32(*offer.DurationInMonths, fmt.Sprintf("intro offer %q duration in months", offer.ID))
			if err != nil {
				return nil, err
			}
			months = &value
		}
		if offer.MaxRedemptions != nil {
			value, err := LandingConfigInt32(*offer.MaxRedemptions, fmt.Sprintf("intro offer %q max redemptions", offer.ID))
			if err != nil {
				return nil, err
			}
			max = &value
		}
		times, err := LandingConfigInt32(offer.TimesRedeemed, fmt.Sprintf("intro offer %q times redeemed", offer.ID))
		if err != nil {
			return nil, err
		}
		amountOff, percentOff, redeemBy := offer.AmountOff, offer.PercentOff, offer.RedeemBy
		result = append(result, &lpbsv1.IntroOffer{Id: offer.ID, Name: offer.Name, AmountOff: &amountOff, PercentOff: &percentOff, Currency: offer.Currency, Duration: offer.Duration, DurationInMonths: months, MaxRedemptions: max, RedeemBy: &redeemBy, TimesRedeemed: times, Valid: offer.Valid, Created: offer.Created, IsIntroCoupon: offer.IsIntroCoupon, IntroTier: offer.IntroTier})
	}
	return result, nil
}

// LandingConfigInt32 bounds-checks a domain integer before protobuf encoding.
func LandingConfigInt32(value int, field string) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%s %d is outside the protobuf int32 range", field, value)
	}
	return int32(value), nil
}

func RegisterLandingConfigConnectRoutes(router *mux.Router, service *landing.LandingConfigService) {
	path, handler := lpbsconnect.NewLandingConfigServiceHandler(NewLandingConfigConnectHandler(service))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

var _ lpbsconnect.LandingConfigServiceHandler = LandingConfigConnectHandler{}
