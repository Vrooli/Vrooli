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
	"agent-inbox/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
)

// Handlers provides HTTP handlers with access to all dependencies.
// This struct enables dependency injection for testing.
type Handlers struct {
	Repo          *persistence.Repository
	OllamaClient  *integrations.OllamaClient
	ToolRegistry  *services.ToolRegistry
	ModelRegistry *services.ModelRegistry
	Storage       services.StorageService
	ToolExecutor  *integrations.ToolExecutor
	AsyncTracker  *services.AsyncTrackerService
	Templates     *services.TemplatesService
	Skills        *services.SkillsService
}

// New creates a new Handlers instance with all dependencies.
func New(repo *persistence.Repository, ollamaClient *integrations.OllamaClient, storage services.StorageService) *Handlers {
	toolExecutor := integrations.NewToolExecutor()
	toolRegistry := services.NewToolRegistry(repo, toolExecutor)
	asyncTracker := services.NewAsyncTrackerService(toolRegistry, toolExecutor)

	return &Handlers{
		Repo:          repo,
		OllamaClient:  ollamaClient,
		ToolRegistry:  toolRegistry,
		ModelRegistry: services.NewModelRegistry(),
		Storage:       storage,
		ToolExecutor:  toolExecutor,
		AsyncTracker:  asyncTracker,
	}
}

// RegisterRoutes sets up all API routes on the given router.
func (h *Handlers) RegisterRoutes(r *mux.Router) {
	// Health
	healthHandler := health.New().Version("1.0.0").Check(health.DB(h.Repo.DB()), health.Critical).Handler()
	r.HandleFunc("/health", healthHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/health", healthHandler).Methods("GET", "OPTIONS")

	// Chats
	r.HandleFunc("/api/v1/chats", h.ListChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats", h.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/bulk", h.BulkOperation).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/search", h.SearchChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}", h.GetChat).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}", h.UpdateChat).Methods("PATCH", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}", h.DeleteChat).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/export", h.ExportChat).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/fork", h.ForkChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/active-template", h.SetActiveTemplate).Methods("PATCH", "OPTIONS")

	// Messages and chat state
	r.HandleFunc("/api/v1/chats/{id}/messages", h.AddMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/messages/{msgId}/regenerate", h.RegenerateMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/messages/{msgId}/edit", h.EditMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/messages/{msgId}/select", h.SelectBranch).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/messages/{msgId}/siblings", h.GetMessageSiblings).Methods("GET", "OPTIONS")
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

	// Tool Configuration
	r.HandleFunc("/api/v1/tools/set", h.GetToolSet).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/tools/scenarios", h.GetScenarioStatuses).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/tools/config", h.SetToolEnabled).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/tools/config", h.ResetToolConfig).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/tools/config/approval", h.SetToolApproval).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/tools/refresh", h.RefreshTools).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/tools/sync", h.SyncTools).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/tools/execute", h.ExecuteToolManually).Methods("POST", "OPTIONS")

	// Tool Call Approvals
	r.HandleFunc("/api/v1/chats/{id}/pending-approvals", h.GetPendingApprovals).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/tool-calls/{id}/approve", h.ApproveToolCall).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/tool-calls/{id}/reject", h.RejectToolCall).Methods("POST", "OPTIONS")

	// Async Tool Operations (SSE for real-time updates)
	r.HandleFunc("/api/v1/chats/{id}/async-status", h.StreamAsyncStatus).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/async-operations", h.GetAsyncOperations).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/async-operations/{toolCallId}/cancel", h.CancelAsyncOperation).Methods("POST", "OPTIONS")

	// Settings
	r.HandleFunc("/api/v1/settings/yolo-mode", h.GetYoloMode).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/settings/yolo-mode", h.SetYoloMode).Methods("POST", "OPTIONS")

	// Usage tracking
	r.HandleFunc("/api/v1/usage", h.GetUsageStats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/usage/records", h.GetUsageRecords).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/chats/{id}/usage", h.GetChatUsageStats).Methods("GET", "OPTIONS")

	// Utilities
	r.HandleFunc("/api/v1/link-preview", h.GetLinkPreview).Methods("GET", "OPTIONS")

	// Templates
	r.HandleFunc("/api/v1/templates", h.ListTemplates).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/templates", h.CreateTemplate).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/templates/import", h.ImportTemplates).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/templates/export", h.ExportTemplates).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/templates/{id}", h.GetTemplate).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/templates/{id}", h.UpdateTemplate).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/v1/templates/{id}", h.DeleteTemplate).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/templates/{id}/reset", h.ResetTemplate).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/templates/{id}/update-default", h.UpdateDefaultTemplate).Methods("PUT", "OPTIONS")

	// Skills
	r.HandleFunc("/api/v1/skills", h.ListSkills).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/skills", h.CreateSkill).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/skills/import", h.ImportSkills).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/skills/export", h.ExportSkills).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/skills/{id}", h.GetSkill).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/skills/{id}", h.UpdateSkill).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/v1/skills/{id}", h.DeleteSkill).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/v1/skills/{id}/reset", h.ResetSkill).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/skills/{id}/update-default", h.UpdateDefaultSkill).Methods("PUT", "OPTIONS")
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

// NewCompletionService creates a completion service with async tracker configured.
// This is the preferred way to create completion services in handlers.
func (h *Handlers) NewCompletionService() *services.CompletionService {
	svc := services.NewCompletionService(h.Repo, h.Storage)
	svc.SetAsyncTracker(h.AsyncTracker)
	return svc
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
