package bundle

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/ssh"
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

// HandleBundleCleanup returns a handler that removes old bundles from local storage and optionally from VPS.
// POST /api/v1/bundles/cleanup
func HandleBundleCleanup(sshRunner ssh.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.BundleCleanupRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}

		// Apply defaults
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

		resp := domain.BundleCleanupResponse{
			OK:              true,
			LocalDeleted:    deleted,
			LocalFreedBytes: freedBytes,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		}

		// VPS cleanup (if requested and SSH config provided)
		if req.CleanVPS && req.Host != "" && req.KeyPath != "" && req.Workdir != "" {
			vpsDeleted, vpsFreed, vpsErr := CleanupVPSBundles(r.Context(), sshRunner, req)
			resp.VPSDeleted = vpsDeleted
			resp.VPSFreedBytes = vpsFreed
			if vpsErr != nil {
				resp.VPSError = vpsErr.Error()
				resp.OK = false // Partial failure
			}
		}

		resp.Message = BuildCleanupMessage(deleted, freedBytes, resp.VPSDeleted, resp.VPSFreedBytes)
		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

// HandleListVPSBundles returns a handler that lists bundles stored on the VPS.
// POST /api/v1/bundles/vps/list
func HandleListVPSBundles(sshRunner ssh.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.VPSBundleListRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}

		if !ValidateVPSBundleRequest(w, req.Host, req.KeyPath, req.Workdir) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		bundles, totalSize, err := ListVPSBundlesSSH(ctx, sshRunner, req)
		if err != nil {
			httputil.WriteJSON(w, http.StatusOK, domain.VPSBundleListResponse{
				OK:        false,
				Bundles:   []domain.VPSBundleInfo{},
				Error:     fmt.Sprintf("Failed to connect or list bundles: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, domain.VPSBundleListResponse{
			OK:             true,
			Bundles:        bundles,
			TotalSizeBytes: totalSize,
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// HandleDeleteVPSBundle returns a handler that deletes a single bundle from the VPS.
// POST /api/v1/bundles/vps/delete
func HandleDeleteVPSBundle(sshRunner ssh.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.VPSBundleDeleteRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}

		// Validate required fields
		if req.Host == "" || req.KeyPath == "" || req.Workdir == "" || req.Filename == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "missing_fields",
				Message: "host, key_path, workdir, and filename are required",
			})
			return
		}

		// Validate filename format to prevent path traversal
		if !strings.HasPrefix(req.Filename, "mini-vrooli_") || !strings.HasSuffix(req.Filename, ".tar.gz") {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_filename",
				Message: "Invalid bundle filename format",
			})
			return
		}

		cfg := ssh.NewConfig(req.Host, req.Port, req.User, req.KeyPath)

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		bundlePath := ssh.SafeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles", req.Filename)

		// Get file size before deleting
		sizeCmd := fmt.Sprintf("stat -c %%s %s 2>/dev/null || echo 0", ssh.QuoteSingle(bundlePath))
		sizeRes, _ := sshRunner.Run(ctx, cfg, sizeCmd)
		sizeBytes := ParseBytes(strings.TrimSpace(sizeRes.Stdout))

		// Delete the file
		deleteCmd := fmt.Sprintf("rm -f %s", ssh.QuoteSingle(bundlePath))
		_, err := sshRunner.Run(ctx, cfg, deleteCmd)
		if err != nil {
			httputil.WriteJSON(w, http.StatusOK, domain.VPSBundleDeleteResponse{
				OK:        false,
				Error:     fmt.Sprintf("Failed to delete bundle: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		var message string
		if sizeBytes > 0 {
			message = fmt.Sprintf("Deleted %s, freed %s", req.Filename, FormatBytes(sizeBytes/1024))
		} else {
			message = "Bundle not found or already deleted"
		}

		httputil.WriteJSON(w, http.StatusOK, domain.VPSBundleDeleteResponse{
			OK:         true,
			FreedBytes: sizeBytes,
			Message:    message,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ValidateVPSBundleRequest validates common required fields for VPS bundle operations.
// Returns false and writes an error response if validation fails.
func ValidateVPSBundleRequest(w http.ResponseWriter, host, keyPath, workdir string) bool {
	if host == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_host",
			Message: "Host is required",
		})
		return false
	}
	if keyPath == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_key_path",
			Message: "SSH key path is required",
		})
		return false
	}
	if workdir == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_workdir",
			Message: "Working directory is required",
		})
		return false
	}
	return true
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
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}
}

// CleanupLocalBundles deletes old bundles from the local filesystem.
func CleanupLocalBundles(bundlesDir, scenarioID string, keepLatest int) ([]domain.BundleInfo, int64, error) {
	if scenarioID != "" {
		return DeleteBundlesForScenario(bundlesDir, scenarioID, keepLatest)
	}
	return DeleteAllOldBundles(bundlesDir, keepLatest)
}

// BuildCleanupMessage creates a human-readable summary of cleanup operations.
func BuildCleanupMessage(localDeleted []domain.BundleInfo, localFreed int64, vpsDeleted int, vpsFreed int64) string {
	var parts []string
	if len(localDeleted) > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d local bundles (%s)", len(localDeleted), FormatBytes(localFreed/1024)))
	}
	if vpsDeleted > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d VPS bundles (%s)", vpsDeleted, FormatBytes(vpsFreed/1024)))
	}
	if len(parts) == 0 {
		return "No bundles needed cleanup"
	}
	return strings.Join(parts, "; ")
}

// CleanupVPSBundles removes old bundles from the VPS.
// Returns count of deleted bundles, freed bytes, and any error.
func CleanupVPSBundles(ctx context.Context, sshRunner ssh.Runner, req domain.BundleCleanupRequest) (int, int64, error) {
	cfg := ssh.NewConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	bundlesPath := ssh.SafeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles")

	// Get initial disk usage
	beforeCmd := fmt.Sprintf("du -sb %s 2>/dev/null | cut -f1 || echo 0", ssh.QuoteSingle(bundlesPath))
	beforeRes, _ := sshRunner.Run(ctx, cfg, beforeCmd)
	beforeBytes := ParseBytes(strings.TrimSpace(beforeRes.Stdout))

	// Build cleanup command based on options
	var cleanupCmd string
	if req.ScenarioID != "" {
		// Clean specific scenario, keep N newest
		pattern := fmt.Sprintf("mini-vrooli_%s_*.tar.gz", req.ScenarioID)
		cleanupCmd = fmt.Sprintf(
			`cd %s && ls -t %s 2>/dev/null | tail -n +%d | xargs -r rm -f && ls %s 2>/dev/null | wc -l`,
			ssh.QuoteSingle(bundlesPath),
			ssh.QuoteSingle(pattern),
			req.KeepLatest+1,
			ssh.QuoteSingle(pattern),
		)
	} else {
		// Clean all scenarios, keep N newest per scenario
		// This is more complex - we need to group by scenario and keep N per group
		cleanupCmd = fmt.Sprintf(`
			cd %s 2>/dev/null || exit 0
			for scenario in $(ls mini-vrooli_*.tar.gz 2>/dev/null | sed 's/mini-vrooli_\([^_]*\)_.*/\1/' | sort -u); do
				ls -t mini-vrooli_${scenario}_*.tar.gz 2>/dev/null | tail -n +%d | xargs -r rm -f
			done
			ls mini-vrooli_*.tar.gz 2>/dev/null | wc -l || echo 0
		`, ssh.QuoteSingle(bundlesPath), req.KeepLatest+1)
	}

	// Run cleanup
	_, err := sshRunner.Run(ctx, cfg, cleanupCmd)
	if err != nil {
		return 0, 0, fmt.Errorf("cleanup command failed: %w", err)
	}

	// Get final disk usage
	afterRes, _ := sshRunner.Run(ctx, cfg, beforeCmd)
	afterBytes := ParseBytes(strings.TrimSpace(afterRes.Stdout))

	freedBytes := beforeBytes - afterBytes
	if freedBytes < 0 {
		freedBytes = 0
	}

	// Estimate count based on average bundle size (rough estimate)
	deletedCount := 0
	if freedBytes > 0 {
		avgBundleSize := int64(50 * 1024 * 1024) // 50MB estimate
		deletedCount = int(freedBytes / avgBundleSize)
		if deletedCount == 0 && freedBytes > 0 {
			deletedCount = 1
		}
	}

	return deletedCount, freedBytes, nil
}

// ListVPSBundlesSSH fetches bundle information from the VPS via SSH.
// Returns the list of bundles, total size, and any error.
func ListVPSBundlesSSH(ctx context.Context, sshRunner ssh.Runner, req domain.VPSBundleListRequest) ([]domain.VPSBundleInfo, int64, error) {
	cfg := ssh.NewConfig(req.Host, req.Port, req.User, req.KeyPath)
	bundlesPath := ssh.SafeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles")

	// List bundles with size and modification time (format: size_bytes\tfilename\tmod_time)
	listCmd := fmt.Sprintf(
		`cd %s 2>/dev/null && ls -1 mini-vrooli_*.tar.gz 2>/dev/null | while read f; do stat --printf="%%s\t%%n\t%%Y\n" "$f" 2>/dev/null; done || true`,
		ssh.QuoteSingle(bundlesPath),
	)

	res, err := sshRunner.Run(ctx, cfg, listCmd)
	if err != nil {
		return nil, 0, err
	}

	return ParseVPSBundleOutput(res.Stdout)
}

// ParseVPSBundleOutput parses the output from listing VPS bundles.
// Expected format: size_bytes\tfilename\tmod_time_unix (one per line)
func ParseVPSBundleOutput(output string) ([]domain.VPSBundleInfo, int64, error) {
	var bundles []domain.VPSBundleInfo
	var totalSize int64

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}

		sizeBytes := ParseBytes(parts[0])
		filename := parts[1]
		modTimeUnix := ParseBytes(parts[2])

		scenarioID, sha256Hash := ParseBundleFilename(filename)

		bundles = append(bundles, domain.VPSBundleInfo{
			Filename:   filename,
			ScenarioID: scenarioID,
			Sha256:     sha256Hash,
			SizeBytes:  sizeBytes,
			ModTime:    time.Unix(modTimeUnix, 0).UTC().Format(time.RFC3339),
		})
		totalSize += sizeBytes
	}

	return bundles, totalSize, nil
}

// ParseBytes parses a byte count string, returning 0 on error.
func ParseBytes(s string) int64 {
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	return n
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
