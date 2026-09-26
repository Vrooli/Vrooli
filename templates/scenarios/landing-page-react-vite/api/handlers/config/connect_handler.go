package config

import (
	"context"
	"landing-page-react-vite-api/internal/content"
	"landing-page-react-vite-api/internal/download"
	"landing-page-react-vite-api/internal/landingconfig"
	"landing-page-react-vite-api/internal/variant"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// Deps wires the LandingConfig Connect handler.
type Deps struct {
	Service *landingconfig.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the LandingConfigService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetLandingConfig(ctx context.Context, req *connect.Request[landingv1.GetLandingConfigRequest]) (*connect.Response[landingv1.LandingConfigResponse], error) {
	cfg, err := h.deps.Service.GetLandingConfig(ctx, req.Msg.VariantSlug)
	if err != nil {
		h.deps.Logger.Printf("config.GetLandingConfig: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(configToProto(cfg)), nil
}

func configToProto(c *landingconfig.Config) *landingv1.LandingConfigResponse {
	out := &landingv1.LandingConfigResponse{
		Variant: &landingv1.LandingVariantSummary{
			Id:          int64(c.Variant.ID),
			Slug:        c.Variant.Slug,
			Name:        c.Variant.Name,
			Description: c.Variant.Description,
			Axes:        c.Variant.Axes,
		},
		Pricing:  c.Pricing, // plan.PricingOverview aliases landingv1.PricingOverview
		Header:   headerToProto(c.Header),
		Fallback: c.Fallback,
	}
	for i := range c.Sections {
		out.Sections = append(out.Sections, sectionToProto(&c.Sections[i]))
	}
	for i := range c.Downloads {
		out.Downloads = append(out.Downloads, appToProto(&c.Downloads[i]))
	}
	if c.Branding != nil {
		out.Branding = &landingv1.LandingBranding{
			SiteName:             c.Branding.SiteName,
			Tagline:              c.Branding.Tagline,
			LogoUrl:              c.Branding.LogoURL,
			LogoIconUrl:          c.Branding.LogoIconURL,
			FaviconUrl:           c.Branding.FaviconURL,
			ThemePrimaryColor:    c.Branding.ThemePrimaryColor,
			ThemeBackgroundColor: c.Branding.ThemeBackgroundColor,
		}
	}
	return out
}

func sectionToProto(s *content.Section) *landingv1.LandingSection {
	out := &landingv1.LandingSection{
		SectionType: s.SectionType,
		Order:       int32(s.Order),
		Enabled:     s.Enabled,
	}
	if len(s.Content) > 0 {
		if st, err := structpb.NewStruct(s.Content); err == nil {
			out.Content = st
		}
	}
	return out
}

func headerToProto(h variant.LandingHeaderConfig) *landingv1.LandingHeaderConfig {
	return &landingv1.LandingHeaderConfig{
		Branding: &landingv1.HeaderBrandingConfig{
			Mode:             h.Branding.Mode,
			Label:            h.Branding.Label,
			Subtitle:         h.Branding.Subtitle,
			MobilePreference: h.Branding.MobilePreference,
		},
		Nav:      &landingv1.HeaderNavConfig{Links: navLinksToProto(h.Nav.Links)},
		Ctas:     &landingv1.HeaderCTAGroup{Primary: ctaToProto(h.Ctas.Primary), Secondary: ctaToProto(h.Ctas.Secondary)},
		Behavior: &landingv1.HeaderBehaviorConfig{Sticky: h.Behavior.Sticky, HideOnScroll: h.Behavior.HideOnScroll},
	}
}

func navLinksToProto(links []variant.HeaderNavLink) []*landingv1.HeaderNavLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]*landingv1.HeaderNavLink, 0, len(links))
	for i := range links {
		l := links[i]
		node := &landingv1.HeaderNavLink{
			Id:          l.ID,
			Type:        l.Type,
			Label:       l.Label,
			SectionType: l.SectionType,
			Anchor:      l.Anchor,
			Href:        l.Href,
			VisibleOn:   &landingv1.HeaderVisibilityConfig{Desktop: l.VisibleOn.Desktop, Mobile: l.VisibleOn.Mobile},
			Children:    navLinksToProto(l.Children),
		}
		if l.SectionID != nil {
			id := int32(*l.SectionID)
			node.SectionId = &id
		}
		out = append(out, node)
	}
	return out
}

func ctaToProto(c variant.HeaderCTAConfig) *landingv1.HeaderCTAConfig {
	return &landingv1.HeaderCTAConfig{Mode: c.Mode, Label: c.Label, Href: c.Href, Variant: c.Variant}
}

func appToProto(a *download.App) *landingv1.DownloadApp {
	out := &landingv1.DownloadApp{
		BundleKey:       a.BundleKey,
		AppKey:          a.AppKey,
		Name:            a.Name,
		Tagline:         a.Tagline,
		Description:     a.Description,
		InstallOverview: a.InstallOverview,
		InstallSteps:    a.InstallSteps,
		DisplayOrder:    int32(a.DisplayOrder),
	}
	if len(a.Metadata) > 0 {
		if st, err := structpb.NewStruct(a.Metadata); err == nil {
			out.Metadata = st
		}
	}
	for _, sf := range a.Storefronts {
		out.Storefronts = append(out.Storefronts, &landingv1.DownloadStorefront{Store: sf.Store, Label: sf.Label, Url: sf.URL, Badge: sf.Badge})
	}
	for i := range a.Platforms {
		p := a.Platforms[i]
		asset := &landingv1.DownloadAsset{
			Id:                  p.ID,
			BundleKey:           p.BundleKey,
			AppKey:              p.AppKey,
			Platform:            p.Platform,
			ArtifactUrl:         p.ArtifactURL,
			ReleaseVersion:      p.ReleaseVersion,
			ReleaseNotes:        p.ReleaseNotes,
			Checksum:            p.Checksum,
			RequiresEntitlement: p.RequiresEntitlement,
		}
		if len(p.Metadata) > 0 {
			if st, err := structpb.NewStruct(p.Metadata); err == nil {
				asset.Metadata = st
			}
		}
		out.Platforms = append(out.Platforms, asset)
	}
	return out
}
