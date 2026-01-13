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

// CreateSkill creates a new user skill.
// POST /api/v1/skills
func (h *Handlers) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var sk services.Skill
	if err := json.NewDecoder(r.Body).Decode(&sk); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if sk.Name == "" {
		h.JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if sk.Content == "" {
		h.JSONError(w, "content is required", http.StatusBadRequest)
		return
	}

	result, err := h.Skills.CreateSkill(&sk)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.JSONResponse(w, result, http.StatusCreated)
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

// UpdateDefaultSkill updates the actual default skill (not a user override).
// PUT /api/v1/skills/{id}/update-default
func (h *Handlers) UpdateDefaultSkill(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.Skills.UpdateDefaultSkill(id, &updates)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// ResetSkill resets a modified default skill to its original state.
// POST /api/v1/skills/{id}/reset
func (h *Handlers) ResetSkill(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "skill ID required", http.StatusBadRequest)
		return
	}

	result, err := h.Skills.ResetSkill(id)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
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
