package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"landing-page-business-suite-api/internal/contracts"
)

// seam: SEOStore supplies only the content data needed to render SEO documents.
// Keeping this boundary narrow prevents the content domain from depending on
// the root application's JSON-backed configuration implementation.
type SEOStore interface {
	Branding() SEOBranding
	Variant(slug string) (SEOVariant, error)
	Variants() []SEOVariant
	UpdateVariantSEO(slug string, config contracts.VariantSEOConfig) error
}

type SEOBranding struct {
	SiteName           string
	DefaultTitle       *string
	DefaultDescription *string
	DefaultOGImageURL  *string
	FaviconURL         *string
	AppleTouchIconURL  *string
	ThemePrimaryColor  *string
	CanonicalBaseURL   *string
	RobotsTxt          *string
}

type SEOVariant struct {
	Slug      string
	SEOConfig json.RawMessage
}

type SEOResponse struct {
	SiteName          string           `json:"site_name"`
	Title             string           `json:"title"`
	Description       string           `json:"description"`
	OGTitle           string           `json:"og_title"`
	OGDescription     string           `json:"og_description"`
	OGImageURL        string           `json:"og_image_url,omitempty"`
	TwitterCard       string           `json:"twitter_card"`
	CanonicalURL      string           `json:"canonical_url,omitempty"`
	FaviconURL        string           `json:"favicon_url,omitempty"`
	AppleTouchIconURL string           `json:"apple_touch_icon_url,omitempty"`
	ThemePrimaryColor string           `json:"theme_primary_color,omitempty"`
	NoIndex           bool             `json:"noindex"`
	StructuredData    *json.RawMessage `json:"structured_data,omitempty"`
}

type SEOService struct {
	store SEOStore
	logf  func(string, map[string]interface{})
}

func NewSEOService(store SEOStore, logf func(string, map[string]interface{})) *SEOService {
	if logf == nil {
		logf = func(string, map[string]interface{}) {}
	}
	return &SEOService{store: store, logf: logf}
}

func (s *SEOService) VariantSEO(slug string) (*SEOResponse, error) {
	branding := s.store.Branding()
	variant, err := s.store.Variant(slug)
	if err != nil {
		return nil, err
	}

	var variantSEO contracts.VariantSEOConfig
	if len(variant.SEOConfig) > 0 {
		if err := json.Unmarshal(variant.SEOConfig, &variantSEO); err != nil {
			s.logf("parse_variant_seo_failed", map[string]interface{}{"slug": slug, "error": err.Error()})
		}
	}

	response := SEOResponse{
		SiteName:          branding.SiteName,
		Title:             coalesce(variantSEO.Title, ptrString(branding.DefaultTitle), branding.SiteName),
		Description:       coalesce(variantSEO.Description, ptrString(branding.DefaultDescription), ""),
		OGTitle:           coalesce(variantSEO.OGTitle, variantSEO.Title, ptrString(branding.DefaultTitle), branding.SiteName),
		OGDescription:     coalesce(variantSEO.OGDescription, variantSEO.Description, ptrString(branding.DefaultDescription), ""),
		OGImageURL:        coalesce(variantSEO.OGImageURL, ptrString(branding.DefaultOGImageURL)),
		TwitterCard:       coalesce(variantSEO.TwitterCard, "summary_large_image"),
		FaviconURL:        ptrString(branding.FaviconURL),
		AppleTouchIconURL: ptrString(branding.AppleTouchIconURL),
		ThemePrimaryColor: ptrString(branding.ThemePrimaryColor),
		NoIndex:           variantSEO.NoIndex,
	}
	if branding.CanonicalBaseURL != nil && *branding.CanonicalBaseURL != "" {
		canonicalPath := variantSEO.CanonicalPath
		if canonicalPath == "" {
			canonicalPath = "/"
		}
		response.CanonicalURL = strings.TrimSuffix(*branding.CanonicalBaseURL, "/") + canonicalPath
	}
	if variantSEO.StructuredData != nil {
		raw, _ := json.Marshal(variantSEO.StructuredData)
		rawMsg := json.RawMessage(raw)
		response.StructuredData = &rawMsg
	}
	return &response, nil
}

// UpdateVariantSEO persists a variant's SEO policy through the content-owned
// store seam. Transport adapters do not need to know how variants are stored.
func (s *SEOService) UpdateVariantSEO(slug string, config contracts.VariantSEOConfig) error {
	return s.store.UpdateVariantSEO(slug, config)
}

func (s *SEOService) SitemapXML(fallbackBase string) (string, error) {
	branding := s.store.Branding()
	baseURL := strings.TrimSpace(fallbackBase)
	if branding.CanonicalBaseURL != nil && strings.TrimSpace(*branding.CanonicalBaseURL) != "" {
		baseURL = strings.TrimSuffix(strings.TrimSpace(*branding.CanonicalBaseURL), "/")
	}
	var sb strings.Builder
	sb.WriteString("<?xml version=\\\"1.0\\\" encoding=\\\"UTF-8\\\"?>\\n\\n")
	sb.WriteString("<urlset xmlns=\\\"http://www.sitemaps.org/schemas/sitemap/0.9\\\">\\n")
	sb.WriteString(fmt.Sprintf("  <url>\\n    <loc>%s/</loc>\\n    <changefreq>weekly</changefreq>\\n    <priority>1.0</priority>\\n  </url>\\n", baseURL))
	for _, variant := range s.store.Variants() {
		var seoConfig contracts.VariantSEOConfig
		if len(variant.SEOConfig) > 0 {
			if err := json.Unmarshal(variant.SEOConfig, &seoConfig); err != nil {
				s.logf("seo_config_parse_failed", map[string]interface{}{"slug": variant.Slug, "error": err.Error()})
			}
		}
		if !seoConfig.NoIndex && seoConfig.CanonicalPath != "" && seoConfig.CanonicalPath != "/" {
			sb.WriteString(fmt.Sprintf("  <url>\\n    <loc>%s%s</loc>\\n    <changefreq>weekly</changefreq>\\n    <priority>0.8</priority>\\n  </url>\\n", baseURL, seoConfig.CanonicalPath))
		}
	}
	sb.WriteString("</urlset>\\n")
	return sb.String(), nil
}

func (s *SEOService) RobotsTXT(fallbackBase string) (string, error) {
	branding := s.store.Branding()
	robotsTxt := "User-agent: *\\nAllow: /\\n"
	if branding.RobotsTxt != nil && strings.TrimSpace(*branding.RobotsTxt) != "" {
		robotsTxt = *branding.RobotsTxt
	}
	baseURL := strings.TrimSpace(fallbackBase)
	if branding.CanonicalBaseURL != nil && strings.TrimSpace(*branding.CanonicalBaseURL) != "" {
		baseURL = strings.TrimSuffix(strings.TrimSpace(*branding.CanonicalBaseURL), "/")
	}
	if baseURL != "" && !strings.Contains(robotsTxt, "Sitemap:") {
		robotsTxt += fmt.Sprintf("\\nSitemap: %s/sitemap.xml\\n", baseURL)
	}
	return robotsTxt, nil
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
