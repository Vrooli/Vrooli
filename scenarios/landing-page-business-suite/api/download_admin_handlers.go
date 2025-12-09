package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type downloadAppRequest struct {
	AppKey          string                 `json:"app_key"`
	Name            string                 `json:"name"`
	Tagline         string                 `json:"tagline"`
	Description     string                 `json:"description"`
	InstallOverview string                 `json:"install_overview"`
	InstallSteps    []string               `json:"install_steps"`
	Storefronts     []DownloadStorefront   `json:"storefronts"`
	Metadata        map[string]interface{} `json:"metadata"`
	DisplayOrder    *int                   `json:"display_order"`
	Platforms       []downloadAssetRequest `json:"platforms"`
}

type downloadAssetRequest struct {
	Platform            string                 `json:"platform"`
	ArtifactURL         string                 `json:"artifact_url"`
	ReleaseVersion      string                 `json:"release_version"`
	ReleaseNotes        string                 `json:"release_notes"`
	Checksum            string                 `json:"checksum"`
	RequiresEntitlement *bool                  `json:"requires_entitlement"`
	Metadata            map[string]interface{} `json:"metadata"`
}

func handleAdminListDownloadApps(downloads *DownloadService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apps, err := downloads.ListApps(plans.BundleKey())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list download apps: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]interface{}{
			"apps": apps,
		})
	}
}

func handleAdminCreateDownloadApp(downloads *DownloadService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload downloadAppRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		app, err := buildDownloadAppFromPayload(payload, plans.BundleKey(), payload.AppKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := downloads.UpsertDownloadApp(app)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to save download app: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, created)
	}
}

func handleAdminSaveDownloadApp(downloads *DownloadService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appKey := vars["app_key"]
		if strings.TrimSpace(appKey) == "" {
			http.Error(w, "app_key path parameter is required", http.StatusBadRequest)
			return
		}

		var payload downloadAppRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		app, err := buildDownloadAppFromPayload(payload, plans.BundleKey(), appKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := downloads.UpsertDownloadApp(app)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to save download app: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, updated)
	}
}

func handleAdminDeleteDownloadApp(downloads *DownloadService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appKey := vars["app_key"]
		if strings.TrimSpace(appKey) == "" {
			http.Error(w, "app_key path parameter is required", http.StatusBadRequest)
			return
		}

		if err := downloads.DeleteApp(plans.BundleKey(), appKey); err != nil {
			if errors.Is(err, ErrDownloadAppNotFound) {
				http.Error(w, "download app not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("failed to delete download app: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]bool{"success": true})
	}
}

func buildDownloadAppFromPayload(payload downloadAppRequest, bundleKey string, overrideKey string) (DownloadApp, error) {
	appKey := strings.TrimSpace(overrideKey)
	if appKey == "" {
		appKey = strings.TrimSpace(payload.AppKey)
	}
	if appKey == "" {
		return DownloadApp{}, fmt.Errorf("app_key is required")
	}

	displayOrder := 0
	if payload.DisplayOrder != nil {
		displayOrder = *payload.DisplayOrder
	}

	app := DownloadApp{
		BundleKey:       bundleKey,
		AppKey:          appKey,
		Name:            strings.TrimSpace(payload.Name),
		Tagline:         strings.TrimSpace(payload.Tagline),
		Description:     strings.TrimSpace(payload.Description),
		InstallOverview: strings.TrimSpace(payload.InstallOverview),
		InstallSteps:    filterStrings(payload.InstallSteps),
		Storefronts:     payload.Storefronts,
		Metadata:        payload.Metadata,
		DisplayOrder:    displayOrder,
	}

	if app.Name == "" {
		return DownloadApp{}, fmt.Errorf("name is required")
	}

	for _, storefront := range app.Storefronts {
		if strings.TrimSpace(storefront.URL) == "" {
			return DownloadApp{}, fmt.Errorf("storefront url is required when storefront entries are provided")
		}
	}

	for _, platform := range payload.Platforms {
		if strings.TrimSpace(platform.Platform) == "" {
			return DownloadApp{}, fmt.Errorf("platform is required for all installers")
		}
		if strings.TrimSpace(platform.ArtifactURL) == "" {
			return DownloadApp{}, fmt.Errorf("artifact_url is required for platform %s", platform.Platform)
		}
		if strings.TrimSpace(platform.ReleaseVersion) == "" {
			return DownloadApp{}, fmt.Errorf("release_version is required for platform %s", platform.Platform)
		}

		requireEntitlement := false
		if platform.RequiresEntitlement != nil {
			requireEntitlement = *platform.RequiresEntitlement
		}

		app.Platforms = append(app.Platforms, DownloadAsset{
			BundleKey:           bundleKey,
			AppKey:              appKey,
			Platform:            strings.TrimSpace(platform.Platform),
			ArtifactURL:         strings.TrimSpace(platform.ArtifactURL),
			ReleaseVersion:      strings.TrimSpace(platform.ReleaseVersion),
			ReleaseNotes:        strings.TrimSpace(platform.ReleaseNotes),
			Checksum:            strings.TrimSpace(platform.Checksum),
			RequiresEntitlement: requireEntitlement,
			Metadata:            platform.Metadata,
		})
	}

	return app, nil
}

func filterStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}
