package main

import (
	"context"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
)

// landingConfigConnectHandler exposes the complete public landing payload via
// its generated contract.
type landingConfigConnectHandler struct {
	service *LandingConfigService
}

func newLandingConfigConnectHandler(service *LandingConfigService) *landingConfigConnectHandler {
	return &landingConfigConnectHandler{service: service}
}

func (h *landingConfigConnectHandler) GetLandingConfig(ctx context.Context, request *connect.Request[lpbsv1.GetLandingConfigRequest]) (*connect.Response[lpbsv1.LandingConfigResponse], error) {
	response, err := h.service.GetLandingConfig(ctx, request.Msg.GetVariantSlug())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load landing configuration: %w", err))
	}
	message, err := landingConfigProto(response)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode landing configuration: %w", err))
	}
	return connect.NewResponse(message), nil
}

func landingConfigProto(response *LandingConfigResponse) (*lpbsv1.LandingConfigResponse, error) {
	sections := make([]*lpbsv1.LandingSection, 0, len(response.Sections))
	for index, section := range response.Sections {
		content, err := structpb.NewStruct(section.Content)
		if err != nil {
			return nil, fmt.Errorf("section %d (%q) content: %w", index, section.SectionType, err)
		}
		order, err := landingInt32(section.Order, fmt.Sprintf("section %d (%q) order", index, section.SectionType))
		if err != nil {
			return nil, err
		}
		sections = append(sections, &lpbsv1.LandingSection{
			SectionType: section.SectionType,
			Content:     content,
			Order:       order,
			Enabled:     section.Enabled,
		})
	}

	downloads, err := landingDownloadsProto(response.Downloads)
	if err != nil {
		return nil, err
	}
	header, err := landingHeaderProto(response.Header)
	if err != nil {
		return nil, err
	}
	introOffers, err := introOffersProto(response.IntroOffers)
	if err != nil {
		return nil, err
	}

	return &lpbsv1.LandingConfigResponse{
		Variant: &lpbsv1.LandingVariantSummary{
			Id:          int64(response.Variant.ID),
			Slug:        response.Variant.Slug,
			Name:        response.Variant.Name,
			Description: response.Variant.Description,
			Axes:        response.Variant.Axes,
		},
		Sections:       sections,
		Pricing:        response.Pricing,
		Downloads:      downloads,
		Header:         header,
		Branding:       landingBrandingProto(response.Branding),
		Fallback:       response.Fallback,
		CouponMappings: response.CouponMappings,
		IntroOffers:    introOffers,
	}, nil
}

func landingDownloadsProto(downloads []DownloadApp) ([]*sharedv1.DownloadApp, error) {
	result := make([]*sharedv1.DownloadApp, 0, len(downloads))
	for index, app := range downloads {
		metadata, err := structpb.NewStruct(app.Metadata)
		if err != nil {
			return nil, fmt.Errorf("download %d (%q) metadata: %w", index, app.AppKey, err)
		}
		updatePolicy, err := structpb.NewStruct(app.UpdatePolicy)
		if err != nil {
			return nil, fmt.Errorf("download %d (%q) update policy: %w", index, app.AppKey, err)
		}
		platforms, err := landingAssetsProto(app.Platforms)
		if err != nil {
			return nil, fmt.Errorf("download %d (%q): %w", index, app.AppKey, err)
		}
		storefronts := make([]*sharedv1.DownloadStorefront, 0, len(app.Storefronts))
		for _, storefront := range app.Storefronts {
			storefronts = append(storefronts, &sharedv1.DownloadStorefront{Store: storefront.Store, Label: storefront.Label, Url: storefront.URL, Badge: storefront.Badge})
		}
		displayOrder, err := landingInt32(app.DisplayOrder, fmt.Sprintf("download %d (%q) display order", index, app.AppKey))
		if err != nil {
			return nil, err
		}
		result = append(result, &sharedv1.DownloadApp{
			Id: app.ID, BundleKey: app.BundleKey, AppKey: app.AppKey, Name: app.Name, Tagline: app.Tagline,
			Description: app.Description, IconUrl: app.IconURL, ScreenshotUrl: app.ScreenshotURL,
			InstallOverview: app.InstallOverview, InstallSteps: app.InstallSteps, Storefronts: storefronts,
			Metadata: metadata, DisplayOrder: displayOrder, UpdateApiKey: app.UpdateAPIKey,
			UpdatePolicy: updatePolicy, Platforms: platforms,
		})
	}
	return result, nil
}

func landingAssetsProto(assets []DownloadAsset) ([]*sharedv1.DownloadAsset, error) {
	result := make([]*sharedv1.DownloadAsset, 0, len(assets))
	for index, asset := range assets {
		metadata, err := structpb.NewStruct(asset.Metadata)
		if err != nil {
			return nil, fmt.Errorf("asset %d (%q) metadata: %w", index, asset.Platform, err)
		}
		artifactCount, err := landingInt32(asset.ArtifactCount, fmt.Sprintf("asset %d (%q) artifact count", index, asset.Platform))
		if err != nil {
			return nil, err
		}
		result = append(result, &sharedv1.DownloadAsset{
			Id: asset.ID, BundleKey: asset.BundleKey, AppKey: asset.AppKey, Platform: asset.Platform,
			ArtifactUrl: asset.ArtifactURL, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes,
			Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement, Metadata: metadata,
			ArtifactSource: asset.ArtifactSource, ArtifactId: asset.ArtifactID, VariantKey: asset.VariantKey,
			ArtifactFilename: asset.ArtifactFilename, ArtifactSizeBytes: asset.ArtifactSizeBytes, ArtifactCount: artifactCount,
		})
	}
	return result, nil
}

func landingHeaderProto(header LandingHeaderConfig) (*sharedv1.LandingHeaderConfig, error) {
	links, err := landingHeaderLinksProto(header.Nav.Links)
	if err != nil {
		return nil, err
	}
	return &sharedv1.LandingHeaderConfig{
		Branding: &sharedv1.HeaderBrandingConfig{Mode: header.Branding.Mode, Label: header.Branding.Label, Subtitle: header.Branding.Subtitle, MobilePreference: header.Branding.MobilePreference},
		Nav:      &sharedv1.HeaderNavConfig{Links: links},
		Ctas: &sharedv1.HeaderCTAGroup{
			Primary:   &sharedv1.HeaderCTAConfig{Mode: header.Ctas.Primary.Mode, Label: header.Ctas.Primary.Label, Href: header.Ctas.Primary.Href, Variant: header.Ctas.Primary.Variant},
			Secondary: &sharedv1.HeaderCTAConfig{Mode: header.Ctas.Secondary.Mode, Label: header.Ctas.Secondary.Label, Href: header.Ctas.Secondary.Href, Variant: header.Ctas.Secondary.Variant},
		},
		Behavior: &sharedv1.HeaderBehaviorConfig{Sticky: header.Behavior.Sticky, HideOnScroll: header.Behavior.HideOnScroll},
	}, nil
}

func landingHeaderLinksProto(links []HeaderNavLink) ([]*sharedv1.HeaderNavLink, error) {
	result := make([]*sharedv1.HeaderNavLink, 0, len(links))
	for _, link := range links {
		var sectionID *int32
		if link.SectionID != nil {
			value, err := landingInt32(*link.SectionID, fmt.Sprintf("header link %q section ID", link.ID))
			if err != nil {
				return nil, err
			}
			sectionID = &value
		}
		children, err := landingHeaderLinksProto(link.Children)
		if err != nil {
			return nil, err
		}
		result = append(result, &sharedv1.HeaderNavLink{
			Id: link.ID, Type: link.Type, Label: link.Label, SectionType: link.SectionType, SectionId: sectionID,
			Anchor: link.Anchor, Href: link.Href, VisibleOn: &sharedv1.HeaderVisibilityConfig{Desktop: link.VisibleOn.Desktop, Mobile: link.VisibleOn.Mobile},
			Children: children,
		})
	}
	return result, nil
}

func landingBrandingProto(branding *LandingBranding) *lpbsv1.LandingBranding {
	if branding == nil {
		return nil
	}
	return &lpbsv1.LandingBranding{SiteName: branding.SiteName, Tagline: branding.Tagline, LogoUrl: branding.LogoURL, LogoIconUrl: branding.LogoIconURL, FaviconUrl: branding.FaviconURL, ThemePrimaryColor: branding.ThemePrimaryColor, ThemeBackgroundColor: branding.ThemeBackgroundColor, SupportChatUrl: branding.SupportChatURL, SupportEmail: branding.SupportEmail, ComingSoonEnabled: branding.ComingSoonEnabled, ComingSoonMessage: branding.ComingSoonMessage}
}

func introOffersProto(offers []StripeCoupon) ([]*lpbsv1.IntroOffer, error) {
	result := make([]*lpbsv1.IntroOffer, 0, len(offers))
	for _, offer := range offers {
		var durationInMonths, maxRedemptions *int32
		if offer.DurationInMonths != nil {
			value, err := landingInt32(*offer.DurationInMonths, fmt.Sprintf("intro offer %q duration in months", offer.ID))
			if err != nil {
				return nil, err
			}
			durationInMonths = &value
		}
		if offer.MaxRedemptions != nil {
			value, err := landingInt32(*offer.MaxRedemptions, fmt.Sprintf("intro offer %q max redemptions", offer.ID))
			if err != nil {
				return nil, err
			}
			maxRedemptions = &value
		}
		timesRedeemed, err := landingInt32(offer.TimesRedeemed, fmt.Sprintf("intro offer %q times redeemed", offer.ID))
		if err != nil {
			return nil, err
		}
		name, currency, introTier := offer.Name, offer.Currency, offer.IntroTier
		result = append(result, &lpbsv1.IntroOffer{Id: offer.ID, Name: &name, AmountOff: offer.AmountOff, PercentOff: offer.PercentOff, Currency: &currency, Duration: offer.Duration, DurationInMonths: durationInMonths, MaxRedemptions: maxRedemptions, RedeemBy: offer.RedeemBy, TimesRedeemed: timesRedeemed, Valid: offer.Valid, Created: offer.Created, IsIntroCoupon: offer.IsIntroCoupon, IntroTier: &introTier})
	}
	return result, nil
}

func landingInt32(value int, field string) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%s %d is outside the protobuf int32 range", field, value)
	}
	return int32(value), nil
}

func registerLandingConfigConnectRoutes(router *mux.Router, service *LandingConfigService) {
	path, handler := lpbsconnect.NewLandingConfigServiceHandler(newLandingConfigConnectHandler(service))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
