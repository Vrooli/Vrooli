package content

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type fakeSEOStore struct {
	branding SEOBranding
	variants map[string]SEOVariant
}

func (s fakeSEOStore) Branding() SEOBranding { return s.branding }

func (s fakeSEOStore) Variant(slug string) (SEOVariant, error) {
	variant, ok := s.variants[slug]
	if !ok {
		return SEOVariant{}, errors.New("variant not found")
	}
	return variant, nil
}

func (s fakeSEOStore) Variants() []SEOVariant {
	result := make([]SEOVariant, 0, len(s.variants))
	for _, variant := range s.variants {
		result = append(result, variant)
	}
	return result
}

func TestSEOServiceAppliesBrandingAndVariantOverrides(t *testing.T) {
	canonical := "https://example.test/"
	defaultTitle := "Default title"
	config, err := json.Marshal(VariantSEOConfig{Title: "Variant title", CanonicalPath: "/variant"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewSEOService(fakeSEOStore{
		branding: SEOBranding{SiteName: "Example", DefaultTitle: &defaultTitle, CanonicalBaseURL: &canonical},
		variants: map[string]SEOVariant{"variant": {Slug: "variant", SEOConfig: config}},
	}, nil)

	response, err := service.VariantSEO("variant")
	if err != nil {
		t.Fatal(err)
	}
	if response.Title != "Variant title" || response.CanonicalURL != "https://example.test/variant" {
		t.Fatalf("unexpected SEO response: %#v", response)
	}
}

func TestSEOServiceSitemapExcludesNoIndexVariants(t *testing.T) {
	visible, _ := json.Marshal(VariantSEOConfig{CanonicalPath: "/visible"})
	hidden, _ := json.Marshal(VariantSEOConfig{CanonicalPath: "/hidden", NoIndex: true})
	service := NewSEOService(fakeSEOStore{
		variants: map[string]SEOVariant{
			"visible": {Slug: "visible", SEOConfig: visible},
			"hidden":  {Slug: "hidden", SEOConfig: hidden},
		},
	}, nil)

	sitemap, err := service.SitemapXML("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sitemap, "https://example.test/visible") || strings.Contains(sitemap, "https://example.test/hidden") {
		t.Fatalf("sitemap visibility mismatch: %s", sitemap)
	}
}
