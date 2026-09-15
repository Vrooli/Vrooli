// Package landingconfig is the read-time aggregator for the public landing
// page: it composes the selected variant, its content sections, pricing,
// downloads, header presentation, and branding into a single payload. When any
// critical lookup fails (or an active variant is misconfigured — no renderable
// hero) it fails closed to a built-in fallback payload rather than serving a
// broken page. The LandingConfig Connect handler in handlers/config adapts this
// Service; it owns no tables.
package landingconfig

import (
	"context"
	"fmt"
	"landing-page-react-vite-api/internal/branding"
	"landing-page-react-vite-api/internal/content"
	"landing-page-react-vite-api/internal/download"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/variant"
	"log"
	"sort"
)

// Service aggregates the landing payload from the domain services.
type Service struct {
	variant  *variant.Service
	content  *content.Service
	plan     *plan.Service
	download *download.Service
	branding *branding.Service
}

// NewService wires the aggregator's domain-service dependencies.
func NewService(variantSvc *variant.Service, contentSvc *content.Service, planSvc *plan.Service, downloadSvc *download.Service, brandingSvc *branding.Service) *Service {
	return &Service{variant: variantSvc, content: contentSvc, plan: planSvc, download: downloadSvc, branding: brandingSvc}
}

// VariantSummary is the compact variant descriptor the frontend renders.
type VariantSummary struct {
	ID          int
	Slug        string
	Name        string
	Description string
	Axes        map[string]string
}

// Branding is the public branding subset surfaced on the landing page.
type Branding struct {
	SiteName             string
	Tagline              *string
	LogoURL              *string
	LogoIconURL          *string
	FaviconURL           *string
	ThemePrimaryColor    *string
	ThemeBackgroundColor *string
}

// Config is the aggregated landing payload.
type Config struct {
	Variant   VariantSummary
	Sections  []content.Section
	Pricing   *plan.PricingOverview
	Downloads []download.App
	Header    variant.LandingHeaderConfig
	Branding  *Branding
	Fallback  bool
}

// GetLandingConfig assembles the landing payload for a variant (explicit slug or
// weighted selection), failing closed to the fallback payload on any critical
// error or an unrenderable variant.
func (s *Service) GetLandingConfig(ctx context.Context, variantSlug string) (*Config, error) {
	pricing, err := s.plan.GetPricingOverview()
	if err != nil {
		return s.fallback(ctx, "pricing_fetch_failed", err), nil
	}
	downloads, err := s.download.ListApps(s.plan.BundleKey())
	if err != nil {
		return s.fallback(ctx, "download_list_failed", err), nil
	}

	var v *variant.Variant
	if variantSlug != "" {
		v, err = s.variant.GetVariantBySlug(variantSlug)
	} else {
		v, err = s.variant.SelectVariant()
	}
	if err != nil || v == nil {
		reason := "weighted_selection_failed"
		if variantSlug != "" {
			reason = "variant_lookup_failed"
		}
		return s.fallback(ctx, reason, err), nil
	}

	sections, err := s.content.GetPublicSections(int64(v.ID))
	if err != nil {
		return s.fallback(ctx, "section_fetch_failed", err), nil
	}
	sort.SliceStable(sections, func(i, j int) bool { return sections[i].Order < sections[j].Order })
	if err := ensureRenderableSections(sections); err != nil {
		return s.fallback(ctx, "section_renderability_failed", err), nil
	}

	return &Config{
		Variant: VariantSummary{
			ID:          v.ID,
			Slug:        v.Slug,
			Name:        v.Name,
			Description: v.Description,
			Axes:        v.Axes,
		},
		Sections:  sections,
		Pricing:   pricing,
		Downloads: downloads,
		Header:    v.HeaderConfig,
		Branding:  s.publicBranding(ctx),
		Fallback:  false,
	}, nil
}

// ensureRenderableSections fails closed unless at least one section is enabled
// and an enabled hero section is present.
func ensureRenderableSections(sections []content.Section) error {
	var anyEnabled, heroEnabled bool
	for _, s := range sections {
		if !s.Enabled {
			continue
		}
		anyEnabled = true
		if s.SectionType == "hero" {
			heroEnabled = true
		}
	}
	if !anyEnabled {
		return fmt.Errorf("variant has no enabled sections")
	}
	if !heroEnabled {
		return fmt.Errorf("variant has no enabled hero section")
	}
	return nil
}

func (s *Service) publicBranding(ctx context.Context) *Branding {
	if s.branding == nil {
		return nil
	}
	b, err := s.branding.Get(ctx)
	if err != nil || b == nil {
		return nil
	}
	return &Branding{
		SiteName:             b.SiteName,
		Tagline:              b.Tagline,
		LogoURL:              b.LogoURL,
		LogoIconURL:          b.LogoIconURL,
		FaviconURL:           b.FaviconURL,
		ThemePrimaryColor:    b.ThemePrimaryColor,
		ThemeBackgroundColor: b.ThemeBackgroundColor,
	}
}

// fallback logs the reason and returns the built-in renderable payload.
func (s *Service) fallback(ctx context.Context, reason string, err error) *Config {
	if err != nil {
		log.Printf("landingconfig: fallback (%s): %v", reason, err)
	} else {
		log.Printf("landingconfig: fallback (%s)", reason)
	}
	cfg := fallbackConfig()
	cfg.Branding = s.publicBranding(ctx)
	return cfg
}

// fallbackConfig is the built-in, always-renderable landing payload used when
// the database has no usable variant.
func fallbackConfig() *Config {
	return &Config{
		Variant: VariantSummary{Slug: "fallback", Name: "Welcome"},
		Sections: []content.Section{
			{
				SectionType: "hero",
				Content: map[string]interface{}{
					"headline":    "Welcome",
					"subheadline": "This landing page is running on default content.",
				},
				Order:   0,
				Enabled: true,
			},
		},
		Fallback: true,
	}
}
