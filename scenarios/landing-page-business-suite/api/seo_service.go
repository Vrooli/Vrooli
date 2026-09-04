package main

import (
	"encoding/json"

	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/contracts"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
)

// SEOServicer is the transport-facing SEO contract.
type SEOServicer interface {
	VariantSEO(slug string) (*content.SEOResponse, error)
	SitemapXML(fallbackBase string) (string, error)
	RobotsTXT(fallbackBase string) (string, error)
}

var _ SEOServicer = (*content.SEOService)(nil)

type seoConfigStoreAdapter struct{ store *experimentation.ConfigStore }

func (a seoConfigStoreAdapter) Branding() content.SEOBranding {
	b := a.store.GetBranding()
	return content.SEOBranding{
		SiteName: b.SiteName, DefaultTitle: b.DefaultTitle, DefaultDescription: b.DefaultDescription,
		DefaultOGImageURL: b.DefaultOGImageURL, FaviconURL: b.FaviconURL,
		AppleTouchIconURL: b.AppleTouchIconURL, ThemePrimaryColor: b.ThemePrimaryColor,
		CanonicalBaseURL: b.CanonicalBaseURL, RobotsTxt: b.RobotsTxt,
	}
}

func (a seoConfigStoreAdapter) Variant(slug string) (content.SEOVariant, error) {
	v, err := a.store.GetVariant(slug)
	if err != nil {
		return content.SEOVariant{}, err
	}
	return content.SEOVariant{Slug: v.Variant.Slug, SEOConfig: v.Variant.SEOConfig}, nil
}

func (a seoConfigStoreAdapter) Variants() []content.SEOVariant {
	variants := a.store.ListVariants()
	result := make([]content.SEOVariant, 0, len(variants))
	for _, variant := range variants {
		result = append(result, content.SEOVariant{Slug: variant.Variant.Slug, SEOConfig: variant.Variant.SEOConfig})
	}
	return result
}

func (a seoConfigStoreAdapter) UpdateVariantSEO(slug string, config contracts.VariantSEOConfig) error {
	variant, err := a.store.GetVariant(slug)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	variant.Variant.SEOConfig = encoded
	return a.store.SaveVariant(slug, variant)
}

// NewSEOService constructs content-domain SEO policy over the JSON configuration adapter.
func NewSEOService(store *experimentation.ConfigStore) *content.SEOService {
	return content.NewSEOService(seoConfigStoreAdapter{store: store}, logx.Error)
}
