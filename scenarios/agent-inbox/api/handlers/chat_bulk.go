// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains handlers for bulk chat operations, forking, and template management.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"agent-inbox/domain"
	"agent-inbox/middleware"
)

// DeleteArchivedChats removes all archived chats.
// DELETE /api/v1/chats/archived
func (h *Handlers) DeleteArchivedChats(w http.ResponseWriter, r *http.Request) {
	count, err := h.Repo.DeleteArchivedChats(r.Context())
	if err != nil {
		log.Printf("[ERROR] [%s] DeleteArchivedChats failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("delete archived chats", err))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"deleted": count,
	}, http.StatusOK)
}

// MarkAllAsRead marks all unread chats as read.
// POST /api/v1/chats/mark-all-read
func (h *Handlers) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	count, err := h.Repo.MarkAllChatsRead(r.Context())
	if err != nil {
		log.Printf("[ERROR] [%s] MarkAllAsRead failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("mark all chats as read", err))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"updated": count,
	}, http.StatusOK)
}

// BulkOperation performs a bulk operation on multiple chats.
// Operations: delete, archive, unarchive, mark_read, mark_unread, add_label, remove_label
func (h *Handlers) BulkOperation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChatIDs   []string `json:"chat_ids"`
		Operation string   `json:"operation"`
		LabelID   string   `json:"label_id,omitempty"` // For add_label/remove_label operations
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	// Validate
	if len(req.ChatIDs) == 0 {
		h.WriteAppError(w, r, domain.ErrInvalidInput("chat_ids is required"))
		return
	}
	if len(req.ChatIDs) > 100 {
		h.WriteAppError(w, r, domain.ErrInvalidInput("maximum 100 chats per bulk operation"))
		return
	}

	validOps := map[string]bool{
		"delete":       true,
		"archive":      true,
		"unarchive":    true,
		"mark_read":    true,
		"mark_unread":  true,
		"add_label":    true,
		"remove_label": true,
	}
	if !validOps[req.Operation] {
		h.WriteAppError(w, r, domain.ErrInvalidInput("invalid operation: "+req.Operation))
		return
	}

	// Label operations require label_id
	if (req.Operation == "add_label" || req.Operation == "remove_label") && req.LabelID == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("label_id is required for "+req.Operation))
		return
	}

	ctx := r.Context()
	successCount := 0
	failCount := 0

	for _, chatID := range req.ChatIDs {
		var err error
		switch req.Operation {
		case "delete":
			_, err = h.Repo.DeleteChat(ctx, chatID)
		case "archive":
			val := true
			_, err = h.Repo.ToggleChatBool(ctx, chatID, "is_archived", &val)
		case "unarchive":
			val := false
			_, err = h.Repo.ToggleChatBool(ctx, chatID, "is_archived", &val)
		case "mark_read":
			val := true
			_, err = h.Repo.ToggleChatBool(ctx, chatID, "is_read", &val)
		case "mark_unread":
			val := false
			_, err = h.Repo.ToggleChatBool(ctx, chatID, "is_read", &val)
		case "add_label":
			err = h.Repo.AssignLabel(ctx, chatID, req.LabelID)
		case "remove_label":
			_, err = h.Repo.RemoveLabel(ctx, chatID, req.LabelID)
		}

		if err != nil {
			failCount++
			log.Printf("[WARN] [%s] BulkOperation %s failed for chat %s: %v",
				middleware.GetRequestID(ctx), req.Operation, chatID, err)
		} else {
			successCount++
		}
	}

	h.JSONResponse(w, map[string]interface{}{
		"success_count": successCount,
		"fail_count":    failCount,
		"total":         len(req.ChatIDs),
	}, http.StatusOK)
}

// ForkChat creates a new chat from an existing one, copying messages up to a specified point.
// POST /api/v1/chats/{id}/fork
// Body: { "message_id": "uuid" } - Fork from this message (includes it and all ancestors)
func (h *Handlers) ForkChat(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req struct {
		MessageID string `json:"message_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	if req.MessageID == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("message_id is required"))
		return
	}

	ctx := r.Context()

	// Get the source chat
	sourceChat, err := h.Repo.GetChat(ctx, chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] ForkChat GetChat failed: %v", middleware.GetRequestID(ctx), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}
	if sourceChat == nil {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	// Fork the chat
	newChat, err := h.Repo.ForkChat(ctx, chatID, req.MessageID, sourceChat.Name+" (fork)", sourceChat.Model)
	if err != nil {
		log.Printf("[ERROR] [%s] ForkChat failed: %v", middleware.GetRequestID(ctx), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("fork chat", err))
		return
	}

	h.JSONResponse(w, newChat, http.StatusCreated)
}

// SetActiveTemplate sets or clears the active template for a chat.
// PATCH /api/v1/chats/{id}/active-template
// Body: { "template_id": "template-123", "tool_ids": ["scenario:tool1", "scenario:tool2"] }
// To clear: { "template_id": "" } or { "template_id": null }
func (h *Handlers) SetActiveTemplate(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req struct {
		TemplateID string   `json:"template_id"`
		ToolIDs    []string `json:"tool_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	ctx := r.Context()

	// Verify chat exists
	exists, err := h.Repo.ChatExists(ctx, chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] SetActiveTemplate ChatExists failed: %v", middleware.GetRequestID(ctx), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("check chat", err))
		return
	}
	if !exists {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	// If template_id is empty, clear the active template
	if req.TemplateID == "" {
		if err := h.Repo.ClearActiveTemplate(ctx, chatID); err != nil {
			log.Printf("[ERROR] [%s] ClearActiveTemplate failed: %v", middleware.GetRequestID(ctx), err)
			h.WriteAppError(w, r, domain.ErrDatabaseError("clear active template", err))
			return
		}
		h.JSONResponse(w, map[string]interface{}{
			"active_template_id": nil,
		}, http.StatusOK)
		return
	}

	// Set the active template
	if err := h.Repo.SetActiveTemplate(ctx, chatID, req.TemplateID); err != nil {
		log.Printf("[ERROR] [%s] SetActiveTemplate failed: %v", middleware.GetRequestID(ctx), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("set active template", err))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"active_template_id": req.TemplateID,
	}, http.StatusOK)
}
