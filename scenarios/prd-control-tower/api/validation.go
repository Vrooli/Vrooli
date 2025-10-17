package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
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

// ValidationResponse represents the result of validation
type ValidationResponse struct {
	DraftID     string      `json:"draft_id"`
	EntityType  string      `json:"entity_type"`
	EntityName  string      `json:"entity_name"`
	Violations  interface{} `json:"violations"`
	CachedAt    *time.Time  `json:"cached_at,omitempty"`
	ValidatedAt time.Time   `json:"validated_at"`
	CacheUsed   bool        `json:"cache_used"`
}

func handleValidateDraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body (optional)
	var req ValidationRequest
	req.UseCache = true // Default to using cache
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	// Get draft from database
	var draft Draft
	var owner sql.NullString
	err := db.QueryRow(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		WHERE id = $1
	`, draftID).Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&owner,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.Status,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get draft: %v", err), http.StatusInternalServerError)
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
			var violationsData interface{}
			json.Unmarshal(violations, &violationsData)

			response := ValidationResponse{
				DraftID:     draftID,
				EntityType:  draft.EntityType,
				EntityName:  draft.EntityName,
				Violations:  violationsData,
				CachedAt:    &cachedAt,
				ValidatedAt: time.Now(),
				CacheUsed:   true,
			}

			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Run validation
	violations, err := runScenarioAuditor(draft.EntityType, draft.EntityName, draft.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache validation results
	violationsJSON, _ := json.Marshal(violations)
	now := time.Now()
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

	response := ValidationResponse{
		DraftID:     draftID,
		EntityType:  draft.EntityType,
		EntityName:  draft.EntityName,
		Violations:  violations,
		ValidatedAt: now,
		CacheUsed:   false,
	}

	json.NewEncoder(w).Encode(response)
}

func runScenarioAuditor(entityType string, entityName string, content string) (interface{}, error) {
	// Check if scenario-auditor is available
	auditorURL := os.Getenv("SCENARIO_AUDITOR_URL")
	if auditorURL != "" {
		// Try HTTP API first
		return runScenarioAuditorHTTP(auditorURL, entityType, entityName, content)
	}

	// Fallback to CLI
	return runScenarioAuditorCLI(entityType, entityName)
}

func runScenarioAuditorHTTP(baseURL string, entityType string, entityName string, content string) (interface{}, error) {
	// For now, we'll use the CLI since the HTTP API integration is more complex
	// This is a placeholder for future HTTP API implementation
	return runScenarioAuditorCLI(entityType, entityName)
}

func runScenarioAuditorCLI(entityType string, entityName string) (interface{}, error) {
	// Check if scenario-auditor CLI is available
	_, err := exec.LookPath("scenario-auditor")
	if err != nil {
		return map[string]interface{}{
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
			return map[string]interface{}{
				"message": fmt.Sprintf("scenario-auditor returned no output: %v", err),
				"stderr":  stderr.String(),
			}, nil
		}
	}

	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// If JSON parse fails, return text output
		return map[string]interface{}{
			"output":      stdout.String(),
			"parse_error": err.Error(),
		}, nil
	}

	return result, nil
}

// Helper function to call HTTP API (for future use)
func callAuditorAPI(url string, payload interface{}) (interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call auditor API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auditor API returned error: %s, body: %s", resp.Status, string(body))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode auditor response: %w", err)
	}

	return result, nil
}

// handleValidatePRD validates a published PRD (not a draft)
func handleValidatePRD(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ValidatePRDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.EntityType != "scenario" && req.EntityType != "resource" {
		http.Error(w, "Invalid entity_type. Must be 'scenario' or 'resource'", http.StatusBadRequest)
		return
	}

	if req.EntityName == "" {
		http.Error(w, "entity_name is required", http.StatusBadRequest)
		return
	}

	// Run validation using scenario-auditor CLI
	violations, err := runScenarioAuditorCLI(req.EntityType, req.EntityName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"entity_type":  req.EntityType,
		"entity_name":  req.EntityName,
		"violations":   violations,
		"validated_at": time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}
