// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains handlers for basic chat CRUD operations.
//
// Error Handling Pattern:
//   - Validation errors use domain.ErrInvalidInput or specific validators
//   - Not found errors use domain.ErrChatNotFound
//   - Database errors use domain.ErrDatabaseError
//   - All errors are written using WriteAppError for structured responses
package handlers

import (
	"agent-inbox/config"
	"agent-inbox/domain"
	"agent-inbox/middleware"
	"encoding/json"
	"log"
	"net/http"
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
		ChatMode string `json:"chat_mode"`
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
		// No per-request model: resolve the operator default via policy role.
		// There is no concrete code default; if the OpenRouter resource is
		// unavailable and no explicit override is configured, fail clearly.
		resolved, err := config.ResolveDefaultChatModel(r.Context())
		if err != nil {
			log.Printf("[ERROR] [%s] CreateChat default-model resolution failed: %v", middleware.GetRequestID(r.Context()), err)
			h.WriteAppError(w, r, domain.ErrOpenRouterUnavailable(err))
			return
		}
		req.Model = resolved
	}
	if req.ViewMode == "" {
		req.ViewMode = domain.ViewModeBubble
	}
	if req.ChatMode == "" {
		req.ChatMode = domain.ChatModeLLM
	}

	// Validate using centralized validation
	if result := domain.ValidateChatCreate(req.Name, req.Model, req.ViewMode); !result.Valid {
		h.WriteAppError(w, r, domain.ErrInvalidInput(result.Message))
		return
	}

	// Validate chat mode
	if !domain.IsValidChatMode(req.ChatMode) {
		h.WriteAppError(w, r, domain.ErrInvalidInput("invalid chat_mode: must be 'llm' or 'agent'"))
		return
	}

	chat, err := h.Repo.CreateChat(r.Context(), req.Name, req.Model, req.ViewMode, req.ChatMode)
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

// parseInt is a helper to parse integers from query params.
func parseInt(s string) (int, error) {
	var n int
	err := json.Unmarshal([]byte(s), &n)
	return n, err
}
