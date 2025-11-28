package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// LightScanRequest defines the request body for light scanning
type LightScanRequest struct {
	ScenarioPath string `json:"scenario_path"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
	Incremental  bool   `json:"incremental,omitempty"` // Only scan modified files
}

// ParseRequest defines request for parsing lint/type output
type ParseRequest struct {
	Scenario string `json:"scenario"`
	Tool     string `json:"tool"`
	Output   string `json:"output"`
}

// hasRecentScan checks if we have scanned this scenario recently
func (s *Server) hasRecentScan(ctx context.Context, scenario string, within time.Duration) (bool, error) {
	var lastUpdate time.Time
	query := `
		SELECT MAX(updated_at)
		FROM file_metrics
		WHERE scenario = $1
	`
	err := s.db.QueryRowContext(ctx, query, scenario).Scan(&lastUpdate)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check if last update was within the specified duration
	age := time.Since(lastUpdate)
	return age <= within, nil
}

// handleLightScan performs a complete light scan on a scenario
func (s *Server) handleLightScan(w http.ResponseWriter, r *http.Request) {
	var req LightScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.ScenarioPath == "" {
		respondError(w, http.StatusBadRequest, "scenario_path is required")
		return
	}

	// Validate path exists
	absPath, err := filepath.Abs(req.ScenarioPath)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid scenario_path")
		return
	}

	timeout := 120 * time.Second
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}

	scanner := NewLightScanner(absPath, timeout)

	// Auto-enable incremental mode if:
	// 1. Not explicitly set to false AND
	// 2. We have a recent scan (within 24 hours)
	incremental := req.Incremental
	if !incremental {
		hasRecent, _ := s.hasRecentScan(r.Context(), filepath.Base(absPath), 24*time.Hour)
		if hasRecent {
			incremental = true
			s.log("auto-enabled incremental scan", map[string]interface{}{
				"scenario": filepath.Base(absPath),
			})
		}
	}

	scanOpts := ScanOptions{
		Incremental: incremental,
		DB:          s.db,
	}

	result, err := scanner.ScanWithOptions(r.Context(), scanOpts)
	if err != nil {
		s.log("light scan failed", map[string]interface{}{
			"error":       err.Error(),
			"scenario":    absPath,
			"incremental": req.Incremental,
		})
		respondError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}

	// Collect and persist detailed file metrics to database (TM-FM-001, TM-FM-002)
	scenarioPath := getScenarioPath(result.Scenario)

	// Extract all file paths from scan results
	filePaths := make([]string, len(result.FileMetrics))
	for i, fm := range result.FileMetrics {
		filePaths[i] = fm.Path
	}

	detailedMetrics, err := CollectDetailedFileMetrics(scenarioPath, filePaths)
	if err != nil {
		s.log("failed to collect detailed metrics", map[string]interface{}{
			"error":    err.Error(),
			"scenario": result.Scenario,
		})
		// Fallback to basic metrics
		if err := s.persistFileMetrics(r.Context(), result.Scenario, result.FileMetrics); err != nil {
			s.log("failed to persist basic file metrics", map[string]interface{}{
				"error":    err.Error(),
				"scenario": result.Scenario,
			})
		}
	} else {
		if err := s.persistDetailedFileMetrics(r.Context(), result.Scenario, detailedMetrics); err != nil {
			s.log("failed to persist detailed file metrics", map[string]interface{}{
				"error":    err.Error(),
				"scenario": result.Scenario,
			})
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// handleParseLint parses lint output into structured issues
func (s *Server) handleParseLint(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseLintOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// handleParseType parses type checker output into structured issues
func (s *Server) handleParseType(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseTypeOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// decodeAndValidateJSON decodes JSON request body and validates required fields
func decodeAndValidateJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// getScenarioPath resolves the full scenario path from VROOLI_ROOT
func getScenarioPath(scenario string) string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	return filepath.Join(vrooliRoot, "scenarios", scenario)
}

// handleSmartScan performs AI-powered smart scanning (TM-SS-001, TM-SS-002)
func (s *Server) handleSmartScan(w http.ResponseWriter, r *http.Request) {
	var req SmartScanRequest
	if !decodeAndValidateJSON(w, r, &req) {
		return
	}

	if req.Scenario == "" {
		respondError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	if len(req.Files) == 0 {
		respondError(w, http.StatusBadRequest, "files list cannot be empty")
		return
	}

	// Create smart scanner with default configuration
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create scanner: "+err.Error())
		return
	}

	// Perform smart scan
	result, err := scanner.ScanScenario(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}

	// Store issues in database (TM-SS-002)
	for _, batchResult := range result.BatchResults {
		for _, issue := range batchResult.Issues {
			if err := s.storeAIIssue(r.Context(), req.Scenario, issue, result.SessionID, req.CampaignID); err != nil {
				s.log("failed to store issue", map[string]interface{}{
					"error":    err.Error(),
					"scenario": req.Scenario,
					"file":     issue.FilePath,
				})
				// Continue processing other issues even if one fails
			}
		}
	}

	// Record scan in history
	if err := s.recordScanHistory(r.Context(), req.Scenario, "smart", result, req.CampaignID); err != nil {
		s.log("failed to record scan history", map[string]interface{}{
			"error":    err.Error(),
			"scenario": req.Scenario,
		})
	}

	respondJSON(w, http.StatusOK, result)
}

// storeAIIssue stores an AI-discovered issue in the database (TM-SS-002, TM-API-006)
func (s *Server) storeAIIssue(ctx context.Context, scenario string, issue AIIssue, sessionID string, campaignID *int) error {
	query := `
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			line_number, column_number, agent_notes, remediation_steps,
			session_id, campaign_id, resource_used, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (scenario, file_path, category, line_number, column_number, created_at)
		DO UPDATE SET
			description = EXCLUDED.description,
			remediation_steps = EXCLUDED.remediation_steps,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.ExecContext(ctx, query,
		scenario,
		issue.FilePath,
		issue.Category,
		issue.Severity,
		issue.Title,
		issue.Description,
		issue.LineNumber,
		issue.ColumnNumber,
		issue.AgentNotes,
		issue.RemediationSteps,
		sessionID,
		campaignID,
		"resource-claude-code", // or detect from config
		"open",
	)

	return err
}

// recordScanHistory records a scan in the audit trail
func (s *Server) recordScanHistory(ctx context.Context, scenario, scanType string, result *SmartScanResult, campaignID *int) error {
	query := `
		INSERT INTO scan_history (
			scenario, scan_type, resource_used, issues_found,
			duration_seconds, campaign_id, session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		scenario,
		scanType,
		"resource-claude-code",
		result.IssuesFound,
		result.Duration.Seconds(),
		campaignID,
		result.SessionID,
	)

	return err
}

// persistFileMetrics stores file metrics from a light scan in the database (TM-FM-001, TM-FM-002)
func (s *Server) persistFileMetrics(ctx context.Context, scenario string, metrics []FileMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// Use upsert to handle repeated scans
	query := `
		INSERT INTO file_metrics (scenario, file_path, line_count, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (scenario, file_path)
		DO UPDATE SET
			line_count = EXCLUDED.line_count,
			updated_at = CURRENT_TIMESTAMP
	`

	for _, metric := range metrics {
		_, err := s.db.ExecContext(ctx, query,
			scenario,
			metric.Path,
			metric.Lines,
		)
		if err != nil {
			return fmt.Errorf("failed to persist file metric for %s: %w", metric.Path, err)
		}
	}

	return nil
}
