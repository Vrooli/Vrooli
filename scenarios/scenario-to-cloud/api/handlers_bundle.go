package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// Type aliases for backward compatibility and shorter references within main package.
type (
	BundleCleanupRequest    = domain.BundleCleanupRequest
	BundleCleanupResponse   = domain.BundleCleanupResponse
	BundleStatsResponse     = domain.BundleStatsResponse
	BundleDeleteResponse    = domain.BundleDeleteResponse
	VPSBundleListRequest    = domain.VPSBundleListRequest
	VPSBundleInfo           = domain.VPSBundleInfo
	VPSBundleListResponse   = domain.VPSBundleListResponse
	VPSBundleDeleteRequest  = domain.VPSBundleDeleteRequest
	VPSBundleDeleteResponse = domain.VPSBundleDeleteResponse
)

// handleManifestValidate validates a CloudManifest and returns normalized output.
// POST /api/v1/manifest/validate
func (s *Server) handleManifestValidate(w http.ResponseWriter, r *http.Request) {
	manifest, err := decodeJSON[CloudManifest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	resp := ManifestValidateResponse{
		Valid:      len(issues) == 0,
		Issues:     issues,
		Manifest:   normalized,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		SchemaHint: "This endpoint is the consumer-side contract. deployment-manager is responsible for exporting the manifest.",
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleBundleBuild builds a mini-Vrooli bundle from a manifest.
// POST /api/v1/bundle/build
func (s *Server) handleBundleBuild(w http.ResponseWriter, r *http.Request) {
	manifest, err := decodeJSON[CloudManifest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues := ValidateAndNormalizeManifest(manifest)
	if hasBlockingIssues(issues) {
		writeJSON(w, http.StatusUnprocessableEntity, ManifestValidateResponse{
			Valid:     false,
			Issues:    issues,
			Manifest:  normalized,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_root_not_found",
			Message: "Unable to locate Vrooli repo root from server working directory",
			Hint:    err.Error(),
		})
		return
	}

	outDir := repoRoot + "/scenarios/scenario-to-cloud/coverage/bundles"
	artifact, err := BuildMiniVrooliBundle(repoRoot, outDir, normalized)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "bundle_build_failed",
			Message: "Failed to build mini-Vrooli bundle",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"artifact":  artifact,
		"issues":    issues,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleListBundles lists all stored bundles.
// GET /api/v1/bundles
func (s *Server) handleListBundles(w http.ResponseWriter, r *http.Request) {
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_root_not_found",
			Message: "Unable to locate Vrooli repo root from server working directory",
			Hint:    err.Error(),
		})
		return
	}

	bundlesDir := repoRoot + "/scenarios/scenario-to-cloud/coverage/bundles"
	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "list_bundles_failed",
			Message: "Failed to list bundles",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bundles":   bundles,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleBundleStats returns aggregate statistics about stored bundles.
func (s *Server) handleBundleStats(w http.ResponseWriter, r *http.Request) {
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_not_found",
			Message: "Could not find repository root",
			Hint:    err.Error(),
		})
		return
	}

	bundlesDir := filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "coverage", "bundles")
	stats, err := GetBundleStats(bundlesDir)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "stats_failed",
			Message: "Failed to get bundle statistics",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, BundleStatsResponse{
		Stats:     stats,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleBundleCleanup removes old bundles from local storage and optionally from VPS.
func (s *Server) handleBundleCleanup(w http.ResponseWriter, r *http.Request) {
	var req BundleCleanupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Invalid request body",
			Hint:    err.Error(),
		})
		return
	}

	// Set defaults
	if req.KeepLatest <= 0 {
		req.KeepLatest = 3
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}

	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_not_found",
			Message: "Could not find repository root",
			Hint:    err.Error(),
		})
		return
	}

	bundlesDir := filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "coverage", "bundles")

	var deleted []BundleInfo
	var freedBytes int64

	// Local cleanup
	if req.ScenarioID != "" {
		// Clean specific scenario
		deleted, freedBytes, err = DeleteBundlesForScenario(bundlesDir, req.ScenarioID, req.KeepLatest)
	} else {
		// Clean all scenarios
		deleted, freedBytes, err = DeleteAllOldBundles(bundlesDir, req.KeepLatest)
	}

	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "cleanup_failed",
			Message: "Failed to clean up local bundles",
			Hint:    err.Error(),
		})
		return
	}

	resp := BundleCleanupResponse{
		OK:              true,
		LocalDeleted:    deleted,
		LocalFreedBytes: freedBytes,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	// VPS cleanup (if requested and SSH config provided)
	if req.CleanVPS && req.Host != "" && req.KeyPath != "" && req.Workdir != "" {
		vpsDeleted, vpsFreed, vpsErr := s.cleanupVPSBundles(r.Context(), req)
		resp.VPSDeleted = vpsDeleted
		resp.VPSFreedBytes = vpsFreed
		if vpsErr != nil {
			resp.VPSError = vpsErr.Error()
			resp.OK = false // Partial failure
		}
	}

	// Build message
	var parts []string
	if len(deleted) > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d local bundles (%s)", len(deleted), formatBytes(freedBytes/1024)))
	}
	if resp.VPSDeleted > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d VPS bundles (%s)", resp.VPSDeleted, formatBytes(resp.VPSFreedBytes/1024)))
	}
	if len(parts) == 0 {
		resp.Message = "No bundles needed cleanup"
	} else {
		resp.Message = strings.Join(parts, "; ")
	}

	writeJSON(w, http.StatusOK, resp)
}

// cleanupVPSBundles removes old bundles from the VPS.
// Returns count of deleted bundles, freed bytes, and any error.
func (s *Server) cleanupVPSBundles(ctx context.Context, req BundleCleanupRequest) (int, int64, error) {
	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	bundlesPath := safeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles")

	// Get initial disk usage
	beforeCmd := fmt.Sprintf("du -sb %s 2>/dev/null | cut -f1 || echo 0", shellQuoteSingle(bundlesPath))
	beforeRes, _ := s.sshRunner.Run(ctx, cfg, beforeCmd)
	beforeBytes := parseBytes(strings.TrimSpace(beforeRes.Stdout))

	// Build cleanup command based on options
	var cleanupCmd string
	if req.ScenarioID != "" {
		// Clean specific scenario, keep N newest
		pattern := fmt.Sprintf("mini-vrooli_%s_*.tar.gz", req.ScenarioID)
		cleanupCmd = fmt.Sprintf(
			`cd %s && ls -t %s 2>/dev/null | tail -n +%d | xargs -r rm -f && ls %s 2>/dev/null | wc -l`,
			shellQuoteSingle(bundlesPath),
			shellQuoteSingle(pattern),
			req.KeepLatest+1,
			shellQuoteSingle(pattern),
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
		`, shellQuoteSingle(bundlesPath), req.KeepLatest+1)
	}

	// Run cleanup
	_, err := s.sshRunner.Run(ctx, cfg, cleanupCmd)
	if err != nil {
		return 0, 0, fmt.Errorf("cleanup command failed: %w", err)
	}

	// Get final disk usage
	afterRes, _ := s.sshRunner.Run(ctx, cfg, beforeCmd)
	afterBytes := parseBytes(strings.TrimSpace(afterRes.Stdout))

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

// parseBytes parses a byte count string, returning 0 on error.
func parseBytes(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

// handleListVPSBundles lists bundles stored on the VPS.
func (s *Server) handleListVPSBundles(w http.ResponseWriter, r *http.Request) {
	var req VPSBundleListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Invalid request body",
			Hint:    err.Error(),
		})
		return
	}

	// Validate required fields
	if req.Host == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_host",
			Message: "Host is required",
		})
		return
	}
	if req.KeyPath == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_key_path",
			Message: "SSH key path is required",
		})
		return
	}
	if req.Workdir == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_workdir",
			Message: "Working directory is required",
		})
		return
	}

	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	bundlesPath := safeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles")

	// List bundles with size and modification time
	// Format: size_bytes filename mod_time
	listCmd := fmt.Sprintf(
		`cd %s 2>/dev/null && ls -1 mini-vrooli_*.tar.gz 2>/dev/null | while read f; do stat --printf="%%s\t%%n\t%%Y\n" "$f" 2>/dev/null; done || true`,
		shellQuoteSingle(bundlesPath),
	)

	res, err := s.sshRunner.Run(ctx, cfg, listCmd)
	if err != nil {
		writeJSON(w, http.StatusOK, VPSBundleListResponse{
			OK:        false,
			Bundles:   []VPSBundleInfo{},
			Error:     fmt.Sprintf("Failed to connect or list bundles: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	var bundles []VPSBundleInfo
	var totalSize int64

	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}

		sizeBytes := parseBytes(parts[0])
		filename := parts[1]
		modTimeUnix := parseBytes(parts[2])

		scenarioID, sha256Hash := parseBundleFilename(filename)

		bundles = append(bundles, VPSBundleInfo{
			Filename:   filename,
			ScenarioID: scenarioID,
			Sha256:     sha256Hash,
			SizeBytes:  sizeBytes,
			ModTime:    time.Unix(modTimeUnix, 0).UTC().Format(time.RFC3339),
		})
		totalSize += sizeBytes
	}

	writeJSON(w, http.StatusOK, VPSBundleListResponse{
		OK:             true,
		Bundles:        bundles,
		TotalSizeBytes: totalSize,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	})
}

// handleDeleteVPSBundle deletes a single bundle from the VPS.
func (s *Server) handleDeleteVPSBundle(w http.ResponseWriter, r *http.Request) {
	var req VPSBundleDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Invalid request body",
			Hint:    err.Error(),
		})
		return
	}

	// Validate required fields
	if req.Host == "" || req.KeyPath == "" || req.Workdir == "" || req.Filename == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_fields",
			Message: "host, key_path, workdir, and filename are required",
		})
		return
	}

	// Validate filename format to prevent path traversal
	if !strings.HasPrefix(req.Filename, "mini-vrooli_") || !strings.HasSuffix(req.Filename, ".tar.gz") {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_filename",
			Message: "Invalid bundle filename format",
		})
		return
	}

	cfg := NewSSHConfig(req.Host, req.Port, req.User, req.KeyPath)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	bundlePath := safeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles", req.Filename)

	// Get file size before deleting
	sizeCmd := fmt.Sprintf("stat -c %%s %s 2>/dev/null || echo 0", shellQuoteSingle(bundlePath))
	sizeRes, _ := s.sshRunner.Run(ctx, cfg, sizeCmd)
	sizeBytes := parseBytes(strings.TrimSpace(sizeRes.Stdout))

	// Delete the file
	deleteCmd := fmt.Sprintf("rm -f %s", shellQuoteSingle(bundlePath))
	_, err := s.sshRunner.Run(ctx, cfg, deleteCmd)
	if err != nil {
		writeJSON(w, http.StatusOK, VPSBundleDeleteResponse{
			OK:        false,
			Error:     fmt.Sprintf("Failed to delete bundle: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	var message string
	if sizeBytes > 0 {
		message = fmt.Sprintf("Deleted %s, freed %s", req.Filename, formatBytes(sizeBytes/1024))
	} else {
		message = "Bundle not found or already deleted"
	}

	writeJSON(w, http.StatusOK, VPSBundleDeleteResponse{
		OK:         true,
		FreedBytes: sizeBytes,
		Message:    message,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleDeleteBundle deletes a single bundle by SHA256 hash.
func (s *Server) handleDeleteBundle(w http.ResponseWriter, r *http.Request) {
	sha256Hash := strings.TrimPrefix(r.URL.Path, "/api/v1/bundles/")
	if sha256Hash == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_sha256",
			Message: "SHA256 hash is required in the URL path",
			Hint:    "Use DELETE /api/v1/bundles/{sha256}",
		})
		return
	}

	// Validate SHA256 format (64 hex chars)
	if len(sha256Hash) != 64 {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_sha256",
			Message: "Invalid SHA256 hash format",
			Hint:    "SHA256 hash must be exactly 64 hexadecimal characters",
		})
		return
	}

	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "repo_not_found",
			Message: "Could not find repository root",
			Hint:    err.Error(),
		})
		return
	}

	bundlesDir := filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "coverage", "bundles")
	freedBytes, err := DeleteBundle(bundlesDir, sha256Hash)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "delete_failed",
			Message: "Failed to delete bundle",
			Hint:    err.Error(),
		})
		return
	}

	var message string
	if freedBytes > 0 {
		message = fmt.Sprintf("Deleted bundle, freed %s", formatBytes(freedBytes/1024))
	} else {
		message = "Bundle not found or already deleted"
	}

	writeJSON(w, http.StatusOK, BundleDeleteResponse{
		OK:         true,
		FreedBytes: freedBytes,
		Message:    message,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
}
