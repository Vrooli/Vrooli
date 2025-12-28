package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"agent-inbox/domain"

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
		}
	}

	// Update active leaf to point to this new message
	h.Repo.SetActiveLeaf(r.Context(), chatID, msg.ID)

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

// RegenerateMessage regenerates an assistant response, creating a new sibling.
// This is the ChatGPT-style regenerate that preserves the original and creates alternatives.
func (h *Handlers) RegenerateMessage(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}
	msgID := h.ParseUUID(w, r, "msgId")
	if msgID == "" {
		return
	}

	// Get the message to regenerate
	msg, err := h.Repo.GetMessageByID(r.Context(), msgID)
	if err != nil {
		h.JSONError(w, "Failed to get message", http.StatusInternalServerError)
		return
	}
	if msg == nil {
		h.JSONError(w, "Message not found", http.StatusNotFound)
		return
	}

	// Validate message belongs to this chat
	if msg.ChatID != chatID {
		h.JSONError(w, "Message does not belong to this chat", http.StatusNotFound)
		return
	}

	// Validate it's an assistant message
	if msg.Role != domain.RoleAssistant {
		h.JSONError(w, "Can only regenerate assistant messages", http.StatusBadRequest)
		return
	}

	// The parent of the new message should be the same as the original's parent
	// (i.e., the user message that triggered the original response)
	parentMessageID := msg.ParentMessageID

	// Set active leaf to the parent so the new response is a sibling of the original
	if parentMessageID != "" {
		h.Repo.SetActiveLeaf(r.Context(), chatID, parentMessageID)
	}

	// Now trigger a new completion - this will create a sibling response
	// Redirect to the normal completion flow
	h.ChatComplete(w, r)
}

// EditMessage edits a user message by creating a new sibling with updated content.
// This preserves the original message (branch-based editing).
// The frontend should call the completion endpoint separately after this returns.
func (h *Handlers) EditMessage(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}
	msgID := h.ParseUUID(w, r, "msgId")
	if msgID == "" {
		return
	}

	var req struct {
		Content       string   `json:"content"`
		AttachmentIDs []string `json:"attachment_ids,omitempty"`
		WebSearch     *bool    `json:"web_search,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate content is not empty
	if strings.TrimSpace(req.Content) == "" {
		h.JSONError(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	// Get the original message
	msg, err := h.Repo.GetMessageByID(r.Context(), msgID)
	if err != nil {
		h.JSONError(w, "Failed to get message", http.StatusInternalServerError)
		return
	}
	if msg == nil {
		h.JSONError(w, "Message not found", http.StatusNotFound)
		return
	}

	// Validate message belongs to this chat
	if msg.ChatID != chatID {
		h.JSONError(w, "Message does not belong to this chat", http.StatusNotFound)
		return
	}

	// Only allow editing user messages
	if msg.Role != domain.RoleUser {
		h.JSONError(w, "Can only edit user messages", http.StatusBadRequest)
		return
	}

	// The new message shares the same parent as the original, making it a sibling
	parentMessageID := msg.ParentMessageID

	// Create the new user message as a sibling of the original
	newMsg, err := h.Repo.CreateMessage(r.Context(), chatID, domain.RoleUser, req.Content, "", "", 0, parentMessageID, req.WebSearch)
	if err != nil {
		h.JSONError(w, "Failed to create edited message", http.StatusInternalServerError)
		return
	}

	// Link attachments if provided
	if len(req.AttachmentIDs) > 0 {
		if linkErr := h.Repo.LinkAttachmentsToMessage(r.Context(), newMsg.ID, req.AttachmentIDs); linkErr != nil {
			log.Printf("[WARN] Failed to link attachments to edited message: %v", linkErr)
		}
	}

	// Update active leaf to point to the new message
	h.Repo.SetActiveLeaf(r.Context(), chatID, newMsg.ID)

	// Update chat preview with the new content
	preview := domain.TruncatePreview(req.Content)
	h.Repo.UpdateChatPreview(r.Context(), chatID, preview, false)

	// Return the new message - frontend will trigger completion separately
	h.JSONResponse(w, newMsg, http.StatusOK)
}

// SelectBranch changes the active branch to the specified message.
// This allows users to navigate between alternative responses.
func (h *Handlers) SelectBranch(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}
	msgID := h.ParseUUID(w, r, "msgId")
	if msgID == "" {
		return
	}

	// Get the message
	msg, err := h.Repo.GetMessageByID(r.Context(), msgID)
	if err != nil {
		h.JSONError(w, "Failed to get message", http.StatusInternalServerError)
		return
	}
	if msg == nil {
		h.JSONError(w, "Message not found", http.StatusNotFound)
		return
	}

	// Validate message belongs to this chat
	if msg.ChatID != chatID {
		h.JSONError(w, "Message does not belong to this chat", http.StatusNotFound)
		return
	}

	// Update active leaf to point to this message
	// This changes which branch is "active" for the conversation
	if err := h.Repo.SetActiveLeaf(r.Context(), chatID, msgID); err != nil {
		h.JSONError(w, "Failed to select branch", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]string{"active_leaf_message_id": msgID}, http.StatusOK)
}

// GetMessageSiblings returns all sibling messages (alternatives) for a given message.
func (h *Handlers) GetMessageSiblings(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}
	msgID := h.ParseUUID(w, r, "msgId")
	if msgID == "" {
		return
	}

	// Validate message belongs to this chat first
	msg, err := h.Repo.GetMessageByID(r.Context(), msgID)
	if err != nil {
		h.JSONError(w, "Failed to get message", http.StatusInternalServerError)
		return
	}
	if msg == nil || msg.ChatID != chatID {
		h.JSONError(w, "Message not found", http.StatusNotFound)
		return
	}

	siblings, err := h.Repo.GetMessageSiblings(r.Context(), msgID)
	if err != nil {
		h.JSONError(w, "Failed to get siblings", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, siblings, http.StatusOK)
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
