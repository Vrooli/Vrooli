package bundle

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
)

// HandleListBundles returns a handler that lists all stored bundles.
// GET /api/v1/bundles
func HandleListBundles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundlesDir, err := GetLocalBundlesDir()
		if err != nil {
			WriteRepoNotFoundError(w, err)
			return
		}

		bundles, err := ListBundles(bundlesDir)
		if err != nil {
			WriteBundlesDirError(w, "list bundles", err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"bundles":   bundles,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// HandleBundleStats returns a handler that returns aggregate statistics about stored bundles.
// GET /api/v1/bundles/stats
func HandleBundleStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundlesDir, err := GetLocalBundlesDir()
		if err != nil {
			WriteRepoNotFoundError(w, err)
			return
		}

		stats, err := GetBundleStats(bundlesDir)
		if err != nil {
			WriteBundlesDirError(w, "get bundle statistics", err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, domain.BundleStatsResponse{
			Stats:     stats,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// HandleDeleteBundle returns a handler that deletes a single bundle by SHA256 hash.
// DELETE /api/v1/bundles/{sha256}
func HandleDeleteBundle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sha256Hash := strings.TrimPrefix(r.URL.Path, "/api/v1/bundles/")
		if !ValidateSHA256Hash(w, sha256Hash) {
			return
		}

		bundlesDir, err := GetLocalBundlesDir()
		if err != nil {
			WriteRepoNotFoundError(w, err)
			return
		}

		freedBytes, err := DeleteBundle(bundlesDir, sha256Hash)
		if err != nil {
			WriteBundlesDirError(w, "delete bundle", err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, domain.BundleDeleteResponse{
			OK:         true,
			FreedBytes: freedBytes,
			Message:    BundleDeleteMessage(freedBytes),
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// HandleBundleCleanup returns a handler that removes old bundles from the
// local store. Target release retention is the deployment-scoped GC through
// the target owner; a clean_vps request is refused with that next action.
// POST /api/v1/bundles/cleanup
func HandleBundleCleanup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.BundleCleanupRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		if req.CleanVPS {
			apierrors.Write(w, apierrors.New(apierrors.CodeUnsupportedCapability, "target release retention runs through the deployment's target owner, not a host-named bundle sweep").
				WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "gc", Reference: "/api/v1/deployments/{id}/bundles/vps/gc", Label: "Garbage-collect the deployment's target release store"}))
			return
		}
		ApplyBundleCleanupDefaults(&req)

		bundlesDir, err := GetLocalBundlesDir()
		if err != nil {
			WriteRepoNotFoundError(w, err)
			return
		}

		deleted, freedBytes, err := CleanupLocalBundles(bundlesDir, req.ScenarioID, req.KeepLatest)
		if err != nil {
			WriteBundlesDirError(w, "clean up local bundles", err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, domain.BundleCleanupResponse{
			OK:              true,
			LocalDeleted:    deleted,
			LocalFreedBytes: freedBytes,
			Message:         BuildCleanupMessage(deleted, freedBytes),
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ValidateSHA256Hash validates the SHA256 hash format.
// Returns false and writes an error response if validation fails.
func ValidateSHA256Hash(w http.ResponseWriter, sha256Hash string) bool {
	if sha256Hash == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_sha256",
			Message: "SHA256 hash is required in the URL path",
			Hint:    "Use DELETE /api/v1/bundles/{sha256}",
		})
		return false
	}
	if len(sha256Hash) != 64 {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_sha256",
			Message: "Invalid SHA256 hash format",
			Hint:    "SHA256 hash must be exactly 64 hexadecimal characters",
		})
		return false
	}
	return true
}

// ApplyBundleCleanupDefaults sets default values for bundle cleanup request.
func ApplyBundleCleanupDefaults(req *domain.BundleCleanupRequest) {
	if req.KeepLatest <= 0 {
		req.KeepLatest = 3
	}
}

// CleanupLocalBundles deletes old bundles from the local filesystem.
func CleanupLocalBundles(bundlesDir, scenarioID string, keepLatest int) ([]domain.BundleInfo, int64, error) {
	if scenarioID != "" {
		return DeleteBundlesForScenario(bundlesDir, scenarioID, keepLatest)
	}
	return DeleteAllOldBundles(bundlesDir, keepLatest)
}

// BuildCleanupMessage creates a human-readable summary of the local cleanup.
func BuildCleanupMessage(localDeleted []domain.BundleInfo, localFreed int64) string {
	if len(localDeleted) == 0 {
		return "No bundles needed cleanup"
	}
	return fmt.Sprintf("Deleted %d local bundles (%s)", len(localDeleted), FormatBytes(localFreed/1024))
}

// FormatBytes formats a byte count (in KB) as a human-readable string.
func FormatBytes(kb int64) string {
	if kb < 1024 {
		return fmt.Sprintf("%d KB", kb)
	}
	if kb < 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	}
	return fmt.Sprintf("%.2f GB", float64(kb)/(1024*1024))
}

// BundleDeleteMessage creates a human-readable message for bundle deletion.
func BundleDeleteMessage(freedBytes int64) string {
	if freedBytes > 0 {
		return fmt.Sprintf("Deleted bundle, freed %s", FormatBytes(freedBytes/1024))
	}
	return "Bundle not found or already deleted"
}

// WriteRepoNotFoundError writes a standardized error response for repo not found.
func WriteRepoNotFoundError(w http.ResponseWriter, err error) {
	httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
		Code:    "repo_not_found",
		Message: "Could not find repository root",
		Hint:    err.Error(),
	})
}

// WriteBundlesDirError writes a standardized error response for bundles directory errors.
func WriteBundlesDirError(w http.ResponseWriter, operation string, err error) {
	httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
		Code:    "bundles_error",
		Message: fmt.Sprintf("Failed to %s", operation),
		Hint:    err.Error(),
	})
}
