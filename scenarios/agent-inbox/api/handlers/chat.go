// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains handlers for chat CRUD operations.
//
// Error Handling Pattern:
//   - Validation errors use domain.ErrInvalidInput or specific validators
//   - Not found errors use domain.ErrChatNotFound
//   - Database errors use domain.ErrDatabaseError
//   - All errors are written using WriteAppError for structured responses
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"agent-inbox/domain"
	"agent-inbox/middleware"
)

// ListChats returns all chats matching the given filters.
func (h *Handlers) ListChats(w http.ResponseWriter, r *http.Request) {
	archived := r.URL.Query().Get("archived") == "true"
	starred := r.URL.Query().Get("starred") == "true"

	chats, err := h.Repo.ListChats(r.Context(), archived, starred)
	if err != nil {
		log.Printf("[ERROR] [%s] ListChats failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("list chats", err))
		return
	}

	h.JSONResponse(w, chats, http.StatusOK)
}

// CreateChat creates a new chat.
func (h *Handlers) CreateChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Model    string `json:"model"`
		ViewMode string `json:"view_mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	// Defaults
	if req.Name == "" {
		req.Name = "New Chat"
	}
	if req.Model == "" {
		req.Model = "anthropic/claude-3.5-sonnet"
	}
	if req.ViewMode == "" {
		req.ViewMode = domain.ViewModeBubble
	}

	// Validate using centralized validation
	if result := domain.ValidateChatCreate(req.Name, req.Model, req.ViewMode); !result.Valid {
		h.WriteAppError(w, r, domain.ErrInvalidInput(result.Message))
		return
	}

	chat, err := h.Repo.CreateChat(r.Context(), req.Name, req.Model, req.ViewMode)
	if err != nil {
		log.Printf("[ERROR] [%s] CreateChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("create chat", err))
		return
	}

	h.JSONResponse(w, chat, http.StatusCreated)
}

// GetChat retrieves a chat with its messages.
func (h *Handlers) GetChat(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	chat, err := h.Repo.GetChat(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] GetChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}
	if chat == nil {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	messages, err := h.Repo.GetMessages(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] GetMessages failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get messages", err))
		return
	}

	// Fetch attachments for all messages
	if len(messages) > 0 {
		messageIDs := make([]string, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID
		}

		attachmentsByMsgID, err := h.Repo.GetAttachmentsForMessages(r.Context(), messageIDs)
		if err != nil {
			log.Printf("[WARN] [%s] GetAttachmentsForMessages failed: %v", middleware.GetRequestID(r.Context()), err)
			// Non-fatal: continue without attachments
		} else {
			// Populate attachments on each message and add URLs
			for i := range messages {
				if attachments, ok := attachmentsByMsgID[messages[i].ID]; ok {
					// Add URL for each attachment
					for j := range attachments {
						attachments[j].URL = h.Storage.GetFileURL(attachments[j].StoragePath)
					}
					messages[i].Attachments = attachments
				}
			}
		}
	}

	// Fetch tool call records for status/result info
	toolCallRecords, err := h.Repo.ListToolCallsForChat(r.Context(), chatID)
	if err != nil {
		log.Printf("[WARN] [%s] ListToolCallsForChat failed: %v", middleware.GetRequestID(r.Context()), err)
		// Non-fatal: continue without tool call records
		toolCallRecords = []domain.ToolCallRecord{}
	}

	h.JSONResponse(w, map[string]interface{}{
		"chat":              chat,
		"messages":          messages,
		"tool_call_records": toolCallRecords,
	}, http.StatusOK)
}

// UpdateChat updates a chat's name, model, or tools_enabled.
func (h *Handlers) UpdateChat(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req struct {
		Name         *string `json:"name"`
		Model        *string `json:"model"`
		ToolsEnabled *bool   `json:"tools_enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	// Validate using centralized validation (tools_enabled is always valid if provided)
	if result := domain.ValidateChatUpdate(req.Name, req.Model, req.ToolsEnabled); !result.Valid {
		h.WriteAppError(w, r, domain.NewError(
			domain.ErrCodeNoFieldsToUpdate,
			domain.CategoryValidation,
			result.Message,
			domain.ActionCorrectInput,
		))
		return
	}

	chat, err := h.Repo.UpdateChat(r.Context(), chatID, req.Name, req.Model, req.ToolsEnabled)
	if err != nil {
		log.Printf("[ERROR] [%s] UpdateChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("update chat", err))
		return
	}
	if chat == nil {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	h.JSONResponse(w, chat, http.StatusOK)
}

// DeleteChat removes a chat.
func (h *Handlers) DeleteChat(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	deleted, err := h.Repo.DeleteChat(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] DeleteChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("delete chat", err))
		return
	}
	if !deleted {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchChats performs full-text search across chat names and message content.
// Query parameters:
//   - q: the search query (required)
//   - limit: maximum number of results (optional, default 20)
func (h *Handlers) SearchChats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("search query is required"))
		return
	}

	// Parse limit with default
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := parseInt(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	results, err := h.Repo.SearchChats(r.Context(), query, limit)
	if err != nil {
		log.Printf("[ERROR] [%s] SearchChats failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("search chats", err))
		return
	}

	h.JSONResponse(w, results, http.StatusOK)
}

// parseInt is a helper to parse integers from query params.
func parseInt(s string) (int, error) {
	var n int
	err := json.Unmarshal([]byte(s), &n)
	return n, err
}

// ExportChat exports a chat in the requested format (markdown, json, or txt).
// Query parameters:
//   - format: export format (required) - "markdown", "json", or "txt"
//
// Returns the chat content with appropriate Content-Type and Content-Disposition headers.
func (h *Handlers) ExportChat(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "markdown" // Default to markdown
	}

	// Validate format
	validFormats := map[string]bool{"markdown": true, "json": true, "txt": true}
	if !validFormats[format] {
		h.WriteAppError(w, r, domain.ErrInvalidInput("format must be 'markdown', 'json', or 'txt'"))
		return
	}

	// Fetch chat and messages
	chat, err := h.Repo.GetChat(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] ExportChat GetChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}
	if chat == nil {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	messages, err := h.Repo.GetMessages(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] ExportChat GetMessages failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get messages", err))
		return
	}

	// Generate export content
	var content string
	var contentType string
	var fileExt string

	switch format {
	case "markdown":
		content = formatMarkdown(chat, messages)
		contentType = "text/markdown; charset=utf-8"
		fileExt = "md"
	case "json":
		jsonData, err := json.MarshalIndent(map[string]interface{}{
			"chat":     chat,
			"messages": messages,
		}, "", "  ")
		if err != nil {
			log.Printf("[ERROR] [%s] ExportChat JSON marshal failed: %v", middleware.GetRequestID(r.Context()), err)
			h.WriteAppError(w, r, domain.ErrDatabaseError("export chat", err))
			return
		}
		content = string(jsonData)
		contentType = "application/json; charset=utf-8"
		fileExt = "json"
	case "txt":
		content = formatPlainText(chat, messages)
		contentType = "text/plain; charset=utf-8"
		fileExt = "txt"
	}

	// Set headers for file download
	filename := sanitizeFilename(chat.Name) + "." + fileExt
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

// formatMarkdown formats a chat and messages as markdown.
func formatMarkdown(chat *domain.Chat, messages []domain.Message) string {
	var sb strings.Builder

	sb.WriteString("# ")
	sb.WriteString(chat.Name)
	sb.WriteString("\n\n")

	sb.WriteString("**Model:** ")
	sb.WriteString(chat.Model)
	sb.WriteString("  \n")
	sb.WriteString("**Created:** ")
	sb.WriteString(chat.CreatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("  \n")
	sb.WriteString("**Updated:** ")
	sb.WriteString(chat.UpdatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n\n---\n\n")

	for _, msg := range messages {
		switch msg.Role {
		case domain.RoleUser:
			sb.WriteString("## User\n\n")
		case domain.RoleAssistant:
			sb.WriteString("## Assistant")
			if msg.Model != "" {
				sb.WriteString(" (")
				sb.WriteString(msg.Model)
				sb.WriteString(")")
			}
			sb.WriteString("\n\n")
		case domain.RoleSystem:
			sb.WriteString("## System\n\n")
		case domain.RoleTool:
			sb.WriteString("## Tool Response")
			if msg.ToolCallID != "" {
				sb.WriteString(" (")
				sb.WriteString(msg.ToolCallID)
				sb.WriteString(")")
			}
			sb.WriteString("\n\n")
		}

		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")

		// Include tool calls if present
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("**Tool Calls:**\n\n")
			for _, tc := range msg.ToolCalls {
				sb.WriteString("- `")
				sb.WriteString(tc.Function.Name)
				sb.WriteString("`")
				if tc.Function.Arguments != "" && tc.Function.Arguments != "{}" {
					sb.WriteString(": `")
					sb.WriteString(tc.Function.Arguments)
					sb.WriteString("`")
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatPlainText formats a chat and messages as plain text.
func formatPlainText(chat *domain.Chat, messages []domain.Message) string {
	var sb strings.Builder

	sb.WriteString(chat.Name)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("=", len(chat.Name)))
	sb.WriteString("\n\n")

	sb.WriteString("Model: ")
	sb.WriteString(chat.Model)
	sb.WriteString("\n")
	sb.WriteString("Created: ")
	sb.WriteString(chat.CreatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")
	sb.WriteString("Updated: ")
	sb.WriteString(chat.UpdatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n\n")
	sb.WriteString(strings.Repeat("-", 40))
	sb.WriteString("\n\n")

	for _, msg := range messages {
		timestamp := msg.CreatedAt.Format("15:04:05")

		switch msg.Role {
		case domain.RoleUser:
			sb.WriteString("[")
			sb.WriteString(timestamp)
			sb.WriteString("] User:\n")
		case domain.RoleAssistant:
			sb.WriteString("[")
			sb.WriteString(timestamp)
			sb.WriteString("] Assistant:\n")
		case domain.RoleSystem:
			sb.WriteString("[")
			sb.WriteString(timestamp)
			sb.WriteString("] System:\n")
		case domain.RoleTool:
			sb.WriteString("[")
			sb.WriteString(timestamp)
			sb.WriteString("] Tool:\n")
		}

		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// sanitizeFilename removes invalid filename characters.
func sanitizeFilename(name string) string {
	// Replace spaces and problematic characters
	result := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, name)

	// Trim and limit length
	result = strings.TrimSpace(result)
	if len(result) > 50 {
		result = result[:50]
	}
	if result == "" {
		result = "chat"
	}
	return result
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
