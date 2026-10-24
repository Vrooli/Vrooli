package content

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
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

// PublishedRouteSet is the only presentation data crawlers need. It contains
// routes proven public by published immutable documents, never preview or
// mutable legacy page names.
type PublishedRouteSet struct {
	Authoritative  bool
	Root           bool
	AppDetailPaths []string
}

// PublishedRouteStore is optional for the pre-seed compatibility seam. The
// ConfigStore adapter implements it; a store that does not implement it keeps
// the old legacy test behavior until a published presentation exists.
type PublishedRouteStore interface {
	PublishedPublicRoutes(context.Context) (PublishedRouteSet, error)
}

var (
	ErrCanonicalBaseURL       = errors.New("trusted canonical base URL is not configured")
	ErrPublishedRoutesMissing = errors.New("published presentation routes are unavailable")
)

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
	return s.SitemapXMLContext(context.Background(), fallbackBase)
}

func (s *SEOService) SitemapXMLContext(ctx context.Context, fallbackBase string) (string, error) {
	_ = fallbackBase // Request-derived hosts are never trusted for crawler URLs.
	branding := s.store.Branding()
	baseURL, err := trustedCanonicalBaseURL(branding)
	if err != nil {
		return "", err
	}
	paths, err := s.sitemapPaths(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(xml.Header)
	sb.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for index, path := range paths {
		priority := "0.8"
		if index == 0 && path == "/" {
			priority = "1.0"
		} else if slices.Contains(SitePagePaths, path) {
			priority = "0.3"
		}
		appendSitemapURL(&sb, joinCanonicalPath(baseURL, path), priority)
	}
	sb.WriteString("</urlset>\n")
	return sb.String(), nil
}

func (s *SEOService) RobotsTXT(fallbackBase string) (string, error) {
	return s.RobotsTXTContext(context.Background(), fallbackBase)
}

func (s *SEOService) RobotsTXTContext(_ context.Context, fallbackBase string) (string, error) {
	_ = fallbackBase // Request-derived hosts are never trusted for crawler URLs.
	branding := s.store.Branding()
	robotsTxt := "User-agent: *\nDisallow: /admin/\nDisallow: /auth/\nDisallow: /login\nDisallow: /checkout\nDisallow: /api/\n"
	if branding.RobotsTxt != nil && strings.TrimSpace(*branding.RobotsTxt) != "" {
		robotsTxt = normalizeRobotsText(*branding.RobotsTxt)
	}
	baseURL, err := trustedCanonicalBaseURL(branding)
	if err != nil && branding.CanonicalBaseURL != nil && strings.TrimSpace(*branding.CanonicalBaseURL) != "" {
		return "", err
	}
	if err == nil && !strings.Contains(strings.ToLower(robotsTxt), "sitemap:") {
		robotsTxt = strings.TrimRight(robotsTxt, "\r\n") + "\n\nSitemap: " + joinCanonicalPath(baseURL, "/sitemap.xml") + "\n"
	}
	return robotsTxt, nil
}

func (s *SEOService) sitemapPaths(ctx context.Context) ([]string, error) {
	if routeStore, ok := s.store.(PublishedRouteStore); ok {
		routes, err := routeStore.PublishedPublicRoutes(ctx)
		if err != nil {
			return nil, err
		}
		if routes.Authoritative {
			if !routes.Root && len(routes.AppDetailPaths) == 0 {
				return nil, ErrPublishedRoutesMissing
			}
			paths := make([]string, 0, len(routes.AppDetailPaths)+1)
			if routes.Root {
				paths = append(paths, "/")
			}
			paths = append(paths, routes.AppDetailPaths...)
			paths = append(paths, SitePagePaths...)
			return stablePublicPaths(paths), nil
		}
	}

	// Explicit pre-seed compatibility: only stores without an authoritative
	// published-presentation seam may expose their legacy SEO paths.
	paths := []string{"/"}
	for _, variant := range s.store.Variants() {
		var seoConfig contracts.VariantSEOConfig
		if len(variant.SEOConfig) > 0 {
			if err := json.Unmarshal(variant.SEOConfig, &seoConfig); err != nil {
				s.logf("seo_config_parse_failed", map[string]interface{}{"slug": variant.Slug, "error": err.Error()})
				continue
			}
		}
		if !seoConfig.NoIndex && seoConfig.CanonicalPath != "" {
			if path, ok := safePublicPath(seoConfig.CanonicalPath); ok {
				paths = append(paths, path)
			}
		}
	}
	paths = append(paths, SitePagePaths...)
	return stablePublicPaths(paths), nil
}

// SitePagePaths are the indexable site-owned pages every deployment serves
// alongside its published presentation routes.
var SitePagePaths = []string{"/contact", "/privacy", "/terms"}

func trustedCanonicalBaseURL(branding SEOBranding) (string, error) {
	if branding.CanonicalBaseURL == nil {
		return "", ErrCanonicalBaseURL
	}
	raw := strings.TrimSpace(*branding.CanonicalBaseURL)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("%w: invalid URL", ErrCanonicalBaseURL)
	}
	return strings.TrimRight(raw, "/"), nil
}

func joinCanonicalPath(base, path string) string {
	if path == "/" {
		return base + "/"
	}
	return base + path
}

func appendSitemapURL(sb *strings.Builder, location, priority string) {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(location))
	fmt.Fprintf(sb, "  <url>\n    <loc>%s</loc>\n    <changefreq>weekly</changefreq>\n    <priority>%s</priority>\n  </url>\n", escaped.String(), priority)
}

func stablePublicPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if normalized, ok := safePublicPath(path); ok {
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			result = append(result, normalized)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i] == "/" {
			return true
		}
		if result[j] == "/" {
			return false
		}
		return result[i] < result[j]
	})
	return result
}

func safePublicPath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "/" {
		return path, true
	}
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "", false
	}
	clean := strings.TrimRight(parsed.Path, "/")
	for _, blocked := range []string{"/admin", "/auth", "/login", "/checkout", "/api"} {
		if clean == blocked || strings.HasPrefix(clean, blocked+"/") {
			return "", false
		}
	}
	return clean, clean != ""
}

func normalizeRobotsText(value string) string {
	return strings.ReplaceAll(value, `\n`, "\n")
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
