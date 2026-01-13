package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleGetVariantSEO returns merged SEO config for a variant
func handleGetVariantSEO(seoService *SEOService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["slug"]

		if slug == "" {
			http.Error(w, "variant slug required", http.StatusBadRequest)
			return
		}

		response, err := seoService.VariantSEO(slug)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			logStructuredError("get_variant_seo_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			http.Error(w, "failed to get SEO configuration", status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
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

		if err := variantService.UpdateSEOConfigBySlug(slug, seoConfig); err != nil {
			logStructuredError("update_variant_seo_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			status := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				status = http.StatusNotFound
			} else if strings.Contains(err.Error(), "slug") {
				status = http.StatusBadRequest
			}
			http.Error(w, "failed to update SEO config", status)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// handleSitemapXML generates a dynamic sitemap.xml
func handleSitemapXML(seoService *SEOService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		fallbackBase := fmt.Sprintf("%s://%s", scheme, r.Host)

		sitemap, err := seoService.SitemapXML(fallbackBase)
		if err != nil {
			logStructuredError("sitemap_generate_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		if _, err := w.Write([]byte(sitemap)); err != nil {
			logStructuredError("sitemap_write_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleRobotsTXT serves configurable robots.txt
func handleRobotsTXT(seoService *SEOService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		fallbackBase := fmt.Sprintf("%s://%s", scheme, r.Host)

		robotsTxt, err := seoService.RobotsTXT(fallbackBase)
		if err != nil {
			logStructuredError("robots_branding_failed", map[string]interface{}{"error": err.Error()})
			robotsTxt = "User-agent: *\nAllow: /\n"
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte(robotsTxt)); err != nil {
			logStructuredError("robots_write_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}
