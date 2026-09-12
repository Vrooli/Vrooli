// Package seo resolves per-variant SEO head payloads (merging variant overrides
// with site-branding defaults), stores overrides on the variant row, and
// renders the raw sitemap.xml / robots.txt. It owns no tables — it composes the
// branding and variant application services.
package seo

import (
	"context"
	"encoding/json"
	"fmt"
	"landing-page-react-vite-api/internal/branding"
	"landing-page-react-vite-api/internal/variant"
	"strings"
	"time"
)

// Service resolves and stores SEO data by composing branding and variant.
type Service struct {
	branding *branding.Service
	variant  *variant.Service
}

// NewService constructs the seo Service.
func NewService(brandingSvc *branding.Service, variantSvc *variant.Service) *Service {
	return &Service{branding: brandingSvc, variant: variantSvc}
}

// ResolvedSEO is the merged head payload the frontend renders.
type ResolvedSEO struct {
	SiteName          string
	Title             string
	Description       string
	OGTitle           string
	OGDescription     string
	OGImageURL        string
	TwitterCard       string
	CanonicalURL      string
	FaviconURL        string
	AppleTouchIconURL string
	ThemePrimaryColor string
	NoIndex           bool
	StructuredData    map[string]interface{}
}

// ResolveHead merges a variant's SEO overrides with site-branding defaults.
func (s *Service) ResolveHead(ctx context.Context, slug string) (*ResolvedSEO, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("variant slug required")
	}
	b, err := s.branding.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get branding: %w", err)
	}
	v, err := s.variant.GetVariantBySlug(slug)
	if err != nil {
		return nil, err
	}
	cfg := parseVariantSEO(v.SEOConfig)

	res := &ResolvedSEO{
		SiteName:          b.SiteName,
		Title:             coalesce(cfg.Title, deref(b.DefaultTitle), b.SiteName),
		Description:       coalesce(cfg.Description, deref(b.DefaultDescription), ""),
		OGTitle:           coalesce(cfg.OGTitle, cfg.Title, deref(b.DefaultTitle), b.SiteName),
		OGDescription:     coalesce(cfg.OGDescription, cfg.Description, deref(b.DefaultDescription), ""),
		OGImageURL:        coalesce(cfg.OGImageURL, deref(b.DefaultOGImageURL)),
		TwitterCard:       coalesce(cfg.TwitterCard, "summary_large_image"),
		FaviconURL:        deref(b.FaviconURL),
		AppleTouchIconURL: deref(b.AppleTouchIconURL),
		ThemePrimaryColor: deref(b.ThemePrimaryColor),
		NoIndex:           cfg.NoIndex,
		StructuredData:    cfg.StructuredData,
	}
	if b.CanonicalBaseURL != nil && *b.CanonicalBaseURL != "" {
		canonicalPath := cfg.CanonicalPath
		if canonicalPath == "" {
			canonicalPath = "/"
		}
		res.CanonicalURL = strings.TrimSuffix(*b.CanonicalBaseURL, "/") + canonicalPath
	}
	return res, nil
}

// UpdateOverride stores SEO overrides for a variant and returns the update time.
func (s *Service) UpdateOverride(slug string, cfg variant.SEOConfigJSON) (time.Time, error) {
	if strings.TrimSpace(slug) == "" {
		return time.Time{}, fmt.Errorf("variant slug required")
	}
	v, err := s.variant.GetVariantBySlug(slug)
	if err != nil {
		return time.Time{}, err
	}
	seoJSON, err := json.Marshal(cfg)
	if err != nil {
		return time.Time{}, fmt.Errorf("encode SEO config: %w", err)
	}
	if err := s.variant.UpdateSEOConfig(v.ID, seoJSON); err != nil {
		return time.Time{}, fmt.Errorf("update SEO config: %w", err)
	}
	return time.Now().UTC(), nil
}

// SitemapXML builds the dynamic sitemap for active, indexable variants.
func (s *Service) SitemapXML(ctx context.Context, requestBaseURL string) (string, error) {
	b, err := s.branding.Get(ctx)
	if err != nil {
		return "", err
	}
	baseURL := ""
	if b.CanonicalBaseURL != nil {
		baseURL = strings.TrimSuffix(*b.CanonicalBaseURL, "/")
	}
	if baseURL == "" {
		baseURL = strings.TrimSuffix(requestBaseURL, "/")
	}

	variants, err := s.variant.ActiveWithSEO()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	sb.WriteString("  <url>\n")
	fmt.Fprintf(&sb, "    <loc>%s/</loc>\n", baseURL)
	sb.WriteString("    <changefreq>weekly</changefreq>\n")
	sb.WriteString("    <priority>1.0</priority>\n")
	sb.WriteString("  </url>\n")

	for _, v := range variants {
		if v.Status != "active" {
			continue
		}
		cfg := parseVariantSEO(v.SEOConfig)
		if cfg.NoIndex {
			continue
		}
		if cfg.CanonicalPath != "" && cfg.CanonicalPath != "/" {
			sb.WriteString("  <url>\n")
			fmt.Fprintf(&sb, "    <loc>%s%s</loc>\n", baseURL, cfg.CanonicalPath)
			sb.WriteString("    <changefreq>weekly</changefreq>\n")
			sb.WriteString("    <priority>0.8</priority>\n")
			sb.WriteString("  </url>\n")
		}
	}
	sb.WriteString("</urlset>\n")
	return sb.String(), nil
}

// RobotsTXT renders robots.txt from branding, appending a sitemap reference
// when a canonical base URL is configured.
func (s *Service) RobotsTXT(ctx context.Context) string {
	robots := "User-agent: *\nAllow: /\n"
	b, err := s.branding.Get(ctx)
	if err != nil {
		return robots
	}
	if b.RobotsTxt != nil && *b.RobotsTxt != "" {
		robots = *b.RobotsTxt
	}
	if b.CanonicalBaseURL != nil && *b.CanonicalBaseURL != "" {
		baseURL := strings.TrimSuffix(*b.CanonicalBaseURL, "/")
		if !strings.Contains(robots, "Sitemap:") {
			robots += fmt.Sprintf("\nSitemap: %s/sitemap.xml\n", baseURL)
		}
	}
	return robots
}

func parseVariantSEO(raw *json.RawMessage) variant.SEOConfigJSON {
	var cfg variant.SEOConfigJSON
	if raw != nil && len(*raw) > 0 {
		_ = json.Unmarshal(*raw, &cfg)
	}
	return cfg
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
