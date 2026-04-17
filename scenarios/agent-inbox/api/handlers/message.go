package handlers

import (
	"agent-inbox/domain"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// AddMessage adds a message to a chat.
// For branching support, the message is parented to the current active_leaf_message_id.
func (h *Handlers) AddMessage(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req struct {
		Role            string   `json:"role"`
		Content         string   `json:"content"`
		Model           string   `json:"model"`
		TokenCount      int      `json:"token_count"`
		ToolCallID      string   `json:"tool_call_id,omitempty"`
		ParentMessageID string   `json:"parent_message_id,omitempty"` // Optional override for explicit parent
		AttachmentIDs   []string `json:"attachment_ids,omitempty"`    // IDs of uploaded attachments to link
		WebSearch       *bool    `json:"web_search,omitempty"`        // Per-message web search override
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG] AddMessage: web_search=%v, attachment_ids=%v", req.WebSearch, req.AttachmentIDs)

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

	// Determine parent message ID for branching
	// If not explicitly provided, use the current active leaf
	parentMessageID := req.ParentMessageID
	if parentMessageID == "" {
		parentMessageID, _ = h.Repo.GetActiveLeaf(r.Context(), chatID)
	}

	msg, err := h.Repo.CreateMessage(r.Context(), chatID, req.Role, req.Content, req.Model, req.ToolCallID, req.TokenCount, parentMessageID, req.WebSearch)
	if err != nil {
		h.JSONError(w, "Failed to add message", http.StatusInternalServerError)
		return
	}

	// Link attachments to the message if provided
	if len(req.AttachmentIDs) > 0 {
		if linkErr := h.Repo.LinkAttachmentsToMessage(r.Context(), msg.ID, req.AttachmentIDs); linkErr != nil {
			// Log but don't fail the request - message was created successfully
			// The attachments may have expired or been deleted
			log.Printf("[WARN] Failed to link attachments to message: %v", linkErr)
		}
	}

	// Update active leaf to point to this new message
	_ = h.Repo.SetActiveLeaf(r.Context(), chatID, msg.ID) // Ignore error: leaf update is best-effort

	// Update chat preview using centralized truncation
	preview := domain.TruncatePreview(req.Content)
	_ = h.Repo.UpdateChatPreview(r.Context(), chatID, preview, req.Role == domain.RoleAssistant) // Ignore error: preview update is best-effort

	// Backend-owned auto-naming: only applies while chat has default name.
	if req.Role == domain.RoleUser {
		h.maybeAutoNameChat(r.Context(), chatID)
	}

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
	_ = json.NewDecoder(r.Body).Decode(&req) // Ignore error: value is optional

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
