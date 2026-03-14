// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file implements skill CRUD endpoints.
package handlers

import (
	"encoding/json"
	"net/http"

	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// ListSkills returns all skills (defaults merged with user overrides).
// GET /api/v1/skills
func (h *Handlers) ListSkills(w http.ResponseWriter, r *http.Request) {
	result, err := h.Skills.ListSkills()
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.JSONResponse(w, result, http.StatusOK)
}

// GetSkill returns a single skill by ID.
// GET /api/v1/skills/{id}
func (h *Handlers) GetSkill(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "skill ID required", http.StatusBadRequest)
		return
	}

	result, err := h.Skills.GetSkill(id)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// CreateSkill creates a new skill in prompt-manager.
// POST /api/v1/skills
func (h *Handlers) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var req services.CreateSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		h.JSONError(w, "content is required", http.StatusBadRequest)
		return
	}

	// Default to local folder if not specified
	if req.Folder == "" {
		req.Folder = "local"
	}

	// Try to create in prompt-manager first
	result, err := h.Skills.CreateSkillInPromptManager(&req)
	if err != nil {
		// Fall back to local creation if prompt-manager is unavailable
		sk := &services.Skill{
			Name:         req.Name,
			Description:  req.Description,
			Content:      req.Content,
			Modes:        req.Modes,
			Tags:         req.Tags,
			Icon:         req.Icon,
			Draft:        req.Draft,
			TargetToolID: req.TargetToolID,
		}
		result, err = h.Skills.CreateSkill(sk)
		if err != nil {
			h.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	h.JSONResponse(w, result, http.StatusCreated)
}

// SyncSkills triggers an immediate sync from prompt-manager.
// POST /api/v1/skills/sync
func (h *Handlers) SyncSkills(w http.ResponseWriter, r *http.Request) {
	status, err := h.Skills.TriggerSync()
	if err != nil {
		// Return the status even on error - it contains useful info
		h.JSONResponse(w, status, http.StatusOK)
		return
	}

	h.JSONResponse(w, status, http.StatusOK)
}

// UpdateSkill updates an existing skill.
// PUT /api/v1/skills/{id}
func (h *Handlers) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "skill ID required", http.StatusBadRequest)
		return
	}

	var updates services.Skill
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.Skills.UpdateSkill(id, &updates)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// DeleteSkill deletes a user skill or user override.
// DELETE /api/v1/skills/{id}
func (h *Handlers) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "skill ID required", http.StatusBadRequest)
		return
	}

	if err := h.Skills.DeleteSkill(id); err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, map[string]bool{"deleted": true}, http.StatusOK)
}

// ImportSkills imports multiple skills from a JSON array.
// POST /api/v1/skills/import
func (h *Handlers) ImportSkills(w http.ResponseWriter, r *http.Request) {
	var skills []services.Skill
	if err := json.NewDecoder(r.Body).Decode(&skills); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	imported, err := h.Skills.ImportSkills(skills)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]int{"imported": imported}, http.StatusOK)
}

// ExportSkills exports all user skills.
// GET /api/v1/skills/export
func (h *Handlers) ExportSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.Skills.ExportSkills()
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, skills, http.StatusOK)
}
