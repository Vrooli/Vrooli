package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-inbox/domain"

	"github.com/gorilla/mux"
)

// ListLabels returns all labels.
func (h *Handlers) ListLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := h.Repo.ListLabels(r.Context())
	if err != nil {
		h.JSONError(w, "Failed to list labels", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, labels, http.StatusOK)
}

// CreateLabel creates a new label.
func (h *Handlers) CreateLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply default color
	if req.Color == "" {
		req.Color = "#6366f1"
	}

	// Validate using centralized validation
	if result := domain.ValidateLabelCreate(req.Name, req.Color); !result.Valid {
		h.JSONError(w, result.Message, http.StatusBadRequest)
		return
	}

	label, err := h.Repo.CreateLabel(r.Context(), req.Name, req.Color)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.JSONError(w, "Label with this name already exists", http.StatusConflict)
		} else {
			h.JSONError(w, "Failed to create label", http.StatusInternalServerError)
		}
		return
	}

	h.JSONResponse(w, label, http.StatusCreated)
}

// UpdateLabel updates a label's name and/or color.
func (h *Handlers) UpdateLabel(w http.ResponseWriter, r *http.Request) {
	labelID := h.ParseUUID(w, r, "id")
	if labelID == "" {
		return
	}

	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate using centralized validation
	if result := domain.ValidateLabelUpdate(req.Name, req.Color); !result.Valid {
		h.JSONError(w, result.Message, http.StatusBadRequest)
		return
	}

	label, err := h.Repo.UpdateLabel(r.Context(), labelID, req.Name, req.Color)
	if err != nil {
		h.JSONError(w, "Failed to update label", http.StatusInternalServerError)
		return
	}
	if label == nil {
		h.JSONError(w, "Label not found", http.StatusNotFound)
		return
	}

	h.JSONResponse(w, label, http.StatusOK)
}

// DeleteLabel removes a label.
func (h *Handlers) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	labelID := h.ParseUUID(w, r, "id")
	if labelID == "" {
		return
	}

	deleted, err := h.Repo.DeleteLabel(r.Context(), labelID)
	if err != nil {
		h.JSONError(w, "Failed to delete label", http.StatusInternalServerError)
		return
	}
	if !deleted {
		h.JSONError(w, "Label not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignLabel assigns a label to a chat.
func (h *Handlers) AssignLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := h.ParseUUID(w, r, "chatId")
	if chatID == "" {
		return
	}

	labelID := vars["labelId"]
	if labelID == "" {
		h.JSONError(w, "Invalid label ID", http.StatusBadRequest)
		return
	}

	chatExists, _ := h.Repo.ChatExists(r.Context(), chatID)
	labelExists, _ := h.Repo.LabelExists(r.Context(), labelID)

	if !chatExists {
		h.JSONError(w, "Chat not found", http.StatusNotFound)
		return
	}
	if !labelExists {
		h.JSONError(w, "Label not found", http.StatusNotFound)
		return
	}

	if err := h.Repo.AssignLabel(r.Context(), chatID, labelID); err != nil {
		h.JSONError(w, "Failed to assign label", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]string{"status": "assigned"}, http.StatusOK)
}

// RemoveLabel removes a label from a chat.
func (h *Handlers) RemoveLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := h.ParseUUID(w, r, "chatId")
	if chatID == "" {
		return
	}

	labelID := vars["labelId"]
	if labelID == "" {
		h.JSONError(w, "Invalid label ID", http.StatusBadRequest)
		return
	}

	removed, err := h.Repo.RemoveLabel(r.Context(), chatID, labelID)
	if err != nil {
		h.JSONError(w, "Failed to remove label", http.StatusInternalServerError)
		return
	}
	if !removed {
		h.JSONError(w, "Label assignment not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
