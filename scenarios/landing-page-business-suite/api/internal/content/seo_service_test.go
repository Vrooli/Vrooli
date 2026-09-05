package content

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/contracts"
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

func (s fakeSEOStore) UpdateVariantSEO(slug string, config contracts.VariantSEOConfig) error {
	variant, ok := s.variants[slug]
	if !ok {
		return errors.New("variant not found")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	variant.SEOConfig = encoded
	s.variants[slug] = variant
	return nil
}

func TestSEOServiceAppliesBrandingAndVariantOverrides(t *testing.T) {
	canonical := "https://example.test/"
	defaultTitle := "Default title"
	config, err := json.Marshal(contracts.VariantSEOConfig{Title: "Variant title", CanonicalPath: "/variant"})
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

func TestSEOServiceUpdatesVariantSEOThroughStoreSeam(t *testing.T) {
	store := fakeSEOStore{variants: map[string]SEOVariant{"variant": {Slug: "variant"}}}
	service := NewSEOService(store, nil)
	if err := service.UpdateVariantSEO("variant", contracts.VariantSEOConfig{Title: "Updated title"}); err != nil {
		t.Fatal(err)
	}
	if got := string(store.variants["variant"].SEOConfig); !strings.Contains(got, "Updated title") {
		t.Fatalf("stored SEO config = %s, want updated title", got)
	}
}

func TestSEOServiceSitemapExcludesNoIndexVariants(t *testing.T) {
	visible, _ := json.Marshal(contracts.VariantSEOConfig{CanonicalPath: "/visible"})
	hidden, _ := json.Marshal(contracts.VariantSEOConfig{CanonicalPath: "/hidden", NoIndex: true})
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
