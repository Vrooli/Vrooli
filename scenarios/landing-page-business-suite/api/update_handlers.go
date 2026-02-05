package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// --- Narrow interfaces for update handler dependencies ---

// updateAppLookup retrieves a download app for API key gating.
type updateAppLookup interface {
	GetApp(bundleKey, appKey string) (*DownloadApp, error)
}

// updateAssetLookup retrieves a download asset by variant for manifest serving.
type updateAssetLookup interface {
	GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*DownloadAsset, error)
}

// updateArtifactResolver retrieves artifacts and generates presigned download URLs.
type updateArtifactResolver interface {
	GetArtifact(ctx context.Context, bundleKey string, id int64) (*DownloadArtifact, error)
	GetCurrentArtifactByFilename(ctx context.Context, bundleKey, appKey, variantKey, filename string) (*DownloadArtifact, error)
	PresignGetArtifact(ctx context.Context, bundleKey string, artifact DownloadArtifact) (string, error)
}

// updateBundleKeyProvider returns the active bundle key.
type updateBundleKeyProvider interface {
	BundleKey() string
}

// manifestFilenameToPlatform maps electron-updater manifest filenames to platform values.
func manifestFilenameToPlatform(filename string) string {
	switch filename {
	case "latest.yml":
		return "windows"
	case "latest-mac.yml":
		return "mac"
	case "latest-linux.yml":
		return "linux"
	default:
		return ""
	}
}

// channelToVariantKey maps an update channel name to a download_assets variant_key.
func channelToVariantKey(channel string) string {
	if channel == "stable" || channel == "" {
		return "default"
	}
	return channel
}

// buildElectronManifest generates a YAML manifest in the format electron-updater expects.
func buildElectronManifest(artifact *DownloadArtifact) []byte {
	return []byte(fmt.Sprintf(
		"version: %s\npath: %s\nsha512: %s\nreleaseDate: %s\nfiles:\n  - url: %s\n    sha512: %s\n    size: %d\n",
		artifact.ReleaseVersion,
		artifact.OriginalFilename,
		artifact.SHA512,
		artifact.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		artifact.OriginalFilename,
		artifact.SHA512,
		artifact.SizeBytes,
	))
}

// handleUpdateFile serves electron-updater manifest files and binary download redirects.
// Route: GET /api/v1/updates/{app_key}/{channel}/{file}
func handleUpdateFile(apps updateAppLookup, assets updateAssetLookup, artifacts updateArtifactResolver, bundles updateBundleKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := getPathParam(r, "app_key")
		if !ok || appKey == "" {
			writeJSONError(w, http.StatusBadRequest, "app_key is required", ApiErrorTypeValidation)
			return
		}
		channel, ok := getPathParam(r, "channel")
		if !ok || channel == "" {
			writeJSONError(w, http.StatusBadRequest, "channel is required", ApiErrorTypeValidation)
			return
		}
		file, ok := getPathParam(r, "file")
		if !ok || file == "" {
			writeJSONError(w, http.StatusBadRequest, "file is required", ApiErrorTypeValidation)
			return
		}

		bundleKey := bundles.BundleKey()

		// Check per-app API key gating
		app, err := apps.GetApp(bundleKey, appKey)
		if err != nil {
			if errors.Is(err, ErrDownloadAppNotFound) {
				writeJSONError(w, http.StatusNotFound, "app not found", ApiErrorTypeNotFound)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to look up app", ApiErrorTypeServerError)
			return
		}
		if app.UpdateAPIKey != "" {
			provided := strings.TrimSpace(r.Header.Get("X-Update-Key"))
			if provided == "" || provided != app.UpdateAPIKey {
				writeJSONError(w, http.StatusForbidden, "invalid or missing update API key", ApiErrorTypeForbidden)
				return
			}
		}

		variantKey := channelToVariantKey(channel)

		// Check if this is a manifest request
		if platform := manifestFilenameToPlatform(file); platform != "" {
			asset, err := assets.GetAssetByVariant(bundleKey, appKey, platform, variantKey)
			if err != nil {
				if errors.Is(err, ErrDownloadNotFound) {
					writeJSONError(w, http.StatusNotFound, "no asset found for platform/channel", ApiErrorTypeNotFound)
					return
				}
				writeJSONError(w, http.StatusInternalServerError, "failed to look up asset", ApiErrorTypeServerError)
				return
			}
			if asset.ArtifactID == nil {
				writeJSONError(w, http.StatusNotFound, "no managed artifact linked", ApiErrorTypeNotFound)
				return
			}

			artifact, err := artifacts.GetArtifact(r.Context(), bundleKey, *asset.ArtifactID)
			if err != nil || artifact == nil {
				writeJSONError(w, http.StatusNotFound, "artifact not found", ApiErrorTypeNotFound)
				return
			}
			if artifact.SHA512 == "" {
				writeJSONError(w, http.StatusNotFound, "artifact missing sha512 — re-upload with SHA512", ApiErrorTypeNotFound)
				return
			}

			manifest := buildElectronManifest(artifact)
			w.Header().Set("Content-Type", "application/x-yaml")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifest)
			return
		}

		// Binary download: redirect to presigned S3 URL
		artifact, err := artifacts.GetCurrentArtifactByFilename(r.Context(), bundleKey, appKey, variantKey, file)
		if err != nil || artifact == nil {
			writeJSONError(w, http.StatusNotFound, "artifact not found", ApiErrorTypeNotFound)
			return
		}

		signedURL, err := artifacts.PresignGetArtifact(r.Context(), bundleKey, *artifact)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate download URL", ApiErrorTypeServerError)
			return
		}

		http.Redirect(w, r, signedURL, http.StatusFound)
	}
}

func registerUpdateRoutes(s *Server) {
	s.router.HandleFunc("/api/v1/updates/{app_key}/{channel}/{file}",
		handleUpdateFile(s.downloadService, s.downloadService, s.downloadHosting, s.planService)).Methods("GET")
}
