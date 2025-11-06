package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
	DraftID     string     `json:"draft_id"`
	EntityType  string     `json:"entity_type"`
	EntityName  string     `json:"entity_name"`
	Violations  any        `json:"violations"`
	CachedAt    *time.Time `json:"cached_at,omitempty"`
	ValidatedAt time.Time  `json:"validated_at"`
	CacheUsed   bool       `json:"cache_used"`
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

	// Run validation
	violations, err := runScenarioAuditor(draft.EntityType, draft.EntityName)
	if err != nil {
		respondInternalError(w, "Validation failed", err)
		return
	}

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
		DraftID:     draftID,
		EntityType:  draft.EntityType,
		EntityName:  draft.EntityName,
		Violations:  violations,
		ValidatedAt: now,
		CacheUsed:   false,
	}

	respondJSON(w, http.StatusOK, response)
}

func runScenarioAuditor(entityType string, entityName string) (any, error) {
	// Use CLI-based scenario auditor
	// TODO: Add HTTP API support when scenario-auditor provides HTTP endpoint
	// Note: When HTTP API is added, the draft content may be passed as a parameter
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
