package delivery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"landing-page-business-suite-api/internal/delivery"
)

// AdminDependencies supplies delivery-domain operations at the HTTP composition edge.
type AdminDependencies struct {
	BundleKey          func() string
	SettingsSnapshot   func(context.Context, string) (*delivery.StorageSettingsSnapshot, error)
	SaveSettings       func(context.Context, string, delivery.StorageSettingsUpdate) (*delivery.StorageSettingsSnapshot, error)
	TestConnection     func(context.Context, string) error
	ListArtifacts      func(context.Context, string, string, string, string, int, int) (*delivery.ListArtifactsResult, error)
	ListArtifactsByApp func(context.Context, string, string, string, int, int) (*delivery.ListArtifactsResult, error)
	PresignUpload      func(context.Context, string, delivery.PresignUploadRequest) (*delivery.PresignUploadResponse, error)
	CommitArtifact     func(context.Context, string, delivery.CommitArtifactRequest) (*delivery.Artifact, error)
	GetArtifact        func(context.Context, string, int64) (*delivery.Artifact, error)
	PresignGetArtifact func(context.Context, string, delivery.Artifact) (string, error)
	DecodeJSON         func(http.ResponseWriter, *http.Request, any) bool
	PathInt64          func(http.ResponseWriter, *http.Request, string) (int64, bool)
	WriteSuccessData   func(http.ResponseWriter, any)
	WriteSuccessSimple func(http.ResponseWriter)
	WriteError         func(http.ResponseWriter, int, string, string)
	GetManagedAsset    func(string, string, string) (*ManagedAsset, error)
	UpsertManagedAsset func(context.Context, ManagedAsset) (any, error)
	IsAssetNotFound    func(error) bool
}

// ManagedAsset is the delivery-owned representation used by artifact promotion
// endpoints. The root server adapts its persistence model at composition time.
type ManagedAsset struct {
	BundleKey           string
	AppKey              string
	Platform            string
	VariantKey          string
	ArtifactURL         string
	ArtifactSource      string
	ArtifactID          *int64
	ReleaseVersion      string
	ReleaseNotes        string
	Checksum            string
	RequiresEntitlement bool
	Metadata            map[string]any
}

type applyArtifactRequest struct {
	AppKey              string         `json:"app_key"`
	Platform            string         `json:"platform"`
	VariantKey          string         `json:"variant_key"`
	ArtifactID          int64          `json:"artifact_id"`
	ReleaseVersion      string         `json:"release_version"`
	ReleaseNotes        string         `json:"release_notes"`
	Checksum            string         `json:"checksum"`
	RequiresEntitlement *bool          `json:"requires_entitlement"`
	Metadata            map[string]any `json:"metadata"`
}

type setCurrentArtifactRequest struct {
	ArtifactID int64  `json:"artifact_id"`
	AppKey     string `json:"app_key"`
	Platform   string `json:"platform"`
}

func GetStorage(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := deps.SettingsSnapshot(r.Context(), deps.BundleKey())
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load download storage settings: %v", err), "server_error")
			return
		}
		deps.WriteSuccessData(w, map[string]any{"settings": snapshot})
	}
}

func UpdateStorage(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload delivery.StorageSettingsUpdate
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		settings, err := deps.SaveSettings(r.Context(), deps.BundleKey(), payload)
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		deps.WriteSuccessData(w, map[string]any{"settings": settings})
	}
}

func TestStorage(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.TestConnection(r.Context(), deps.BundleKey()); err != nil {
			status, kind := storageError(err)
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessSimple(w)
	}
}

func ListArtifacts(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, pageSize := pagination(r)
		result, err := deps.ListArtifacts(
			r.Context(), deps.BundleKey(), r.URL.Query().Get("query"), r.URL.Query().Get("platform"), r.URL.Query().Get("app_key"), page, pageSize,
		)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list download artifacts: %v", err), "server_error")
			return
		}
		deps.WriteSuccessData(w, result)
	}
}

func ListArtifactsByApp(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := strings.TrimSpace(r.URL.Query().Get("app_key"))
		if appKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key is required", "validation")
			return
		}
		page, pageSize := pagination(r)
		result, err := deps.ListArtifactsByApp(r.Context(), deps.BundleKey(), appKey, r.URL.Query().Get("platform"), page, pageSize)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list download artifacts: %v", err), "server_error")
			return
		}
		deps.WriteSuccessData(w, result)
	}
}

func PresignUpload(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload delivery.PresignUploadRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		response, err := deps.PresignUpload(r.Context(), deps.BundleKey(), payload)
		if err != nil {
			status, kind := storageError(err)
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessData(w, response)
	}
}

func CommitArtifact(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload delivery.CommitArtifactRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		artifact, err := deps.CommitArtifact(r.Context(), deps.BundleKey(), payload)
		if err != nil {
			status, kind := storageError(err)
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessData(w, artifact)
	}
}

func PresignGet(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt64(w, r, "artifact_id")
		if !ok {
			return
		}
		if id <= 0 {
			deps.WriteError(w, http.StatusBadRequest, "artifact_id must be a positive integer", "validation")
			return
		}
		artifact, err := deps.GetArtifact(r.Context(), deps.BundleKey(), id)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load artifact: %v", err), "server_error")
			return
		}
		if artifact == nil {
			deps.WriteError(w, http.StatusNotFound, "artifact not found", "not_found")
			return
		}
		url, err := deps.PresignGetArtifact(r.Context(), deps.BundleKey(), *artifact)
		if err != nil {
			status, kind := storageError(err)
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessData(w, map[string]string{"url": url})
	}
}

// ApplyArtifact associates a managed artifact with an app/platform release.
func ApplyArtifact(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload applyArtifactRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		appKey, platform := strings.TrimSpace(payload.AppKey), strings.TrimSpace(payload.Platform)
		if appKey == "" || platform == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key and platform are required", "validation")
			return
		}
		if payload.ArtifactID <= 0 {
			deps.WriteError(w, http.StatusBadRequest, "artifact_id is required", "validation")
			return
		}
		artifact, err := deps.GetArtifact(r.Context(), deps.BundleKey(), payload.ArtifactID)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load artifact: %v", err), "server_error")
			return
		}
		if artifact == nil {
			deps.WriteError(w, http.StatusNotFound, "artifact not found", "not_found")
			return
		}
		releaseVersion := strings.TrimSpace(payload.ReleaseVersion)
		if releaseVersion == "" {
			releaseVersion = strings.TrimSpace(artifact.ReleaseVersion)
		}
		if releaseVersion == "" {
			deps.WriteError(w, http.StatusBadRequest, "release_version is required (provide it or set one on the artifact)", "validation")
			return
		}
		requiresEntitlement := false
		if payload.RequiresEntitlement != nil {
			requiresEntitlement = *payload.RequiresEntitlement
		}
		id := payload.ArtifactID
		updated, err := deps.UpsertManagedAsset(r.Context(), ManagedAsset{BundleKey: deps.BundleKey(), AppKey: appKey, Platform: platform, VariantKey: payload.VariantKey, ArtifactSource: "managed", ArtifactID: &id, ReleaseVersion: releaseVersion, ReleaseNotes: payload.ReleaseNotes, Checksum: payload.Checksum, RequiresEntitlement: requiresEntitlement, Metadata: payload.Metadata})
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		deps.WriteSuccessData(w, updated)
	}
}

// SetArtifactCurrent promotes an artifact while preserving existing release settings.
func SetArtifactCurrent(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload setCurrentArtifactRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		appKey, platform := strings.TrimSpace(payload.AppKey), strings.TrimSpace(payload.Platform)
		if appKey == "" || platform == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key and platform are required", "validation")
			return
		}
		if payload.ArtifactID <= 0 {
			deps.WriteError(w, http.StatusBadRequest, "artifact_id is required", "validation")
			return
		}
		artifact, err := deps.GetArtifact(r.Context(), deps.BundleKey(), payload.ArtifactID)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load artifact: %v", err), "server_error")
			return
		}
		if artifact == nil {
			deps.WriteError(w, http.StatusNotFound, "artifact not found", "not_found")
			return
		}
		existing, err := deps.GetManagedAsset(deps.BundleKey(), appKey, platform)
		if err != nil && !deps.IsAssetNotFound(err) {
			deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load existing asset: %v", err), "server_error")
			return
		}
		releaseVersion, releaseNotes, requiresEntitlement := strings.TrimSpace(artifact.ReleaseVersion), "", false
		var metadata map[string]any
		if existing != nil {
			if releaseVersion == "" {
				releaseVersion = existing.ReleaseVersion
			}
			releaseNotes, requiresEntitlement, metadata = existing.ReleaseNotes, existing.RequiresEntitlement, existing.Metadata
		}
		if releaseVersion == "" {
			deps.WriteError(w, http.StatusBadRequest, "artifact has no release_version and no existing version to use", "validation")
			return
		}
		if metadata == nil {
			metadata = make(map[string]any)
		}
		if artifact.SizeBytes > 0 {
			metadata["size_mb"] = float64(artifact.SizeBytes) / (1024 * 1024)
		}
		id := payload.ArtifactID
		updated, err := deps.UpsertManagedAsset(r.Context(), ManagedAsset{BundleKey: deps.BundleKey(), AppKey: appKey, Platform: platform, ArtifactSource: "managed", ArtifactID: &id, ReleaseVersion: releaseVersion, ReleaseNotes: releaseNotes, RequiresEntitlement: requiresEntitlement, Metadata: metadata})
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}
		deps.WriteSuccessData(w, updated)
	}
}

func storageError(err error) (int, string) {
	if errors.Is(err, delivery.ErrStorageNotConfigured) {
		return http.StatusConflict, "server_error"
	}
	return http.StatusBadRequest, "validation"
}

func pagination(r *http.Request) (int, int) {
	return positiveQueryInt(r, "page", 1), positiveQueryInt(r, "page_size", 50)
}

func positiveQueryInt(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
