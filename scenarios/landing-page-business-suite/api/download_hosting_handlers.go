package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func handleAdminGetDownloadStorage(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := hosting.SettingsSnapshot(r.Context(), plans.BundleKey())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load download storage settings: %v", err), ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, map[string]interface{}{
			"settings": snapshot,
		})
	}
}

func handleAdminUpdateDownloadStorage(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload DownloadStorageSettingsUpdate
		if !decodeJSONBody(w, r, &payload) {
			return
		}

		settings, err := hosting.SaveSettings(r.Context(), plans.BundleKey(), payload)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, map[string]interface{}{
			"settings": settings,
		})
	}
}

func handleAdminTestDownloadStorage(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := hosting.TestConnection(r.Context(), plans.BundleKey()); err != nil {
			status := http.StatusBadRequest
			errType := ApiErrorTypeValidation
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
				errType = ApiErrorTypeServerError
			}
			writeJSONError(w, status, err.Error(), errType)
			return
		}
		writeJSONSuccessSimple(w)
	}
}

func handleAdminListDownloadArtifacts(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := getQueryParam(r, "query")
		platform := getQueryParam(r, "platform")
		appKey := getQueryParam(r, "app_key")

		page := 1
		if raw := strings.TrimSpace(getQueryParam(r, "page")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				page = parsed
			}
		}
		pageSize := 50
		if raw := strings.TrimSpace(getQueryParam(r, "page_size")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				pageSize = parsed
			}
		}

		result, err := hosting.ListArtifacts(r.Context(), plans.BundleKey(), query, platform, appKey, page, pageSize)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list download artifacts: %v", err), ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, result)
	}
}

func handleAdminListDownloadArtifactsByApp(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := getQueryParam(r, "app_key")
		platform := getQueryParam(r, "platform")

		if strings.TrimSpace(appKey) == "" {
			writeJSONError(w, http.StatusBadRequest, "app_key is required", ApiErrorTypeValidation)
			return
		}

		page := 1
		if raw := strings.TrimSpace(getQueryParam(r, "page")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				page = parsed
			}
		}
		pageSize := 50
		if raw := strings.TrimSpace(getQueryParam(r, "page_size")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				pageSize = parsed
			}
		}

		result, err := hosting.ListArtifactsByApp(r.Context(), plans.BundleKey(), appKey, platform, page, pageSize)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list download artifacts: %v", err), ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, result)
	}
}

func handleAdminPresignUploadDownloadArtifact(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload PresignUploadRequest
		if !decodeJSONBody(w, r, &payload) {
			return
		}

		resp, err := hosting.PresignUpload(r.Context(), plans.BundleKey(), payload)
		if err != nil {
			status := http.StatusBadRequest
			errType := ApiErrorTypeValidation
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
				errType = ApiErrorTypeServerError
			}
			writeJSONError(w, status, err.Error(), errType)
			return
		}

		writeJSONSuccessData(w, resp)
	}
}

func handleAdminCommitDownloadArtifact(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload CommitArtifactRequest
		if !decodeJSONBody(w, r, &payload) {
			return
		}

		artifact, err := hosting.CommitArtifact(r.Context(), plans.BundleKey(), payload)
		if err != nil {
			status := http.StatusBadRequest
			errType := ApiErrorTypeValidation
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
				errType = ApiErrorTypeServerError
			}
			writeJSONError(w, status, err.Error(), errType)
			return
		}

		writeJSONSuccessData(w, artifact)
	}
}

func handleAdminPresignGetDownloadArtifact(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "artifact_id")
		if !ok {
			return
		}
		if id <= 0 {
			writeJSONError(w, http.StatusBadRequest, "artifact_id must be a positive integer", ApiErrorTypeValidation)
			return
		}

		artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load artifact: %v", err), ApiErrorTypeServerError)
			return
		}
		if artifact == nil {
			writeJSONError(w, http.StatusNotFound, "artifact not found", ApiErrorTypeNotFound)
			return
		}

		url, err := hosting.PresignGetArtifact(r.Context(), plans.BundleKey(), *artifact)
		if err != nil {
			status := http.StatusBadRequest
			errType := ApiErrorTypeValidation
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
				errType = ApiErrorTypeServerError
			}
			writeJSONError(w, status, err.Error(), errType)
			return
		}

		writeJSONSuccessData(w, map[string]string{
			"url": url,
		})
	}
}

func handleAdminApplyDownloadArtifact(downloads *DownloadService, hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			AppKey              string                 `json:"app_key"`
			Platform            string                 `json:"platform"`
			VariantKey          string                 `json:"variant_key"`
			ArtifactID          int64                  `json:"artifact_id"`
			ReleaseVersion      string                 `json:"release_version"`
			ReleaseNotes        string                 `json:"release_notes"`
			Checksum            string                 `json:"checksum"`
			RequiresEntitlement *bool                  `json:"requires_entitlement"`
			Metadata            map[string]interface{} `json:"metadata"`
		}
		if !decodeJSONBody(w, r, &payload) {
			return
		}

		appKey := strings.TrimSpace(payload.AppKey)
		platform := strings.TrimSpace(payload.Platform)
		if appKey == "" || platform == "" {
			writeJSONError(w, http.StatusBadRequest, "app_key and platform are required", ApiErrorTypeValidation)
			return
		}
		if payload.ArtifactID <= 0 {
			writeJSONError(w, http.StatusBadRequest, "artifact_id is required", ApiErrorTypeValidation)
			return
		}

		artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), payload.ArtifactID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load artifact: %v", err), ApiErrorTypeServerError)
			return
		}
		if artifact == nil {
			writeJSONError(w, http.StatusNotFound, "artifact not found", ApiErrorTypeNotFound)
			return
		}

		releaseVersion := strings.TrimSpace(payload.ReleaseVersion)
		if releaseVersion == "" {
			releaseVersion = strings.TrimSpace(artifact.ReleaseVersion)
		}
		if releaseVersion == "" {
			writeJSONError(w, http.StatusBadRequest, "release_version is required (provide it or set one on the artifact)", ApiErrorTypeValidation)
			return
		}

		requiresEntitlement := false
		if payload.RequiresEntitlement != nil {
			requiresEntitlement = *payload.RequiresEntitlement
		}

		id := payload.ArtifactID
		updated, err := downloads.UpsertAsset(r.Context(), DownloadAsset{
			BundleKey:           plans.BundleKey(),
			AppKey:              appKey,
			Platform:            platform,
			VariantKey:          payload.VariantKey,
			ArtifactURL:         "",
			ArtifactSource:      "managed",
			ArtifactID:          &id,
			ReleaseVersion:      releaseVersion,
			ReleaseNotes:        payload.ReleaseNotes,
			Checksum:            payload.Checksum,
			RequiresEntitlement: requiresEntitlement,
			Metadata:            payload.Metadata,
		})
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, updated)
	}
}

// handleAdminSetArtifactAsCurrent promotes an artifact to be the current version for an app/platform.
func handleAdminSetArtifactAsCurrent(downloads *DownloadService, hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ArtifactID int64  `json:"artifact_id"`
			AppKey     string `json:"app_key"`
			Platform   string `json:"platform"`
		}
		if !decodeJSONBody(w, r, &payload) {
			return
		}

		appKey := strings.TrimSpace(payload.AppKey)
		platform := strings.TrimSpace(payload.Platform)
		if appKey == "" || platform == "" {
			writeJSONError(w, http.StatusBadRequest, "app_key and platform are required", ApiErrorTypeValidation)
			return
		}
		if payload.ArtifactID <= 0 {
			writeJSONError(w, http.StatusBadRequest, "artifact_id is required", ApiErrorTypeValidation)
			return
		}

		artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), payload.ArtifactID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load artifact: %v", err), ApiErrorTypeServerError)
			return
		}
		if artifact == nil {
			writeJSONError(w, http.StatusNotFound, "artifact not found", ApiErrorTypeNotFound)
			return
		}

		// Get the existing asset to preserve its settings (entitlement, release notes, etc.)
		existingAsset, err := downloads.GetAsset(plans.BundleKey(), appKey, platform)
		if err != nil && !errors.Is(err, ErrDownloadNotFound) {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load existing asset: %v", err), ApiErrorTypeServerError)
			return
		}

		// Use artifact's release version, fall back to existing asset's version
		releaseVersion := strings.TrimSpace(artifact.ReleaseVersion)
		releaseNotes := ""
		requiresEntitlement := false
		var metadata map[string]interface{}

		if existingAsset != nil {
			if releaseVersion == "" {
				releaseVersion = existingAsset.ReleaseVersion
			}
			releaseNotes = existingAsset.ReleaseNotes
			requiresEntitlement = existingAsset.RequiresEntitlement
			metadata = existingAsset.Metadata
		}

		if releaseVersion == "" {
			writeJSONError(w, http.StatusBadRequest, "artifact has no release_version and no existing version to use", ApiErrorTypeValidation)
			return
		}

		// Add size_bytes from artifact to metadata
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		if artifact.SizeBytes > 0 {
			metadata["size_mb"] = float64(artifact.SizeBytes) / (1024 * 1024)
		}

		id := payload.ArtifactID
		updated, err := downloads.UpsertAsset(r.Context(), DownloadAsset{
			BundleKey:           plans.BundleKey(),
			AppKey:              appKey,
			Platform:            platform,
			ArtifactURL:         "",
			ArtifactSource:      "managed",
			ArtifactID:          &id,
			ReleaseVersion:      releaseVersion,
			ReleaseNotes:        releaseNotes,
			RequiresEntitlement: requiresEntitlement,
			Metadata:            metadata,
		})
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		writeJSONSuccessData(w, updated)
	}
}
