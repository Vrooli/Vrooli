package main

import (
	"encoding/json"
	"net/http"

	brandinghttp "landing-page-business-suite-api/handlers/branding"
)

func brandingDependencies(store *ConfigStore) brandinghttp.Dependencies {
	return brandinghttp.Dependencies{
		Get: func() any { return store.GetBranding() },
		Public: func() any {
			branding := store.GetBranding()
			return map[string]any{"site_name": branding.SiteName, "tagline": branding.Tagline, "logo_url": branding.LogoURL, "logo_icon_url": branding.LogoIconURL, "favicon_url": branding.FaviconURL, "theme_primary_color": branding.ThemePrimaryColor, "theme_background_color": branding.ThemeBackgroundColor, "coming_soon_enabled": branding.ComingSoonEnabled, "coming_soon_message": branding.ComingSoonMessage}
		},
		Update: func(raw json.RawMessage) (any, error) {
			var request BrandingUpdateRequest
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			return store.UpdateBranding(&request)
		},
		Clear:      store.ClearBrandingField,
		DecodeJSON: decodeJSONBody,
		WriteJSON:  writeJSONSuccessData,
		WriteError: writeJSONError,
		Log:        logStructuredError,
	}
}

func handleGetBranding(store *ConfigStore) http.HandlerFunc {
	return brandinghttp.Get(brandingDependencies(store))
}
func handleUpdateBranding(store *ConfigStore) http.HandlerFunc {
	return brandinghttp.Update(brandingDependencies(store))
}
func handleClearBrandingField(store *ConfigStore) http.HandlerFunc {
	return brandinghttp.Clear(brandingDependencies(store))
}
func handleGetPublicBranding(store *ConfigStore) http.HandlerFunc {
	return brandinghttp.Public(brandingDependencies(store))
}
