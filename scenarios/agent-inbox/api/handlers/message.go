package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-inbox/domain"

	"github.com/gorilla/mux"
)

// AddMessage adds a message to a chat.
func (h *Handlers) AddMessage(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		Model      string `json:"model"`
		TokenCount int    `json:"token_count"`
		ToolCallID string `json:"tool_call_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate message input using centralized validation
	if result := domain.ValidateMessageInput(req.Role, req.Content, req.ToolCallID); !result.Valid {
		h.JSONError(w, result.Message, http.StatusBadRequest)
		return
	}

	exists, err := h.Repo.ChatExists(r.Context(), chatID)
	if err != nil || !exists {
		h.JSONError(w, "Chat not found", http.StatusNotFound)
		return
	}

	msg, err := h.Repo.CreateMessage(r.Context(), chatID, req.Role, req.Content, req.Model, req.ToolCallID, req.TokenCount)
	if err != nil {
		h.JSONError(w, "Failed to add message", http.StatusInternalServerError)
		return
	}

	// Update chat preview using centralized truncation
	preview := domain.TruncatePreview(req.Content)
	h.Repo.UpdateChatPreview(r.Context(), chatID, preview, req.Role == domain.RoleAssistant)

	h.JSONResponse(w, msg, http.StatusCreated)
}

// ToggleRead toggles the read status of a chat.
func (h *Handlers) ToggleRead(w http.ResponseWriter, r *http.Request) {
	h.toggleBool(w, r, "is_read")
}

// ToggleArchive toggles the archive status of a chat.
func (h *Handlers) ToggleArchive(w http.ResponseWriter, r *http.Request) {
	h.toggleBool(w, r, "is_archived")
}

// ToggleStar toggles the starred status of a chat.
func (h *Handlers) ToggleStar(w http.ResponseWriter, r *http.Request) {
	h.toggleBool(w, r, "is_starred")
}

// toggleBool is a helper for toggling boolean chat fields.
func (h *Handlers) toggleBool(w http.ResponseWriter, r *http.Request, field string) {
	chatID := mux.Vars(r)["id"]

	if chatID == "" {
		h.JSONError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Value *bool `json:"value"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	newValue, err := h.Repo.ToggleChatBool(r.Context(), chatID, field, req.Value)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.JSONError(w, "Chat not found", http.StatusNotFound)
		} else {
			h.JSONError(w, "Failed to toggle", http.StatusInternalServerError)
		}
		return
	}

	h.JSONResponse(w, map[string]bool{field: newValue}, http.StatusOK)
}
