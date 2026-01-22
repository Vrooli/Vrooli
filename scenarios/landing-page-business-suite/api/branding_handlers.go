package main

import (
	"net/http"
)

// handleGetBranding returns the site branding configuration (from ConfigStore)
func handleGetBranding(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		branding := cs.GetBranding()
		writeJSONSuccessData(w, branding)
	}
}

// handleUpdateBranding updates the site branding configuration (writes to JSON file)
func handleUpdateBranding(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BrandingUpdateRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}

		branding, err := cs.UpdateBranding(&req)
		if err != nil {
			logStructuredError("update_branding_failed", map[string]interface{}{"error": err.Error()})
			writeJSONError(w, http.StatusInternalServerError, "Failed to update branding", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, branding)
	}
}

// handleClearBrandingField clears a specific branding field (writes to JSON file)
func handleClearBrandingField(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Field string `json:"field"`
		}
		if !decodeJSONBody(w, r, &req) {
			return
		}

		field, ok := RequireNonEmpty(w, req.Field, "Field name")
		if !ok {
			return
		}

		if err := cs.ClearBrandingField(field); err != nil {
			logStructuredError("clear_branding_field_failed", map[string]interface{}{
				"field": field,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to clear field", ApiErrorTypeServerError)
			return
		}

		// Return updated branding
		branding := cs.GetBranding()
		writeJSONSuccessData(w, branding)
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
			"coming_soon_enabled":    branding.ComingSoonEnabled,
			"coming_soon_message":    branding.ComingSoonMessage,
		}

		writeJSONSuccessData(w, publicBranding)
	}
}
