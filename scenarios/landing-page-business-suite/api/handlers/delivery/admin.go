package delivery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	PromoteChannel     func(context.Context, delivery.ChannelPromotionRequest) (*delivery.ChannelRevision, error)
	GetChannelHead     func(string, string, string) (*delivery.ChannelHead, error)
	SetChannelHalt     func(context.Context, delivery.ChannelHaltRequest) (*delivery.ChannelHalt, error)
	RecoverChannel     func(context.Context, delivery.ChannelRecoveryRequest) (*delivery.ChannelRecovery, error)
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

// PromoteChannel makes one complete immutable artifact set visible using the
// caller-provided predecessor revision. It is the publication boundary for
// update feeds; applying an asset row alone never advances this head.
func PromoteChannel(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload delivery.ChannelPromotionRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		payload.BundleKey = deps.BundleKey()
		if strings.TrimSpace(payload.AppKey) == "" || len(payload.ArtifactIDs) == 0 {
			deps.WriteError(w, http.StatusBadRequest, "app_key and artifact_ids are required", "validation")
			return
		}
		if deps.PromoteChannel == nil {
			deps.WriteError(w, http.StatusServiceUnavailable, "channel promotion is not configured", "server_error")
			return
		}
		revision, err := deps.PromoteChannel(r.Context(), payload)
		if err != nil {
			status, kind := storageError(err)
			if errors.Is(err, delivery.ErrChannelRevisionConflict) {
				status, kind = http.StatusConflict, "conflict"
			}
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessData(w, revision)
	}
}

func GetChannelHead(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := strings.TrimSpace(r.URL.Query().Get("app_key"))
		variantKey := strings.TrimSpace(r.URL.Query().Get("variant_key"))
		if appKey == "" || variantKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key and variant_key are required", "validation")
			return
		}
		if deps.GetChannelHead == nil {
			deps.WriteError(w, http.StatusServiceUnavailable, "channel head lookup is not configured", "server_error")
			return
		}
		head, err := deps.GetChannelHead(deps.BundleKey(), appKey, variantKey)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, err.Error(), "server_error")
			return
		}
		if head == nil {
			deps.WriteError(w, http.StatusNotFound, "channel head not found", "not_found")
			return
		}
		deps.WriteSuccessData(w, head)
	}
}

func SetChannelHalt(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.SetChannelHalt == nil {
			deps.WriteError(w, http.StatusNotImplemented, "channel halt is not configured", "unavailable")
			return
		}
		var payload delivery.ChannelHaltRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		payload.BundleKey = deps.BundleKey()
		if strings.TrimSpace(payload.AppKey) == "" || payload.ExpectedRevision < 0 {
			deps.WriteError(w, http.StatusBadRequest, "app_key and non-negative expected_revision are required", "validation")
			return
		}
		if payload.DryRun {
			head, err := deps.GetChannelHead(payload.BundleKey, payload.AppKey, payload.VariantKey)
			if err != nil {
				deps.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("read channel head: %v", err), "server_error")
				return
			}
			current := int64(0)
			if head != nil {
				current = head.Revision
			}
			if current != payload.ExpectedRevision {
				deps.WriteError(w, http.StatusConflict, fmt.Sprintf("channel revision conflict: expected %d, current %d", payload.ExpectedRevision, current), "conflict")
				return
			}
			now := time.Now().UTC()
			deps.WriteSuccessData(w, &delivery.ChannelHalt{BundleKey: payload.BundleKey, AppKey: payload.AppKey, VariantKey: payload.VariantKey, Revision: current, Halted: payload.Halted, Reason: payload.Reason, Outcome: "preview", Health: "unknown", DryRun: true, UpdatedAt: now, ObservedAt: now})
			return
		}
		halt, err := deps.SetChannelHalt(r.Context(), payload)
		if err != nil {
			status, kind := http.StatusConflict, "conflict"
			if strings.Contains(err.Error(), "required") {
				status, kind = http.StatusBadRequest, "validation"
			}
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessData(w, halt)
	}
}

// RecoverChannel exposes owner-specific withdrawal, predecessor-bound
// rollback, and compatibility-qualified forward repair. The owner returns a
// receipt describing the channel effect; installed clients remain a separate
// observation.
func RecoverChannel(deps AdminDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.RecoverChannel == nil {
			deps.WriteError(w, http.StatusNotImplemented, "channel recovery is not configured", "unavailable")
			return
		}
		var payload delivery.ChannelRecoveryRequest
		if !deps.DecodeJSON(w, r, &payload) {
			return
		}
		payload.BundleKey = deps.BundleKey()
		if strings.TrimSpace(payload.AppKey) == "" || payload.ExpectedRevision < 0 || strings.TrimSpace(payload.Action) == "" {
			deps.WriteError(w, http.StatusBadRequest, "app_key, action, and non-negative expected_revision are required", "validation")
			return
		}
		if !payload.DryRun && strings.TrimSpace(payload.Reason) == "" {
			deps.WriteError(w, http.StatusPreconditionFailed, "reason is required for channel recovery", "confirmation_required")
			return
		}
		if payload.DryRun {
			receipt, err := deps.RecoverChannel(r.Context(), payload)
			if err != nil {
				deps.WriteError(w, http.StatusConflict, err.Error(), "conflict")
				return
			}
			deps.WriteSuccessData(w, receipt)
			return
		}
		receipt, err := deps.RecoverChannel(r.Context(), payload)
		if err != nil {
			status, kind := http.StatusConflict, "conflict"
			if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "requires") || strings.Contains(err.Error(), "unsupported") {
				status, kind = http.StatusBadRequest, "validation"
			}
			deps.WriteError(w, status, err.Error(), kind)
			return
		}
		deps.WriteSuccessData(w, receipt)
	}
}

func storageError(err error) (int, string) {
	if errors.Is(err, delivery.ErrStorageNotConfigured) {
		return http.StatusConflict, "server_error"
	}
	if errors.Is(err, delivery.ErrImmutableArtifactConflict) || errors.Is(err, delivery.ErrChannelRevisionConflict) {
		return http.StatusConflict, "conflict"
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
