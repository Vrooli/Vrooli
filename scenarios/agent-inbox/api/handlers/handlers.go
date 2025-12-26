// Package handlers provides HTTP handlers for the Agent Inbox API.
// Handlers are organized by domain responsibility:
//   - health.go: Health check endpoints
//   - chat.go: Chat CRUD operations
//   - message.go: Message and chat state operations
//   - label.go: Label management
//   - ai.go: AI completion, models, tools, streaming
//   - errors.go: Structured error responses
package handlers

import (
	"encoding/json"
	"net/http"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/persistence"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers with access to all dependencies.
// This struct enables dependency injection for testing.
type Handlers struct {
	Repo         *persistence.Repository
	OllamaClient *integrations.OllamaClient
}

// New creates a new Handlers instance with all dependencies.
func New(repo *persistence.Repository, ollamaClient *integrations.OllamaClient) *Handlers {
	return &Handlers{
		Repo:         repo,
		OllamaClient: ollamaClient,
	}
}

// RegisterRoutes sets up all API routes on the given router.
func (h *Handlers) RegisterRoutes(r *mux.Router) {
	// Health
	r.HandleFunc("/health", h.Health).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/health", h.Health).Methods("GET", "OPTIONS")

	// Chats
	r.HandleFunc("/api/v1/chats", h.ListChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats", h.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/search", h.SearchChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}", h.GetChat).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}", h.UpdateChat).Methods("PATCH", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}", h.DeleteChat).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/export", h.ExportChat).Methods("GET", "OPTIONS")

	// Messages and chat state
	r.HandleFunc("/api/v1/chats/{id}/messages", h.AddMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/read", h.ToggleRead).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/archive", h.ToggleArchive).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/star", h.ToggleStar).Methods("POST", "OPTIONS")

	// Labels
	r.HandleFunc("/api/v1/labels", h.ListLabels).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/labels", h.CreateLabel).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/labels/{id}", h.UpdateLabel).Methods("PATCH", "OPTIONS")
	r.HandleFunc("/api/v1/labels/{id}", h.DeleteLabel).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{chatId}/labels/{labelId}", h.AssignLabel).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{chatId}/labels/{labelId}", h.RemoveLabel).Methods("DELETE", "OPTIONS")

	// AI / OpenRouter
	r.HandleFunc("/api/v1/models", h.ListModels).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/tools", h.ListTools).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/complete", h.ChatComplete).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/tool-calls", h.ListChatToolCalls).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/auto-name", h.AutoName).Methods("POST", "OPTIONS")

	// Usage tracking
	r.HandleFunc("/api/v1/usage", h.GetUsageStats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/usage/records", h.GetUsageRecords).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/usage", h.GetChatUsageStats).Methods("GET", "OPTIONS")
}

// Response helpers

// JSONResponse writes a JSON response with the given status code.
func (h *Handlers) JSONResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// JSONError writes an error response with the given status code.
// Deprecated: Use WriteAppError for structured errors.
func (h *Handlers) JSONError(w http.ResponseWriter, message string, status int) {
	h.JSONResponse(w, map[string]string{"error": message}, status)
}

// Validation helpers

// ParseUUID extracts and validates a UUID from route variables.
// Returns empty string and writes structured error response if invalid.
func (h *Handlers) ParseUUID(w http.ResponseWriter, r *http.Request, key string) string {
	id := mux.Vars(r)[key]
	if _, err := uuid.Parse(id); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidUUID(key))
		return ""
	}
	return id
}
