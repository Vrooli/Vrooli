package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"landing-page-business-suite-api/internal/delivery"
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
	GetArtifact(ctx context.Context, bundleKey string, id int64) (*delivery.Artifact, error)
	GetCurrentArtifactByFilename(ctx context.Context, bundleKey, appKey, variantKey, filename string) (*delivery.Artifact, error)
	PresignGetArtifact(ctx context.Context, bundleKey string, artifact delivery.Artifact) (string, error)
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

// requireUpdateAPIKey is middleware that validates the X-Update-Key header against
// the app's update_api_key using constant-time comparison. If the app has no key set,
// the request passes through. The validated *DownloadApp is stored in r.Context().
func requireUpdateAPIKey(apps updateAppLookup, bundles updateBundleKeyProvider) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			appKey, ok := getPathParam(r, "app_key")
			if !ok || appKey == "" {
				writeJSONError(w, http.StatusBadRequest, "app_key is required", ApiErrorTypeValidation)
				return
			}

			app, err := apps.GetApp(bundles.BundleKey(), appKey)
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
				if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(app.UpdateAPIKey)) != 1 {
					writeJSONError(w, http.StatusForbidden, "invalid or missing update API key", ApiErrorTypeForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), updateAppContextKey, app)
			next(w, r.WithContext(ctx))
		}
	}
}

const updateAppContextKey contextKey = "updateApp"

// channelToVariantKey maps an update channel name to a download_assets variant_key.
func channelToVariantKey(channel string) string {
	if channel == "stable" || channel == "" {
		return "default"
	}
	return channel
}

// buildElectronManifest generates a YAML manifest in the format electron-updater expects.
// When releaseNotes is non-empty, it is included as a plain text passthrough.
func buildElectronManifest(artifact *delivery.Artifact, releaseNotes string) []byte {
	base := fmt.Sprintf(
		"version: %s\npath: %s\nsha512: %s\nreleaseDate: %s\nfiles:\n  - url: %s\n    sha512: %s\n    size: %d\n",
		artifact.ReleaseVersion,
		artifact.OriginalFilename,
		artifact.SHA512,
		artifact.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		artifact.OriginalFilename,
		artifact.SHA512,
		artifact.SizeBytes,
	)
	if releaseNotes != "" {
		base += fmt.Sprintf("releaseNotes: %s\n", releaseNotes)
	}
	return []byte(base)
}

// handleUpdateFile serves electron-updater manifest files and binary download redirects.
// Route: GET /api/v1/updates/{app_key}/{channel}/{file}
// Expects requireUpdateAPIKey middleware to have validated the app and API key.
func handleUpdateFile(assets updateAssetLookup, artifacts updateArtifactResolver, bundles updateBundleKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, _ := getPathParam(r, "app_key")
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

			manifest := buildElectronManifest(artifact, asset.ReleaseNotes)
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

// --- Channel discovery types ---

type ChannelInfo = delivery.ChannelInfo

// channelDiscoveryLookup retrieves available channels for an app.
type channelDiscoveryLookup interface {
	ListChannels(bundleKey, appKey string) ([]ChannelInfo, error)
}

// handleChannelDiscovery returns available channels with latest version per platform.
// Route: GET /api/v1/updates/{app_key}/channels
// Gated by requireUpdateAPIKey middleware.
func handleChannelDiscovery(channels channelDiscoveryLookup, bundles updateBundleKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, _ := getPathParam(r, "app_key")
		result, err := channels.ListChannels(bundles.BundleKey(), appKey)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list channels", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, result)
	}
}

// updateVerifyLookup provides the asset and artifact data needed for verification.
type updateVerifyLookup interface {
	GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*DownloadAsset, error)
}

// updateVerifyArtifactResolver provides artifact data and S3 operations for verification.
type updateVerifyArtifactResolver interface {
	GetArtifact(ctx context.Context, bundleKey string, id int64) (*delivery.Artifact, error)
	PresignGetArtifact(ctx context.Context, bundleKey string, artifact delivery.Artifact) (string, error)
	HeadArtifact(ctx context.Context, bundleKey string, artifact delivery.Artifact) error
}

// handleUpdateVerify confirms update endpoint correctness.
// Route: GET /api/v1/updates/{app_key}/verify?channel=X&platform=Y&expected_version=Z&deep=false
// Gated by requireUpdateAPIKey middleware.
func handleUpdateVerify(assets updateVerifyLookup, artifacts updateVerifyArtifactResolver, bundles updateBundleKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, _ := getPathParam(r, "app_key")
		channel := r.URL.Query().Get("channel")
		platform := r.URL.Query().Get("platform")
		expectedVersion := r.URL.Query().Get("expected_version")
		deep := r.URL.Query().Get("deep") == "true"

		if channel == "" || platform == "" || expectedVersion == "" {
			writeJSONError(w, http.StatusBadRequest, "channel, platform, and expected_version are required", ApiErrorTypeValidation)
			return
		}

		bundleKey := bundles.BundleKey()
		variantKey := channelToVariantKey(channel)

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

		resp := map[string]interface{}{
			"match":          artifact.ReleaseVersion == expectedVersion && artifact.SHA512 != "",
			"actual_version": artifact.ReleaseVersion,
			"actual_sha512":  artifact.SHA512,
		}

		if deep {
			headErr := artifacts.HeadArtifact(r.Context(), bundleKey, *artifact)
			resp["artifact_accessible"] = headErr == nil

			_, presignErr := artifacts.PresignGetArtifact(r.Context(), bundleKey, *artifact)
			resp["presign_valid"] = presignErr == nil
		}

		writeJSONSuccessData(w, resp)
	}
}

// --- Update policy types ---

type UpdatePolicy = delivery.UpdatePolicy

// updatePolicyLookup retrieves and updates app update policies.
type updatePolicyLookup interface {
	GetApp(bundleKey, appKey string) (*DownloadApp, error)
	UpdateAppPolicy(bundleKey, appKey string, policy UpdatePolicy) error
}

// handleGetUpdatePolicy returns the update policy for an app.
// Route: GET /api/v1/admin/download-apps/{app_key}/update-policy
func handleGetUpdatePolicy(apps updatePolicyLookup, bundles updateBundleKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := getPathParam(r, "app_key")
		if !ok || appKey == "" {
			writeJSONError(w, http.StatusBadRequest, "app_key is required", ApiErrorTypeValidation)
			return
		}

		app, err := apps.GetApp(bundles.BundleKey(), appKey)
		if err != nil {
			if errors.Is(err, ErrDownloadAppNotFound) {
				writeJSONError(w, http.StatusNotFound, "app not found", ApiErrorTypeNotFound)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to look up app", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, app.UpdatePolicy)
	}
}

// handlePutUpdatePolicy updates the update policy for an app.
// Route: PUT /api/v1/admin/download-apps/{app_key}/update-policy
func handlePutUpdatePolicy(apps updatePolicyLookup, bundles updateBundleKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := getPathParam(r, "app_key")
		if !ok || appKey == "" {
			writeJSONError(w, http.StatusBadRequest, "app_key is required", ApiErrorTypeValidation)
			return
		}

		var policy UpdatePolicy
		if !decodeJSONBody(w, r, &policy) {
			return
		}

		if policy.CheckIntervalHours < 1 {
			writeJSONError(w, http.StatusBadRequest, "check_interval_hours must be >= 1", ApiErrorTypeValidation)
			return
		}
		validModes := map[string]bool{"optional": true, "recommended": true, "mandatory": true}
		if !validModes[policy.UpdateMode] {
			writeJSONError(w, http.StatusBadRequest, "update_mode must be optional, recommended, or mandatory", ApiErrorTypeValidation)
			return
		}

		if err := apps.UpdateAppPolicy(bundles.BundleKey(), appKey, policy); err != nil {
			if errors.Is(err, ErrDownloadAppNotFound) {
				writeJSONError(w, http.StatusNotFound, "app not found", ApiErrorTypeNotFound)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to update policy", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, policy)
	}
}

func registerUpdateRoutes(s *Server) {
	updateAPIKeyMiddleware := requireUpdateAPIKey(s.downloadService, s.planService)

	s.router.HandleFunc("/api/v1/updates/{app_key}/{channel}/{file}",
		updateAPIKeyMiddleware(handleUpdateFile(s.downloadService, s.downloadHosting, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/updates/{app_key}/channels",
		updateAPIKeyMiddleware(handleChannelDiscovery(s.downloadService, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/updates/{app_key}/verify",
		updateAPIKeyMiddleware(handleUpdateVerify(s.downloadService, s.downloadHosting, s.planService))).Methods("GET")

	// Update policy admin endpoints
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}/update-policy",
		s.requireAdmin(handleGetUpdatePolicy(s.downloadService, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}/update-policy",
		s.requireAdmin(handlePutUpdatePolicy(s.downloadService, s.planService))).Methods("PUT")
}
