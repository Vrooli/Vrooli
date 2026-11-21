package delivery

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	internal "landing-page-business-suite-api/internal/delivery"
)

type AppRequest struct {
	AppKey          string                 `json:"app_key"`
	Name            string                 `json:"name"`
	Tagline         string                 `json:"tagline"`
	Description     string                 `json:"description"`
	IconURL         string                 `json:"icon_url"`
	ScreenshotURL   string                 `json:"screenshot_url"`
	InstallOverview string                 `json:"install_overview"`
	InstallSteps    []string               `json:"install_steps"`
	Storefronts     []internal.Storefront  `json:"storefronts"`
	Metadata        map[string]interface{} `json:"metadata"`
	DisplayOrder    *int                   `json:"display_order"`
	Platforms       []AssetRequest         `json:"platforms"`
}

type AssetRequest struct {
	Platform            string                 `json:"platform"`
	ArtifactURL         string                 `json:"artifact_url"`
	ArtifactSource      string                 `json:"artifact_source"`
	ArtifactID          *int64                 `json:"artifact_id"`
	ReleaseVersion      string                 `json:"release_version"`
	ReleaseNotes        string                 `json:"release_notes"`
	Checksum            string                 `json:"checksum"`
	RequiresEntitlement *bool                  `json:"requires_entitlement"`
	Metadata            map[string]interface{} `json:"metadata"`
}

type (
	AppCatalog interface {
		ListApps(string) ([]internal.App, error)
		UpsertApp(internal.App) (*internal.App, error)
		DeleteApp(string, string) error
	}
	AppDependencies struct {
		BundleKey    func() string
		PathParam    func(*http.Request, string) (string, bool)
		DecodeJSON   func(http.ResponseWriter, *http.Request, any) bool
		WriteError   func(http.ResponseWriter, int, string, string)
		WriteData    func(http.ResponseWriter, any)
		WriteSuccess func(http.ResponseWriter)
	}
)

// ListApps returns the active bundle's download application catalog.
func ListApps(deps AppDependencies, catalog AppCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apps, err := catalog.ListApps(deps.BundleKey())
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list download apps: %v", err), "server_error")
			return
		}
		deps.WriteData(w, map[string]interface{}{"apps": apps})
	}
}

// CreateApp creates a new application using its payload key.
func CreateApp(deps AppDependencies, catalog AppCatalog) http.HandlerFunc {
	return saveApp(deps, catalog, "")
}

// SaveApp replaces an application identified by the route key.
func SaveApp(deps AppDependencies, catalog AppCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := deps.PathParam(r, "app_key")
		if !ok || strings.TrimSpace(appKey) == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key path parameter is required", "validation")
			return
		}
		saveApp(deps, catalog, appKey)(w, r)
	}
}

func saveApp(deps AppDependencies, catalog AppCatalog, overrideKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload AppRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		app, err := BuildAppFromPayload(payload, deps.BundleKey(), overrideKey)
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		saved, err := catalog.UpsertApp(app)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to save download app: %v", err), "server_error")
			return
		}
		deps.WriteData(w, saved)
	}
}

// DeleteApp removes an application identified by the route key.
func DeleteApp(deps AppDependencies, catalog AppCatalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := deps.PathParam(r, "app_key")
		if !ok || strings.TrimSpace(appKey) == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key path parameter is required", "validation")
			return
		}
		if err := catalog.DeleteApp(deps.BundleKey(), appKey); err != nil {
			if errors.Is(err, internal.ErrAppNotFound) {
				deps.WriteError(w, http.StatusNotFound, "download app not found", "not_found")
			} else {
				deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete download app: %v", err), "server_error")
			}
			return
		}
		deps.WriteSuccess(w)
	}
}

func BuildAppFromPayload(payload AppRequest, bundleKey, overrideKey string) (internal.App, error) {
	appKey := strings.TrimSpace(overrideKey)
	if appKey == "" {
		appKey = strings.TrimSpace(payload.AppKey)
	}
	if appKey == "" {
		return internal.App{}, fmt.Errorf("app_key is required")
	}
	displayOrder := 0
	if payload.DisplayOrder != nil {
		displayOrder = *payload.DisplayOrder
	}
	app := internal.App{
		BundleKey:       bundleKey,
		AppKey:          appKey,
		Name:            strings.TrimSpace(payload.Name),
		Tagline:         strings.TrimSpace(payload.Tagline),
		Description:     strings.TrimSpace(payload.Description),
		IconURL:         strings.TrimSpace(payload.IconURL),
		ScreenshotURL:   strings.TrimSpace(payload.ScreenshotURL),
		InstallOverview: strings.TrimSpace(payload.InstallOverview),
		InstallSteps:    FilterStrings(payload.InstallSteps),
		Storefronts:     payload.Storefronts,
		Metadata:        payload.Metadata,
		DisplayOrder:    displayOrder,
	}
	if app.Name == "" {
		return internal.App{}, fmt.Errorf("name is required")
	}
	for _, s := range app.Storefronts {
		if strings.TrimSpace(s.URL) == "" {
			return internal.App{}, fmt.Errorf("storefront url is required when storefront entries are provided")
		}
	}
	for _, p := range payload.Platforms {
		if strings.TrimSpace(p.Platform) == "" {
			return internal.App{}, fmt.Errorf("platform is required for all installers")
		}
		source := strings.TrimSpace(p.ArtifactSource)
		if source == "" {
			source = "direct"
		}
		if source != "direct" && source != "managed" {
			return internal.App{}, fmt.Errorf("artifact_source must be 'direct' or 'managed' for platform %s", p.Platform)
		}
		if source == "direct" {
			if err := internal.ValidateDirectArtifactURL(p.ArtifactURL); err != nil {
				return internal.App{}, fmt.Errorf("platform %s: %w", p.Platform, err)
			}
		} else if p.ArtifactID == nil || *p.ArtifactID == 0 {
			return internal.App{}, fmt.Errorf("artifact_id is required for managed platform %s", p.Platform)
		}
		if strings.TrimSpace(p.ReleaseVersion) == "" {
			return internal.App{}, fmt.Errorf("release_version is required for platform %s", p.Platform)
		}
		required := false
		if p.RequiresEntitlement != nil {
			required = *p.RequiresEntitlement
		}
		app.Platforms = append(app.Platforms, internal.Asset{
			BundleKey:           bundleKey,
			AppKey:              appKey,
			Platform:            strings.TrimSpace(p.Platform),
			ArtifactURL:         strings.TrimSpace(p.ArtifactURL),
			ArtifactSource:      source,
			ArtifactID:          p.ArtifactID,
			ReleaseVersion:      strings.TrimSpace(p.ReleaseVersion),
			ReleaseNotes:        strings.TrimSpace(p.ReleaseNotes),
			Checksum:            strings.TrimSpace(p.Checksum),
			RequiresEntitlement: required,
			Metadata:            p.Metadata,
		})
	}
	return app, nil
}

func FilterStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}
