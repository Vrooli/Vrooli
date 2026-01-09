package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// VariantSEOConfig represents per-variant SEO settings
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

// SEOResponse combines site branding with variant-specific SEO
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

// handleGetVariantSEO returns merged SEO config for a variant
func handleGetVariantSEO(brandingService *BrandingService, variantService *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["slug"]

		if slug == "" {
			http.Error(w, "variant slug required", http.StatusBadRequest)
			return
		}

		// Get site branding
		branding, err := brandingService.Get()
		if err != nil {
			logStructuredError("get_branding_for_seo_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "failed to get branding", http.StatusInternalServerError)
			return
		}

		// Get variant with SEO config
		variant, err := variantService.GetBySlug(slug)
		if err != nil {
			http.Error(w, "variant not found", http.StatusNotFound)
			return
		}

		// Parse variant SEO config if exists
		var variantSEO VariantSEOConfig
		if variant.SEOConfig != nil && len(*variant.SEOConfig) > 0 {
			if err := json.Unmarshal(*variant.SEOConfig, &variantSEO); err != nil {
				logStructuredError("parse_variant_seo_failed", map[string]interface{}{
					"slug":  slug,
					"error": err.Error(),
				})
			}
		}

		// Merge: variant SEO overrides site branding defaults
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

		// Build canonical URL
		if branding.CanonicalBaseURL != nil && *branding.CanonicalBaseURL != "" {
			canonicalPath := variantSEO.CanonicalPath
			if canonicalPath == "" {
				canonicalPath = "/"
			}
			response.CanonicalURL = strings.TrimSuffix(*branding.CanonicalBaseURL, "/") + canonicalPath
		}

		// Include structured data if present
		if variantSEO.StructuredData != nil {
			raw, _ := json.Marshal(variantSEO.StructuredData)
			rawMsg := json.RawMessage(raw)
			response.StructuredData = &rawMsg
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleUpdateVariantSEO updates SEO config for a variant
func handleUpdateVariantSEO(variantService *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["slug"]

		if slug == "" {
			http.Error(w, "variant slug required", http.StatusBadRequest)
			return
		}

		var seoConfig VariantSEOConfig
		if err := json.NewDecoder(r.Body).Decode(&seoConfig); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Get variant ID
		variant, err := variantService.GetBySlug(slug)
		if err != nil {
			http.Error(w, "variant not found", http.StatusNotFound)
			return
		}

		// Update SEO config
		seoJSON, err := json.Marshal(seoConfig)
		if err != nil {
			http.Error(w, "failed to encode SEO config", http.StatusInternalServerError)
			return
		}

		if err := variantService.UpdateSEOConfig(variant.ID, seoJSON); err != nil {
			logStructuredError("update_variant_seo_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			http.Error(w, "failed to update SEO config", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// handleSitemapXML generates a dynamic sitemap.xml
func handleSitemapXML(brandingService *BrandingService, variantService *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding, err := brandingService.Get()
		if err != nil {
			logStructuredError("sitemap_branding_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		baseURL := ""
		if branding.CanonicalBaseURL != nil {
			baseURL = strings.TrimSuffix(*branding.CanonicalBaseURL, "/")
		}

		if baseURL == "" {
			// Fall back to request host
			scheme := "https"
			if r.TLS == nil {
				scheme = "http"
			}
			baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
		}

		// Get active variants
		variants, err := variantService.List()
		if err != nil {
			logStructuredError("sitemap_variants_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Build sitemap
		var sb strings.Builder
		sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		sb.WriteString("\n")
		sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
		sb.WriteString("\n")

		// Add root URL
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s/</loc>\n", baseURL))
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("    <priority>1.0</priority>\n")
		sb.WriteString("  </url>\n")

		// Add variant URLs (for SEO-visible variants only)
		for _, v := range variants {
			if v.Status != "active" {
				continue
			}

			// Parse SEO config to check noindex
			var seoConfig VariantSEOConfig
			if v.SEOConfig != nil {
				json.Unmarshal(*v.SEOConfig, &seoConfig)
			}
			if seoConfig.NoIndex {
				continue
			}

			// Only include variants with explicit canonical paths
			if seoConfig.CanonicalPath != "" && seoConfig.CanonicalPath != "/" {
				sb.WriteString("  <url>\n")
				sb.WriteString(fmt.Sprintf("    <loc>%s%s</loc>\n", baseURL, seoConfig.CanonicalPath))
				sb.WriteString("    <changefreq>weekly</changefreq>\n")
				sb.WriteString("    <priority>0.8</priority>\n")
				sb.WriteString("  </url>\n")
			}
		}

		sb.WriteString("</urlset>\n")

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(sb.String()))
	}
}

// handleRobotsTXT serves configurable robots.txt
func handleRobotsTXT(brandingService *BrandingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding, err := brandingService.Get()
		if err != nil {
			logStructuredError("robots_branding_failed", map[string]interface{}{"error": err.Error()})
			// Return permissive default on error
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte("User-agent: *\nAllow: /\n"))
			return
		}

		robotsTxt := "User-agent: *\nAllow: /\n"
		if branding.RobotsTxt != nil && *branding.RobotsTxt != "" {
			robotsTxt = *branding.RobotsTxt
		}

		// Append sitemap reference if canonical URL is set
		if branding.CanonicalBaseURL != nil && *branding.CanonicalBaseURL != "" {
			baseURL := strings.TrimSuffix(*branding.CanonicalBaseURL, "/")
			if !strings.Contains(robotsTxt, "Sitemap:") {
				robotsTxt += fmt.Sprintf("\nSitemap: %s/sitemap.xml\n", baseURL)
			}
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(robotsTxt))
	}
}

// Helper functions

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

// UpdateSEOConfig updates the seo_config JSON for a variant
func (s *VariantService) UpdateSEOConfig(variantID int, seoJSON []byte) error {
	_, err := s.db.Exec(`
		UPDATE variants
		SET seo_config = $1::jsonb, updated_at = NOW()
		WHERE id = $2
	`, seoJSON, variantID)
	return err
}

// GetVariantSEOConfig retrieves SEO config for a specific variant
func (s *VariantService) GetVariantSEOConfig(variantID int) (*VariantSEOConfig, error) {
	var seoJSON []byte
	err := s.db.QueryRow(`
		SELECT COALESCE(seo_config, '{}'::jsonb)
		FROM variants
		WHERE id = $1
	`, variantID).Scan(&seoJSON)
	if err != nil {
		return nil, err
	}

	var config VariantSEOConfig
	if err := json.Unmarshal(seoJSON, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Variant extension to include SEOConfig
func (v *Variant) GetSEOConfigParsed() (*VariantSEOConfig, error) {
	if v.SEOConfig == nil {
		return &VariantSEOConfig{}, nil
	}
	var config VariantSEOConfig
	if err := json.Unmarshal(*v.SEOConfig, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// parseInt helper for route parameters
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
