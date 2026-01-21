package main

import (
	"encoding/json"
	"net/http"
)

// handleGetBranding returns the site branding configuration (from ConfigStore)
func handleGetBranding(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding := cs.GetBranding()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(branding); err != nil {
			http.Error(w, "Failed to encode branding", http.StatusInternalServerError)
		}
	}
}

// handleUpdateBranding updates the site branding configuration (writes to JSON file)
func handleUpdateBranding(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BrandingUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		branding, err := cs.UpdateBranding(&req)
		if err != nil {
			logStructuredError("update_branding_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "Failed to update branding", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(branding); err != nil {
			http.Error(w, "Failed to encode branding", http.StatusInternalServerError)
		}
	}
}

// handleClearBrandingField clears a specific branding field (writes to JSON file)
func handleClearBrandingField(cs *ConfigStore) http.HandlerFunc {
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

		if err := cs.ClearBrandingField(req.Field); err != nil {
			logStructuredError("clear_branding_field_failed", map[string]interface{}{
				"field": req.Field,
				"error": err.Error(),
			})
			http.Error(w, "Failed to clear field", http.StatusInternalServerError)
			return
		}

		// Return updated branding
		branding := cs.GetBranding()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(branding); err != nil {
			http.Error(w, "Failed to encode branding", http.StatusInternalServerError)
		}
	}
}

// handleGetPublicBranding returns public branding info (no auth required, from ConfigStore)
func handleGetPublicBranding(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding := cs.GetBranding()

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
		if err := json.NewEncoder(w).Encode(publicBranding); err != nil {
			http.Error(w, "Failed to encode branding", http.StatusInternalServerError)
		}
	}
}
