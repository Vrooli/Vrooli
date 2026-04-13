package main

import (
	"encoding/json"
	"net/http"
)

// handleGetBranding returns the site branding configuration
func handleGetBranding(bs *BrandingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding, err := bs.Get()
		if err != nil {
			logStructuredError("get_branding_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "Failed to get branding", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(branding)
	}
}

// handleUpdateBranding updates the site branding configuration
func handleUpdateBranding(bs *BrandingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BrandingUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		branding, err := bs.Update(&req)
		if err != nil {
			logStructuredError("update_branding_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "Failed to update branding", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(branding)
	}
}

// handleClearBrandingField clears a specific branding field
func handleClearBrandingField(bs *BrandingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Field string `json:"field"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Field == "" {
			http.Error(w, "Field name required", http.StatusBadRequest)
			return
		}

		if err := bs.ClearField(req.Field); err != nil {
			logStructuredError("clear_branding_field_failed", map[string]interface{}{
				"field": req.Field,
				"error": err.Error(),
			})
			http.Error(w, "Failed to clear field", http.StatusInternalServerError)
			return
		}

		// Return updated branding
		branding, err := bs.Get()
		if err != nil {
			http.Error(w, "Failed to get updated branding", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(branding)
	}
}

// handleGetPublicBranding returns public branding info (no auth required)
func handleGetPublicBranding(bs *BrandingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding, err := bs.Get()
		if err != nil {
			logStructuredError("get_public_branding_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "Failed to get branding", http.StatusInternalServerError)
			return
		}

		// Return only public-safe fields
		publicBranding := map[string]interface{}{
			"site_name":              branding.SiteName,
			"tagline":                branding.Tagline,
			"logo_url":               branding.LogoURL,
			"logo_icon_url":          branding.LogoIconURL,
			"favicon_url":            branding.FaviconURL,
			"theme_primary_color":    branding.ThemePrimaryColor,
			"theme_background_color": branding.ThemeBackgroundColor,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(publicBranding)
	}
}
