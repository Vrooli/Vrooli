// Package delivery owns update-feed HTTP transport. Domain persistence and
// policy live in internal/delivery; this package translates HTTP concerns only.
package delivery

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	internal "landing-page-business-suite-api/internal/delivery"
)

type UpdateAppLookup interface {
	GetApp(bundleKey, appKey string) (*internal.App, error)
}

type UpdateAssetLookup interface {
	GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*internal.Asset, error)
}

type UpdateArtifactResolver interface {
	GetArtifact(context.Context, string, int64) (*internal.Artifact, error)
	GetCurrentArtifactByFilename(context.Context, string, string, string, string) (*internal.Artifact, error)
	PresignGetArtifact(context.Context, string, internal.Artifact) (string, error)
}

type UpdateVerifier interface {
	GetArtifact(context.Context, string, int64) (*internal.Artifact, error)
	PresignGetArtifact(context.Context, string, internal.Artifact) (string, error)
	HeadArtifact(context.Context, string, internal.Artifact) error
	ReadArtifact(context.Context, string, internal.Artifact) (io.ReadCloser, int64, string, error)
}

type UpdateChannelLookup interface {
	ListChannels(bundleKey, appKey string) ([]internal.ChannelInfo, error)
}

type UpdateChannelHaltLookup interface {
	IsChannelHalted(bundleKey, appKey, variantKey string) (bool, error)
}

type UpdatePolicyLookup interface {
	GetApp(bundleKey, appKey string) (*internal.App, error)
	UpdateAppPolicy(bundleKey, appKey string, policy internal.UpdatePolicy) error
}

// UpdateDependencies holds the small edge adapters required to keep the
// delivery transport independent from the API root's mux and response package.
type UpdateDependencies struct {
	BundleKey  func() string
	PathParam  func(*http.Request, string) (string, bool)
	WriteError func(http.ResponseWriter, int, string, string)
	WriteData  func(http.ResponseWriter, any)
	DecodeJSON func(http.ResponseWriter, *http.Request, any) bool
}

func (d UpdateDependencies) appKey(r *http.Request) (string, bool) { return d.PathParam(r, "app_key") }

func RequireUpdateAPIKey(deps UpdateDependencies, apps UpdateAppLookup) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			appKey, ok := deps.appKey(r)
			if !ok || appKey == "" {
				deps.WriteError(w, http.StatusBadRequest, "app_key is required", "validation")
				return
			}
			app, err := apps.GetApp(deps.BundleKey(), appKey)
			if err != nil {
				if errors.Is(err, internal.ErrAppNotFound) {
					deps.WriteError(w, http.StatusNotFound, "app not found", "not_found")
					return
				}
				deps.WriteError(w, http.StatusInternalServerError, "failed to look up app", "server_error")
				return
			}
			if app.UpdateAPIKey != "" {
				provided := strings.TrimSpace(r.Header.Get("X-Update-Key"))
				if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(app.UpdateAPIKey)) != 1 {
					deps.WriteError(w, http.StatusForbidden, "invalid or missing update API key", "forbidden")
					return
				}
			}
			next(w, r)
		}
	}
}

func UpdateFile(deps UpdateDependencies, assets UpdateAssetLookup, artifacts UpdateArtifactResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, _ := deps.appKey(r)
		channel, ok := deps.PathParam(r, "channel")
		if !ok || channel == "" {
			deps.WriteError(w, http.StatusBadRequest, "channel is required", "validation")
			return
		}
		file, ok := deps.PathParam(r, "file")
		if !ok || file == "" {
			deps.WriteError(w, http.StatusBadRequest, "file is required", "validation")
			return
		}
		bundleKey, variantKey := deps.BundleKey(), ChannelToVariantKey(channel)
		if halted, err := channelHalted(assets, bundleKey, appKey, variantKey); err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "failed to read channel halt state", "server_error")
			return
		} else if halted {
			deps.WriteError(w, http.StatusGone, "channel is halted; no new offers are available", "channel_halted")
			return
		}
		if platform := ManifestFilenameToPlatform(file); platform != "" {
			asset, err := assets.GetAssetByVariant(bundleKey, appKey, platform, variantKey)
			if err != nil {
				if errors.Is(err, internal.ErrAssetNotFound) {
					deps.WriteError(w, http.StatusNotFound, "no asset found for platform/channel", "not_found")
				} else {
					deps.WriteError(w, http.StatusInternalServerError, "failed to look up asset", "server_error")
				}
				return
			}
			if asset.ArtifactID == nil {
				deps.WriteError(w, http.StatusNotFound, "no managed artifact linked", "not_found")
				return
			}
			artifact, err := artifacts.GetArtifact(r.Context(), bundleKey, *asset.ArtifactID)
			if err != nil || artifact == nil {
				deps.WriteError(w, http.StatusNotFound, "artifact not found", "not_found")
				return
			}
			if artifact.SHA512 == "" {
				deps.WriteError(w, http.StatusNotFound, "artifact missing sha512 — re-upload with SHA512", "not_found")
				return
			}
			w.Header().Set("Content-Type", "application/x-yaml")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(BuildElectronManifest(artifact, asset.ReleaseNotes, channel, appKey))
			return
		}
		artifact, err := artifacts.GetCurrentArtifactByFilename(r.Context(), bundleKey, appKey, variantKey, file)
		if err != nil || artifact == nil {
			deps.WriteError(w, http.StatusNotFound, "artifact not found", "not_found")
			return
		}
		signedURL, err := artifacts.PresignGetArtifact(r.Context(), bundleKey, *artifact)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "failed to generate download URL", "server_error")
			return
		}
		if err := validateDownloadRedirect(signedURL); err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "generated download URL is unsafe", "server_error")
			return
		}
		// The redirect contains a short-lived provider URL. Do not let an
		// intermediary cache that credential-bearing location after expiry.
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, signedURL, http.StatusFound)
	}
}

func validateDownloadRedirect(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("download redirect must be an absolute URL without userinfo")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("download redirect must use http or https")
	}
	return nil
}

func ChannelDiscovery(deps UpdateDependencies, channels UpdateChannelLookup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, _ := deps.appKey(r)
		result, err := channels.ListChannels(deps.BundleKey(), appKey)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "failed to list channels", "server_error")
			return
		}
		deps.WriteData(w, result)
	}
}

func VerifyUpdate(deps UpdateDependencies, assets UpdateAssetLookup, artifacts UpdateVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, _ := deps.appKey(r)
		channel, platform, expectedVersion := r.URL.Query().Get("channel"), r.URL.Query().Get("platform"), r.URL.Query().Get("expected_version")
		expectedSHA512 := strings.TrimSpace(r.URL.Query().Get("expected_sha512"))
		if channel == "" || platform == "" || expectedVersion == "" || expectedSHA512 == "" {
			deps.WriteError(w, http.StatusBadRequest, "channel, platform, expected_version, and expected_sha512 are required", "validation")
			return
		}
		bundleKey := deps.BundleKey()
		if halted, err := channelHalted(assets, bundleKey, appKey, ChannelToVariantKey(channel)); err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "failed to read channel halt state", "server_error")
			return
		} else if halted {
			deps.WriteError(w, http.StatusGone, "channel is halted; no new offers are available", "channel_halted")
			return
		}
		asset, err := assets.GetAssetByVariant(bundleKey, appKey, platform, ChannelToVariantKey(channel))
		if err != nil {
			if errors.Is(err, internal.ErrAssetNotFound) {
				deps.WriteError(w, http.StatusNotFound, "no asset found for platform/channel", "not_found")
			} else {
				deps.WriteError(w, http.StatusInternalServerError, "failed to look up asset", "server_error")
			}
			return
		}
		if asset.ArtifactID == nil {
			deps.WriteError(w, http.StatusNotFound, "no managed artifact linked", "not_found")
			return
		}
		artifact, err := artifacts.GetArtifact(r.Context(), bundleKey, *asset.ArtifactID)
		if err != nil || artifact == nil {
			deps.WriteError(w, http.StatusNotFound, "artifact not found", "not_found")
			return
		}
		versionMatch := artifact.ReleaseVersion == expectedVersion
		metadataSHA512Match := digestMatches(artifact.SHA512, expectedSHA512)
		resp := map[string]any{
			"app_key":          appKey,
			"channel":          channel,
			"platform":         platform,
			"expected_version": expectedVersion,
			"match":            versionMatch && artifact.SHA512 != "" && metadataSHA512Match,
			"actual_version":   artifact.ReleaseVersion, "actual_sha512": artifact.SHA512,
			"sha512_match": metadataSHA512Match,
		}
		if r.URL.Query().Get("deep") == "true" {
			artifactAccessible := artifacts.HeadArtifact(r.Context(), bundleKey, *artifact) == nil
			resp["artifact_accessible"] = artifactAccessible
			signedURL, err := artifacts.PresignGetArtifact(r.Context(), bundleKey, *artifact)
			presignValid := err == nil && validateDownloadRedirect(signedURL) == nil
			resp["presign_valid"] = presignValid

			bytesVerified, actualSize, actualSHA512 := false, int64(0), ""
			if artifactAccessible && presignValid {
				body, declaredSize, _, readErr := artifacts.ReadArtifact(r.Context(), bundleKey, *artifact)
				if readErr == nil && body != nil {
					actualSize, actualSHA512, readErr = digestArtifact(body, declaredSize)
					bytesVerified = readErr == nil && digestMatches(actualSHA512, artifact.SHA512) &&
						(artifact.SizeBytes <= 0 || actualSize == artifact.SizeBytes)
				}
			}
			resp["bytes_verified"] = bytesVerified
			resp["actual_size_bytes"] = actualSize
			resp["actual_sha512"] = actualSHA512
			sha512Match := bytesVerified && digestMatches(actualSHA512, expectedSHA512)
			resp["sha512_match"] = sha512Match
			resp["match"] = versionMatch && artifact.SHA512 != "" && sha512Match
		}
		deps.WriteData(w, resp)
	}
}

func channelHalted(value interface{}, bundleKey, appKey, variantKey string) (bool, error) {
	lookup, ok := value.(UpdateChannelHaltLookup)
	if !ok {
		return false, nil
	}
	return lookup.IsChannelHalted(bundleKey, appKey, variantKey)
}

func digestArtifact(body io.ReadCloser, declaredSize int64) (int64, string, error) {
	defer body.Close()
	hash := sha512.New()
	count, err := io.Copy(hash, body)
	if err != nil {
		return count, "", err
	}
	if declaredSize > 0 && count != declaredSize {
		return count, "", errors.New("artifact byte count differs from provider content length")
	}
	return count, base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

func digestMatches(actual, expected string) bool {
	actual = strings.TrimSpace(strings.TrimPrefix(actual, "sha512:"))
	expected = strings.TrimSpace(strings.TrimPrefix(expected, "sha512:"))
	if actual == "" || expected == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1 {
		return true
	}
	actualBytes, actualErr := decodeDigest(actual)
	expectedBytes, expectedErr := decodeDigest(expected)
	if actualErr == nil && expectedErr == nil {
		return subtle.ConstantTimeCompare(actualBytes, expectedBytes) == 1
	}
	return false
}

func decodeDigest(value string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	return hex.DecodeString(value)
}

func GetUpdatePolicy(deps UpdateDependencies, apps UpdatePolicyLookup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := deps.appKey(r)
		if !ok || appKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key is required", "validation")
			return
		}
		app, err := apps.GetApp(deps.BundleKey(), appKey)
		if err != nil {
			if errors.Is(err, internal.ErrAppNotFound) {
				deps.WriteError(w, http.StatusNotFound, "app not found", "not_found")
			} else {
				deps.WriteError(w, http.StatusInternalServerError, "failed to look up app", "server_error")
			}
			return
		}
		deps.WriteData(w, app.UpdatePolicy)
	}
}

func PutUpdatePolicy(deps UpdateDependencies, apps UpdatePolicyLookup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey, ok := deps.appKey(r)
		if !ok || appKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key is required", "validation")
			return
		}
		var policy internal.UpdatePolicy
		if !deps.DecodeJSON(w, r, &policy) {
			return
		}
		if policy.CheckIntervalHours < 1 {
			deps.WriteError(w, http.StatusBadRequest, "check_interval_hours must be >= 1", "validation")
			return
		}
		if !map[string]bool{"optional": true, "recommended": true, "mandatory": true}[policy.UpdateMode] {
			deps.WriteError(w, http.StatusBadRequest, "update_mode must be optional, recommended, or mandatory", "validation")
			return
		}
		if err := apps.UpdateAppPolicy(deps.BundleKey(), appKey, policy); err != nil {
			if errors.Is(err, internal.ErrAppNotFound) {
				deps.WriteError(w, http.StatusNotFound, "app not found", "not_found")
			} else {
				deps.WriteError(w, http.StatusInternalServerError, "failed to update policy", "server_error")
			}
			return
		}
		deps.WriteData(w, policy)
	}
}
