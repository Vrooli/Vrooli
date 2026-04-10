// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file implements template CRUD endpoints.
package handlers

import (
	"encoding/json"
	"net/http"

	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// ListTemplates returns all templates (defaults merged with user overrides).
// GET /api/v1/templates
func (h *Handlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	result, err := h.Templates.ListTemplates()
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.JSONResponse(w, result, http.StatusOK)
}

// GetTemplate returns a single template by ID.
// GET /api/v1/templates/{id}
func (h *Handlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "template ID required", http.StatusBadRequest)
		return
	}

	result, err := h.Templates.GetTemplate(id)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// CreateTemplate creates a new user template.
// POST /api/v1/templates
func (h *Handlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var t services.Template
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if t.Name == "" {
		h.JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if t.Content == "" {
		h.JSONError(w, "content is required", http.StatusBadRequest)
		return
	}

	result, err := h.Templates.CreateTemplate(&t)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.JSONResponse(w, result, http.StatusCreated)
}

// UpdateTemplate updates an existing template.
// PUT /api/v1/templates/{id}
func (h *Handlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "template ID required", http.StatusBadRequest)
		return
	}

	var updates services.Template
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.Templates.UpdateTemplate(id, &updates)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// DeleteTemplate deletes a user template or user override.
// DELETE /api/v1/templates/{id}
func (h *Handlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "template ID required", http.StatusBadRequest)
		return
	}

	if err := h.Templates.DeleteTemplate(id); err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, map[string]bool{"deleted": true}, http.StatusOK)
}

// UpdateDefaultTemplate updates the actual default template (not a user override).
// PUT /api/v1/templates/{id}/update-default
func (h *Handlers) UpdateDefaultTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "template ID required", http.StatusBadRequest)
		return
	}

	var updates services.Template
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.Templates.UpdateDefaultTemplate(id, &updates)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// ResetTemplate resets a modified default template to its original state.
// POST /api/v1/templates/{id}/reset
func (h *Handlers) ResetTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		h.JSONError(w, "template ID required", http.StatusBadRequest)
		return
	}

	result, err := h.Templates.ResetTemplate(id)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// ImportTemplates imports multiple templates from a JSON array.
// POST /api/v1/templates/import
func (h *Handlers) ImportTemplates(w http.ResponseWriter, r *http.Request) {
	var templates []services.Template
	if err := json.NewDecoder(r.Body).Decode(&templates); err != nil {
		h.JSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	imported, err := h.Templates.ImportTemplates(templates)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]int{"imported": imported}, http.StatusOK)
}

// ExportTemplates exports all user templates.
// GET /api/v1/templates/export
func (h *Handlers) ExportTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.Templates.ExportTemplates()
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, templates, http.StatusOK)
}
