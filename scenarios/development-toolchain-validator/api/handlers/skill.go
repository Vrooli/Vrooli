// DOC: docs/reference/api-endpoints.md#skill-connections
// DOC: docs/internal/SEAMS.md#api-handlers--domain-services
// DOC: docs/internal/SEAMS.md#error-semantics
// DOC: docs/internal/SEAMS.md#decision-skill-error-mapping
//
// Package handlers implements HTTP request handling.
//
// Skill Connection Handler:
// - Uses centralized error mapping via HandleConnectError/HandleConnectionGetError
// - Supports dry-run mode via X-Dry-Run header for validation without persistence
// - All mutating operations (Connect, Update, Disconnect) support dry-run
//
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"development-toolchain-validator/domain/skill"
)

// SkillHandler handles HTTP requests for skill connections.
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
type SkillHandler struct {
	service *skill.Service
}

// NewSkillHandler creates a new skill connection handler.
func NewSkillHandler(service *skill.Service) *SkillHandler {
	return &SkillHandler{
		service: service,
	}
}

// RegisterRoutes adds skill connection routes to the router.
func (h *SkillHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/connections", h.List).Methods("GET")
	r.HandleFunc("/api/v1/connections", h.Connect).Methods("POST")
	r.HandleFunc("/api/v1/connections/{id}", h.GetByID).Methods("GET")
	r.HandleFunc("/api/v1/connections/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/connections/{id}", h.Disconnect).Methods("DELETE")
	r.HandleFunc("/api/v1/connections/{id}/drift", h.CheckDrift).Methods("POST")
	r.HandleFunc("/api/v1/references/{reference_id}/connections/{skill_id}", h.GetByReferenceAndSkill).Methods("GET")
	r.HandleFunc("/api/v1/references/{reference_id}/connections/{skill_id}", h.DisconnectByReferenceAndSkill).Methods("DELETE")
}

// List returns all skill connections with optional filtering.
func (h *SkillHandler) List(w http.ResponseWriter, r *http.Request) {
	opts := skill.ListOptions{
		ReferenceID: r.URL.Query().Get("reference_id"),
		SkillID:     r.URL.Query().Get("skill_id"),
	}

	conns, err := h.service.List(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list connections")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connections": conns,
		"count":       len(conns),
	})
}

// Connect creates a new skill-reference connection.
// Supports dry-run: X-Dry-Run: true header runs validation without persistence.
func (h *SkillHandler) Connect(w http.ResponseWriter, r *http.Request) {
	var input skill.ConnectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Dry-run: validate input and return realistic response without persistence
	if isDryRun(r) {
		if err := h.service.ValidateConnect(r.Context(), input); err != nil {
			status, msg := HandleConnectError(err, input.ReferenceID, input.SkillID)
			writeError(w, status, msg)
			return
		}

		// Return realistic response with generated values
		now := time.Now()
		dryRunConn := skill.Connection{
			ID:               uuid.New().String(),
			ReferenceID:      input.ReferenceID,
			SkillID:          input.SkillID,
			SkillVersion:     input.SkillVersion,
			SkillContentHash: input.SkillContentHash,
			ConnectedAt:      now,
			UpdatedAt:        now,
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"data":    dryRunConn,
		})
		return
	}

	conn, err := h.service.Connect(r.Context(), input)
	if err != nil {
		status, msg := HandleConnectError(err, input.ReferenceID, input.SkillID)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusCreated, conn)
}

// GetByID retrieves a connection by its ID.
func (h *SkillHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	conn, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		status, msg := HandleConnectionGetError(err, id)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

// GetByReferenceAndSkill retrieves a connection by reference and skill IDs.
func (h *SkillHandler) GetByReferenceAndSkill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	referenceID := vars["reference_id"]
	skillID := vars["skill_id"]

	conn, err := h.service.GetByReferenceAndSkill(r.Context(), referenceID, skillID)
	if err != nil {
		status, msg := HandleConnectionGetError(err, referenceID+"/"+skillID)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

// Update modifies an existing connection.
// Supports dry-run: X-Dry-Run: true header runs validation without persistence.
func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input skill.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Dry-run: validate input and return realistic response without persistence
	if isDryRun(r) {
		if err := h.service.ValidateUpdate(r.Context(), id, input); err != nil {
			status, msg := HandleConnectionGetError(err, id)
			writeError(w, status, msg)
			return
		}

		// Get the existing connection to build a realistic response
		existing, err := h.service.GetByID(r.Context(), id)
		if err != nil {
			status, msg := HandleConnectionGetError(err, id)
			writeError(w, status, msg)
			return
		}

		// Apply updates to the existing connection for response
		now := time.Now()
		dryRunConn := *existing
		dryRunConn.UpdatedAt = now
		if input.SkillVersion != nil {
			dryRunConn.SkillVersion = *input.SkillVersion
		}
		if input.SkillContentHash != nil {
			dryRunConn.SkillContentHash = *input.SkillContentHash
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"data":    dryRunConn,
		})
		return
	}

	conn, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		status, msg := HandleConnectionGetError(err, id)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

// Disconnect removes a skill connection by ID.
// Supports dry-run: X-Dry-Run: true header checks existence without deletion.
func (h *SkillHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Dry-run: verify connection exists without deletion
	if isDryRun(r) {
		_, err := h.service.GetByID(r.Context(), id)
		if err != nil {
			status, msg := HandleConnectionGetError(err, id)
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":      true,
			"dry_run":      true,
			"disconnected": id,
		})
		return
	}

	if err := h.service.Disconnect(r.Context(), id); err != nil {
		status, msg := HandleConnectionGetError(err, id)
		writeError(w, status, msg)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DisconnectByReferenceAndSkill removes a connection by reference and skill IDs.
func (h *SkillHandler) DisconnectByReferenceAndSkill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	referenceID := vars["reference_id"]
	skillID := vars["skill_id"]

	// Dry-run: verify connection exists without deletion
	if isDryRun(r) {
		_, err := h.service.GetByReferenceAndSkill(r.Context(), referenceID, skillID)
		if err != nil {
			status, msg := HandleConnectionGetError(err, referenceID+"/"+skillID)
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":      true,
			"dry_run":      true,
			"disconnected": referenceID + "/" + skillID,
		})
		return
	}

	if err := h.service.DisconnectByReferenceAndSkill(r.Context(), referenceID, skillID); err != nil {
		status, msg := HandleConnectionGetError(err, referenceID+"/"+skillID)
		writeError(w, status, msg)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CheckDrift compares a connection's stored version/hash against current values.
// Request body should contain current_version and current_hash from prompt-manager.
func (h *SkillHandler) CheckDrift(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input struct {
		CurrentVersion string `json:"current_version"`
		CurrentHash    string `json:"current_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	status, err := h.service.CheckDrift(r.Context(), id, input.CurrentVersion, input.CurrentHash)
	if err != nil {
		httpStatus, msg := HandleConnectionGetError(err, id)
		writeError(w, httpStatus, msg)
		return
	}

	writeJSON(w, http.StatusOK, status)
}
