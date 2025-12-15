package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func handleAdminGetDownloadStorage(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := hosting.SettingsSnapshot(r.Context(), plans.BundleKey())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load download storage settings: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]interface{}{
			"settings": snapshot,
		})
	}
}

func handleAdminUpdateDownloadStorage(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload DownloadStorageSettingsUpdate
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		settings, err := hosting.SaveSettings(r.Context(), plans.BundleKey(), payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, map[string]interface{}{
			"settings": settings,
		})
	}
}

func handleAdminTestDownloadStorage(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := hosting.TestConnection(r.Context(), plans.BundleKey()); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, map[string]bool{"success": true})
	}
}

func handleAdminListDownloadArtifacts(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		platform := r.URL.Query().Get("platform")

		page := 1
		if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				page = parsed
			}
		}
		pageSize := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				pageSize = parsed
			}
		}

		result, err := hosting.ListArtifacts(r.Context(), plans.BundleKey(), query, platform, page, pageSize)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list download artifacts: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, result)
	}
}

func handleAdminPresignUploadDownloadArtifact(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload PresignUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		resp, err := hosting.PresignUpload(r.Context(), plans.BundleKey(), payload)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, resp)
	}
}

func handleAdminCommitDownloadArtifact(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload CommitArtifactRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		artifact, err := hosting.CommitArtifact(r.Context(), plans.BundleKey(), payload)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, artifact)
	}
}

func handleAdminPresignGetDownloadArtifact(hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		raw := strings.TrimSpace(vars["artifact_id"])
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "artifact_id path parameter must be a positive integer", http.StatusBadRequest)
			return
		}

		artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), id)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load artifact: %v", err), http.StatusInternalServerError)
			return
		}
		if artifact == nil {
			http.Error(w, "artifact not found", http.StatusNotFound)
			return
		}

		url, err := hosting.PresignGetArtifact(r.Context(), plans.BundleKey(), *artifact)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrDownloadStorageNotConfigured) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, map[string]string{
			"url": url,
		})
	}
}

func handleAdminApplyDownloadArtifact(downloads *DownloadService, hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			AppKey              string                 `json:"app_key"`
			Platform            string                 `json:"platform"`
			ArtifactID          int64                  `json:"artifact_id"`
			ReleaseVersion      string                 `json:"release_version"`
			ReleaseNotes        string                 `json:"release_notes"`
			Checksum            string                 `json:"checksum"`
			RequiresEntitlement *bool                  `json:"requires_entitlement"`
			Metadata            map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		appKey := strings.TrimSpace(payload.AppKey)
		platform := strings.TrimSpace(payload.Platform)
		if appKey == "" || platform == "" {
			http.Error(w, "app_key and platform are required", http.StatusBadRequest)
			return
		}
		if payload.ArtifactID <= 0 {
			http.Error(w, "artifact_id is required", http.StatusBadRequest)
			return
		}

		artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), payload.ArtifactID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load artifact: %v", err), http.StatusInternalServerError)
			return
		}
		if artifact == nil {
			http.Error(w, "artifact not found", http.StatusNotFound)
			return
		}

		releaseVersion := strings.TrimSpace(payload.ReleaseVersion)
		if releaseVersion == "" {
			releaseVersion = strings.TrimSpace(artifact.ReleaseVersion)
		}
		if releaseVersion == "" {
			http.Error(w, "release_version is required (provide it or set one on the artifact)", http.StatusBadRequest)
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, updated)
	}
}
