// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains handlers for chat search and export operations.
package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/middleware"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// SearchChats performs full-text search across chat names and message content.
// Query parameters:
//   - q: the search query (required)
//   - limit: maximum number of results (optional, default 20)
//   - per_chat: max message matches per chat, 1–10 (optional, default 1)
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

	// Parse per_chat with default
	perChat := 1
	if perChatStr := r.URL.Query().Get("per_chat"); perChatStr != "" {
		if parsed, err := parseInt(perChatStr); err == nil && parsed > 0 && parsed <= 10 {
			perChat = parsed
		}
	}

	// Parse search options
	caseSensitive := r.URL.Query().Get("case_sensitive") == "true"
	wholeWord := r.URL.Query().Get("whole_word") == "true"
	useRegex := r.URL.Query().Get("regex") == "true"

	results, err := h.Repo.SearchChats(r.Context(), query, limit, perChat, caseSensitive, wholeWord, useRegex)
	if err != nil {
		// Check if it's an invalid regex pattern
		if useRegex && strings.Contains(err.Error(), "invalid search pattern") {
			h.WriteAppError(w, r, domain.ErrInvalidInput("invalid regex pattern: "+err.Error()))
			return
		}
		log.Printf("[ERROR] [%s] SearchChats failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("search chats", err))
		return
	}

	h.JSONResponse(w, results, http.StatusOK)
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
	_, _ = w.Write([]byte(content))
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
