// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains message branching, editing, and regeneration handlers.
package handlers

import (
	"agent-inbox/domain"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

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
		_ = h.Repo.SetActiveLeaf(r.Context(), chatID, parentMessageID) // Ignore error: leaf update is best-effort
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
	_ = h.Repo.SetActiveLeaf(r.Context(), chatID, newMsg.ID) // Ignore error: leaf update is best-effort

	// Update chat preview with the new content
	preview := domain.TruncatePreview(req.Content)
	_ = h.Repo.UpdateChatPreview(r.Context(), chatID, preview, false) // Ignore error: preview update is best-effort

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
