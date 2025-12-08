package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// VariantSEOConfig represents per-variant SEO settings.
// Lives alongside the SEO service to keep domain rules out of HTTP handlers.
type VariantSEOConfig struct {
	Title          string                 `json:"title,omitempty"`
	Description    string                 `json:"description,omitempty"`
	OGTitle        string                 `json:"og_title,omitempty"`
	OGDescription  string                 `json:"og_description,omitempty"`
	OGImageURL     string                 `json:"og_image_url,omitempty"`
	TwitterCard    string                 `json:"twitter_card,omitempty"`
	CanonicalPath  string                 `json:"canonical_path,omitempty"`
	NoIndex        bool                   `json:"noindex,omitempty"`
	StructuredData map[string]interface{} `json:"structured_data,omitempty"`
}

// SEOResponse combines site branding with variant-specific SEO.
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

// SEOService owns domain logic for combining branding + variant SEO metadata.
// Handlers should remain transport-only and delegate merging decisions here.
type SEOService struct {
	branding *BrandingService
	variants *VariantService
}

func NewSEOService(branding *BrandingService, variants *VariantService) *SEOService {
	return &SEOService{
		branding: branding,
		variants: variants,
	}
}

// VariantSEO merges site branding defaults with per-variant overrides.
func (s *SEOService) VariantSEO(slug string) (*SEOResponse, error) {
	branding, err := s.branding.Get()
	if err != nil {
		return nil, fmt.Errorf("branding: %w", err)
	}

	variant, err := s.variants.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	var variantSEO VariantSEOConfig
	if variant.SEOConfig != nil && len(*variant.SEOConfig) > 0 {
		if err := json.Unmarshal(*variant.SEOConfig, &variantSEO); err != nil {
			logStructuredError("parse_variant_seo_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
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

// SitemapXML generates the XML string for all SEO-visible variants.
func (s *SEOService) SitemapXML(fallbackBase string) (string, error) {
	branding, err := s.branding.Get()
	if err != nil {
		return "", fmt.Errorf("branding: %w", err)
	}

	baseURL := strings.TrimSpace(fallbackBase)
	if branding.CanonicalBaseURL != nil && strings.TrimSpace(*branding.CanonicalBaseURL) != "" {
		baseURL = strings.TrimSuffix(strings.TrimSpace(*branding.CanonicalBaseURL), "/")
	}

	variants, err := s.variants.List()
	if err != nil {
		return "", fmt.Errorf("variants: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sb.WriteString("\n")

	sb.WriteString("  <url>\n")
	sb.WriteString(fmt.Sprintf("    <loc>%s/</loc>\n", baseURL))
	sb.WriteString("    <changefreq>weekly</changefreq>\n")
	sb.WriteString("    <priority>1.0</priority>\n")
	sb.WriteString("  </url>\n")

	for _, v := range variants {
		if v.Status != "active" {
			continue
		}

		var seoConfig VariantSEOConfig
		if v.SEOConfig != nil {
			json.Unmarshal(*v.SEOConfig, &seoConfig)
		}
		if seoConfig.NoIndex {
			continue
		}
		if seoConfig.CanonicalPath != "" && seoConfig.CanonicalPath != "/" {
			sb.WriteString("  <url>\n")
			sb.WriteString(fmt.Sprintf("    <loc>%s%s</loc>\n", baseURL, seoConfig.CanonicalPath))
			sb.WriteString("    <changefreq>weekly</changefreq>\n")
			sb.WriteString("    <priority>0.8</priority>\n")
			sb.WriteString("  </url>\n")
		}
	}

	sb.WriteString("</urlset>\n")
	return sb.String(), nil
}

// RobotsTXT returns the robots.txt content with optional sitemap hint.
func (s *SEOService) RobotsTXT(fallbackBase string) (string, error) {
	branding, err := s.branding.Get()
	if err != nil {
		return "", fmt.Errorf("branding: %w", err)
	}

	robotsTxt := "User-agent: *\nAllow: /\n"
	if branding.RobotsTxt != nil && strings.TrimSpace(*branding.RobotsTxt) != "" {
		robotsTxt = *branding.RobotsTxt
	}

	baseURL := strings.TrimSpace(fallbackBase)
	if branding.CanonicalBaseURL != nil && strings.TrimSpace(*branding.CanonicalBaseURL) != "" {
		baseURL = strings.TrimSuffix(strings.TrimSpace(*branding.CanonicalBaseURL), "/")
	}

	if baseURL != "" && !strings.Contains(robotsTxt, "Sitemap:") {
		robotsTxt += fmt.Sprintf("\nSitemap: %s/sitemap.xml\n", baseURL)
	}

	return robotsTxt, nil
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
