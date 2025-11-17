package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ValidationRequest represents a validation request
type ValidationRequest struct {
	UseCache bool `json:"use_cache"`
}

// ValidatePRDRequest represents a direct PRD validation request
type ValidatePRDRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
}

// TargetLinkageIssue represents a critical operational target without requirements
type TargetLinkageIssue struct {
	Title       string `json:"title"`
	Criticality string `json:"criticality"`
	Message     string `json:"message"`
}

// ValidationResponse represents the result of validation
type ValidationResponse struct {
	DraftID              string                       `json:"draft_id"`
	EntityType           string                       `json:"entity_type"`
	EntityName           string                       `json:"entity_name"`
	Violations           any                          `json:"violations"`
	TemplateCompliance   *PRDTemplateValidationResult `json:"template_compliance,omitempty"`    // Legacy validator
	TemplateComplianceV2 *PRDValidationResultV2       `json:"template_compliance_v2,omitempty"` // Enhanced validator
	TargetLinkageIssues  []TargetLinkageIssue         `json:"target_linkage_issues,omitempty"`
	CachedAt             *time.Time                   `json:"cached_at,omitempty"`
	ValidatedAt          time.Time                    `json:"validated_at"`
	CacheUsed            bool                         `json:"cache_used"`
}

var scenarioAuditorHTTPClient = &http.Client{Timeout: 45 * time.Second}

const (
	scenarioAuditorPollInterval = 2 * time.Second
	scenarioAuditorTimeout      = 2 * time.Minute
)

type scenarioAuditorStartResponse struct {
	JobID   string              `json:"job_id"`
	Status  standardsScanStatus `json:"status"`
	Message string              `json:"message"`
	Error   string              `json:"error"`
}

type standardsScanStatus struct {
	ID         string                `json:"id"`
	Scenario   string                `json:"scenario"`
	ScanType   string                `json:"scan_type"`
	Status     string                `json:"status"`
	Message    string                `json:"message"`
	Error      string                `json:"error"`
	Result     *standardsCheckResult `json:"result"`
	TotalFiles int                   `json:"total_files"`
}

type standardsCheckResult struct {
	CheckID      string               `json:"check_id"`
	Status       string               `json:"status"`
	ScanType     string               `json:"scan_type"`
	StartedAt    string               `json:"started_at"`
	CompletedAt  string               `json:"completed_at"`
	Duration     float64              `json:"duration_seconds"`
	FilesScanned int                  `json:"files_scanned"`
	Violations   []standardsViolation `json:"violations"`
	Statistics   map[string]int       `json:"statistics"`
	Message      string               `json:"message"`
	ScenarioName string               `json:"scenario_name"`
}

type standardsViolation struct {
	ID             string `json:"id"`
	ScenarioName   string `json:"scenario_name"`
	Type           string `json:"type"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	Recommendation string `json:"recommendation"`
	Standard       string `json:"standard"`
}

type diagnosticsSection struct {
	Status diagnosticsStatus `json:"status"`
}

type diagnosticsStatus struct {
	State       string             `json:"state"`
	Message     string             `json:"message,omitempty"`
	StartedAt   string             `json:"started_at,omitempty"`
	CompletedAt string             `json:"completed_at,omitempty"`
	Result      *diagnosticsResult `json:"result,omitempty"`
}

type diagnosticsResult struct {
	Statistics   map[string]int       `json:"statistics,omitempty"`
	Violations   []standardsViolation `json:"violations,omitempty"`
	FilesScanned int                  `json:"files_scanned,omitempty"`
	Duration     float64              `json:"duration_seconds,omitempty"`
	ScanType     string               `json:"scan_type,omitempty"`
}

type scenarioDiagnosticsPayload struct {
	Standards *diagnosticsSection `json:"standards,omitempty"`
}

func handleValidateDraft(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body (optional)
	var req ValidationRequest
	req.UseCache = true // Default to using cache
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Non-fatal: just use defaults if decode fails
			slog.Warn("Failed to decode validation request, using defaults", "error", err)
		}
	}

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	// Check cache if requested
	if req.UseCache {
		var violations json.RawMessage
		var cachedAt time.Time
		err := db.QueryRow(`
			SELECT violations, cached_at
			FROM audit_results
			WHERE draft_id = $1
			AND cached_at > $2
		`, draftID, draft.UpdatedAt).Scan(&violations, &cachedAt)

		if err == nil {
			// Cache hit
			var violationsData any
			if err := json.Unmarshal(violations, &violationsData); err != nil {
				slog.Warn("Failed to unmarshal cached violations, re-validating", "error", err, "draft_id", draftID)
				// Fall through to run fresh validation
			} else {
				response := ValidationResponse{
					DraftID:     draftID,
					EntityType:  draft.EntityType,
					EntityName:  draft.EntityName,
					Violations:  violationsData,
					CachedAt:    &cachedAt,
					ValidatedAt: time.Now(),
					CacheUsed:   true,
				}

				respondJSON(w, http.StatusOK, response)
				return
			}
		}
	}

	// Run scenario-auditor validation
	violations, err := runScenarioAuditor(draft.EntityType, draft.EntityName)
	if err != nil {
		respondInternalError(w, "Validation failed", err)
		return
	}

	// Run PRD template validation on draft content (both versions)
	templateValidation := ValidatePRDTemplate(draft.Content)
	templateValidationV2 := ValidatePRDTemplateV2(draft.Content)

	// Validate operational target linkage (P0/P1 targets must have requirements)
	targetLinkageIssues := validateTargetLinkage(draft.EntityType, draft.EntityName)

	// Cache validation results
	now := time.Now()
	violationsJSON, err := json.Marshal(violations)
	if err != nil {
		// Non-fatal, but log and skip caching to avoid storing corrupt data
		slog.Warn("Failed to marshal validation results for caching", "error", err, "draft_id", draftID)
	} else {
		_, err = db.Exec(`
			INSERT INTO audit_results (draft_id, violations, cached_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (draft_id)
			DO UPDATE SET violations = $2, cached_at = $3
		`, draftID, violationsJSON, now)

		if err != nil {
			// Non-fatal, just log
			slog.Warn("Failed to cache validation results", "error", err, "draft_id", draftID)
		}
	}

	response := ValidationResponse{
		DraftID:              draftID,
		EntityType:           draft.EntityType,
		EntityName:           draft.EntityName,
		Violations:           violations,
		TemplateCompliance:   &templateValidation,
		TemplateComplianceV2: &templateValidationV2,
		TargetLinkageIssues:  targetLinkageIssues,
		ValidatedAt:          now,
		CacheUsed:            false,
	}

	respondJSON(w, http.StatusOK, response)
}

// validateTargetLinkage checks if P0/P1 operational targets have linked requirements
func validateTargetLinkage(entityType string, entityName string) []TargetLinkageIssue {
	var issues []TargetLinkageIssue

	// Load operational targets
	targets, err := extractOperationalTargets(entityType, entityName)
	if err != nil {
		// If we can't load targets, return empty (non-fatal)
		return issues
	}

	// Check each P0/P1 target for requirements linkage
	for _, target := range targets {
		if (target.Criticality == "P0" || target.Criticality == "P1") && len(target.LinkedRequirements) == 0 {
			issues = append(issues, TargetLinkageIssue{
				Title:       target.Title,
				Criticality: target.Criticality,
				Message:     fmt.Sprintf("%s target '%s' must be linked to at least one requirement before publishing", target.Criticality, target.Title),
			})
		}
	}

	return issues
}

func runScenarioAuditor(entityType string, entityName string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scenarioAuditorTimeout)
	defer cancel()

	if result, err := runScenarioAuditorHTTP(ctx, entityName); err == nil {
		return result, nil
	} else {
		slog.Warn("scenario-auditor HTTP diagnostics failed, falling back to CLI",
			"entity", entityName,
			"error", err)
	}

	return runScenarioAuditorCLI(entityName)
}

func runScenarioAuditorCLI(entityName string) (any, error) {
	// Check if scenario-auditor CLI is available
	_, err := exec.LookPath("scenario-auditor")
	if err != nil {
		return map[string]any{
			"error":   "scenario-auditor not available",
			"message": "Install scenario-auditor to enable PRD validation",
		}, nil
	}

	// Run scenario-auditor (no --json flag, returns JSON by default)
	cmd := exec.Command("scenario-auditor", "audit", entityName, "--timeout", "240")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// scenario-auditor might return non-zero even with valid output
		// Check if we have any stdout
		if stdout.Len() == 0 {
			// No output, return a simple message
			return map[string]any{
				"message": fmt.Sprintf("scenario-auditor returned no output: %v", err),
				"stderr":  stderr.String(),
			}, nil
		}
	}

	// Try to parse as JSON
	var result any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// If JSON parse fails, return text output
		return map[string]any{
			"output":      stdout.String(),
			"parse_error": err.Error(),
		}, nil
	}

	return result, nil
}

func runScenarioAuditorHTTP(ctx context.Context, scenarioName string) (any, error) {
	trimmed := strings.TrimSpace(scenarioName)
	if trimmed == "" {
		return nil, errors.New("scenario name is required for diagnostics")
	}

	port, err := resolveScenarioPortViaCLI(ctx, "scenario-auditor", "API_PORT")
	if err != nil {
		return nil, fmt.Errorf("scenario-auditor API unavailable: %w", err)
	}
	if port <= 0 {
		return nil, errors.New("scenario-auditor API port not found")
	}

	startResp, err := startScenarioAuditorStandardsScan(ctx, port, trimmed)
	if err != nil {
		return nil, err
	}

	jobID := strings.TrimSpace(startResp.JobID)
	if jobID == "" {
		jobID = strings.TrimSpace(startResp.Status.ID)
	}
	if jobID == "" {
		return nil, errors.New("scenario-auditor did not return a job id")
	}

	status, err := waitForScenarioAuditorScan(ctx, port, jobID)
	if err != nil {
		return nil, err
	}

	payload := scenarioDiagnosticsPayload{
		Standards: buildStandardsDiagnostics(status),
	}

	return payload, nil
}

func startScenarioAuditorStandardsScan(ctx context.Context, port int, scenarioName string) (scenarioAuditorStartResponse, error) {
	endpoint := scenarioAuditorURL(port, fmt.Sprintf("/api/v1/standards/check/%s", url.PathEscape(scenarioName)))
	body, err := json.Marshal(map[string]any{
		"type": "quick",
	})
	if err != nil {
		return scenarioAuditorStartResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return scenarioAuditorStartResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := scenarioAuditorHTTPClient.Do(req)
	if err != nil {
		return scenarioAuditorStartResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return scenarioAuditorStartResponse{}, fmt.Errorf("scenario-auditor rejected scan (%d): %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var startResp scenarioAuditorStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&startResp); err != nil {
		return scenarioAuditorStartResponse{}, fmt.Errorf("failed to decode scan start response: %w", err)
	}

	if errMsg := strings.TrimSpace(startResp.Error); errMsg != "" {
		return scenarioAuditorStartResponse{}, errors.New(errMsg)
	}

	return startResp, nil
}

func waitForScenarioAuditorScan(ctx context.Context, port int, jobID string) (standardsScanStatus, error) {
	ticker := time.NewTicker(scenarioAuditorPollInterval)
	defer ticker.Stop()

	for {
		status, err := fetchScenarioAuditorStatus(ctx, port, jobID)
		if err != nil {
			return standardsScanStatus{}, err
		}

		state := normalizeStatus(status.Status)
		switch state {
		case "completed", "success":
			return status, nil
		case "failed", "error":
			return status, fmt.Errorf("scenario-auditor scan failed: %s", firstNonEmpty(status.Error, status.Message, "unknown failure"))
		}

		select {
		case <-ctx.Done():
			return standardsScanStatus{}, fmt.Errorf("scenario-auditor scan timed out: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func fetchScenarioAuditorStatus(ctx context.Context, port int, jobID string) (standardsScanStatus, error) {
	endpoint := scenarioAuditorURL(port, fmt.Sprintf("/api/v1/standards/check/jobs/%s", url.PathEscape(jobID)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return standardsScanStatus{}, err
	}

	resp, err := scenarioAuditorHTTPClient.Do(req)
	if err != nil {
		return standardsScanStatus{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return standardsScanStatus{}, fmt.Errorf("scenario-auditor status request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var status standardsScanStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return standardsScanStatus{}, fmt.Errorf("failed to decode scenario-auditor status: %w", err)
	}

	return status, nil
}

func buildStandardsDiagnostics(status standardsScanStatus) *diagnosticsSection {
	section := &diagnosticsSection{
		Status: diagnosticsStatus{
			State:   normalizeStatus(status.Status),
			Message: firstNonEmpty(resultMessage(status.Result), status.Message, status.Error),
		},
	}

	if status.Result != nil {
		section.Status.StartedAt = status.Result.StartedAt
		section.Status.CompletedAt = status.Result.CompletedAt
		section.Status.Result = &diagnosticsResult{
			Statistics:   status.Result.Statistics,
			Violations:   status.Result.Violations,
			FilesScanned: status.Result.FilesScanned,
			Duration:     status.Result.Duration,
			ScanType:     status.Result.ScanType,
		}
	}

	return section
}

func resultMessage(result *standardsCheckResult) string {
	if result == nil {
		return ""
	}
	return result.Message
}

// handleValidatePRD validates a published PRD (not a draft)
func handleValidatePRD(w http.ResponseWriter, r *http.Request) {
	var req ValidatePRDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	if !isValidEntityType(req.EntityType) {
		respondInvalidEntityType(w)
		return
	}

	if req.EntityName == "" {
		respondBadRequest(w, "entity_name is required")
		return
	}

	// Run validation using scenario-auditor CLI
	violations, err := runScenarioAuditorCLI(req.EntityName)
	if err != nil {
		respondInternalError(w, "Validation failed", err)
		return
	}

	response := map[string]any{
		"entity_type":  req.EntityType,
		"entity_name":  req.EntityName,
		"violations":   violations,
		"validated_at": time.Now(),
	}

	respondJSON(w, http.StatusOK, response)
}

func scenarioAuditorURL(port int, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("http://localhost:%d%s", port, path)
}

func normalizeStatus(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	return strings.ToLower(trimmed)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
