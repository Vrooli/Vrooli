package main

import (
	"landing-page-business-suite-api/internal/content"
)

// SEOServicer is the transport-facing SEO contract.
type SEOServicer interface {
	VariantSEO(slug string) (*SEOResponse, error)
	SitemapXML(fallbackBase string) (string, error)
	RobotsTXT(fallbackBase string) (string, error)
}

type (
	VariantSEOConfig = content.VariantSEOConfig
	SEOResponse      = content.SEOResponse
	SEOService       = content.SEOService
)

var _ SEOServicer = (*SEOService)(nil)

type seoConfigStoreAdapter struct{ store *ConfigStore }

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

// NewSEOService constructs content-domain SEO policy over the JSON configuration adapter.
func NewSEOService(store *ConfigStore) *SEOService {
	return content.NewSEOService(seoConfigStoreAdapter{store: store}, logStructuredError)
}

// NewSEOServiceWithConfigStore is retained for compatibility with existing callers.
func NewSEOServiceWithConfigStore(store *ConfigStore) *SEOService { return NewSEOService(store) }
