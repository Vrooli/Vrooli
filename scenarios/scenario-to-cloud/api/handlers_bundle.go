package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// BundleCleanupRequest is the request body for bundle cleanup.
type BundleCleanupRequest struct {
	// Local cleanup options
	ScenarioID string `json:"scenario_id,omitempty"` // If set, only clean this scenario's bundles
	KeepLatest int    `json:"keep_latest"`           // Keep N most recent per scenario (default: 3)

	// VPS cleanup options (optional)
	CleanVPS bool   `json:"clean_vps,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	KeyPath  string `json:"key_path,omitempty"`
	Workdir  string `json:"workdir,omitempty"`
}

// BundleCleanupResponse is the response from bundle cleanup.
type BundleCleanupResponse struct {
	OK              bool         `json:"ok"`
	LocalDeleted    []BundleInfo `json:"local_deleted,omitempty"`
	LocalFreedBytes int64        `json:"local_freed_bytes"`
	VPSDeleted      int          `json:"vps_deleted,omitempty"`
	VPSFreedBytes   int64        `json:"vps_freed_bytes,omitempty"`
	VPSError        string       `json:"vps_error,omitempty"`
	Message         string       `json:"message"`
	Timestamp       string       `json:"timestamp"`
}

// BundleStatsResponse is the response from bundle stats.
type BundleStatsResponse struct {
	Stats     BundleStats `json:"stats"`
	Timestamp string      `json:"timestamp"`
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
	cfg := SSHConfig{
		Host:    req.Host,
		Port:    req.Port,
		User:    req.User,
		KeyPath: req.KeyPath,
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	sshRunner := ExecSSHRunner{}
	bundlesPath := safeRemoteJoin(req.Workdir, ".vrooli/cloud/bundles")

	// Get initial disk usage
	beforeCmd := fmt.Sprintf("du -sb %s 2>/dev/null | cut -f1 || echo 0", shellQuoteSingle(bundlesPath))
	beforeRes, _ := sshRunner.Run(ctx, cfg, beforeCmd)
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
	_, err := sshRunner.Run(ctx, cfg, cleanupCmd)
	if err != nil {
		return 0, 0, fmt.Errorf("cleanup command failed: %w", err)
	}

	// Get final disk usage
	afterRes, _ := sshRunner.Run(ctx, cfg, beforeCmd)
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
