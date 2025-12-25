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
		req.Model = "claude-3-5-sonnet-20241022"
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

	h.JSONResponse(w, map[string]interface{}{
		"chat":     chat,
		"messages": messages,
	}, http.StatusOK)
}

// UpdateChat updates a chat's name and/or model.
func (h *Handlers) UpdateChat(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req struct {
		Name  *string `json:"name"`
		Model *string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	// Validate using centralized validation
	if result := domain.ValidateChatUpdate(req.Name, req.Model); !result.Valid {
		h.WriteAppError(w, r, domain.NewError(
			domain.ErrCodeNoFieldsToUpdate,
			domain.CategoryValidation,
			result.Message,
			domain.ActionCorrectInput,
		))
		return
	}

	chat, err := h.Repo.UpdateChat(r.Context(), chatID, req.Name, req.Model)
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
