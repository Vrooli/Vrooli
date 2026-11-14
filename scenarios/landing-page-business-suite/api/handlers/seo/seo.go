// Package seo owns public crawler and variant metadata HTTP transport.
package seo

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Dependencies struct {
	VariantSEO func(string) (any, error)
	Update     func(string, json.RawMessage) (bool, error)
	Sitemap    func(context.Context, string) (string, error)
	Robots     func(context.Context, string) (string, error)
	Path       func(*http.Request, string) (string, bool)
	DecodeJSON func(http.ResponseWriter, *http.Request, any) bool
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
	Now        func() time.Time
}

func Variant(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug, ok := deps.Path(r, "slug")
		if !ok || slug == "" {
			deps.WriteError(w, http.StatusBadRequest, "variant slug required", "validation")
			return
		}
		response, err := deps.VariantSEO(slug)
		if err != nil {
			status, kind := http.StatusInternalServerError, "server_error"
			if strings.Contains(err.Error(), "not found") {
				status, kind = http.StatusNotFound, "not_found"
			}
			deps.Log("get_variant_seo_failed", map[string]any{"slug": slug, "error": err.Error()})
			deps.WriteError(w, status, "failed to get SEO configuration", kind)
			return
		}
		deps.WriteJSON(w, response)
	}
}

func Update(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug, ok := deps.Path(r, "slug")
		if !ok || slug == "" {
			deps.WriteError(w, http.StatusBadRequest, "variant slug required", "validation")
			return
		}
		var config json.RawMessage
		if !deps.DecodeJSON(w, r, &config) {
			return
		}
		found, err := deps.Update(slug, config)
		if err != nil {
			deps.Log("update_variant_seo_failed", map[string]any{"slug": slug, "error": err.Error()})
			if !found {
				deps.WriteError(w, http.StatusNotFound, "variant not found", "not_found")
			} else {
				deps.WriteError(w, http.StatusInternalServerError, "failed to save SEO config", "server_error")
			}
			return
		}
		deps.WriteJSON(w, map[string]any{"success": true, "updated_at": deps.Now().UTC().Format(time.RFC3339)})
	}
}

func Sitemap(deps Dependencies) http.HandlerFunc {
	return text(deps, "application/xml; charset=utf-8", "sitemap_generate_failed", "sitemap unavailable", deps.Sitemap, "")
}

func Robots(deps Dependencies) http.HandlerFunc {
	return text(deps, "text/plain; charset=utf-8", "robots_branding_failed", "robots unavailable", deps.Robots, "User-agent: *\nDisallow: /\n")
}

func text(deps Dependencies, contentType, event, message string, render func(context.Context, string) (string, error), failureBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := render(r.Context(), "")
		if err != nil {
			deps.Log(event, map[string]any{"error": err.Error()})
			if failureBody == "" {
				deps.WriteError(w, http.StatusServiceUnavailable, message, "unavailable")
				return
			}
			body = failureBody
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.Header().Set("Content-Type", contentType)
		if _, err := w.Write([]byte(body)); err != nil {
			deps.Log(strings.TrimSuffix(event, "_failed")+"_write_failed", map[string]any{"error": err.Error()})
		}
	}
}
