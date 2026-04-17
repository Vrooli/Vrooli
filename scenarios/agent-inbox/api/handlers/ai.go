package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/services"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

// SkillPayload represents a skill with its content for tool context injection.
type SkillPayload struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Content      string   `json:"content"`
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Tags         []string `json:"tags,omitempty"`
	TargetToolID string   `json:"targetToolId,omitempty"`
}

// ToolExecutionStreamResult contains the results of streaming tool execution.
type ToolExecutionStreamResult struct {
	MessageID           string
	HasPendingApprovals bool
	HasAsyncOperations  bool
	AsyncOperations     []domain.AsyncOperationInfo
}

// ChatComplete runs AI completion on a chat.
// This is the main entry point for chat completions.
//
// Decision: Streaming vs Non-streaming
// The client specifies whether to use SSE streaming via ?stream=true.
// Streaming provides real-time updates but requires different response handling.
func (h *Handlers) ChatComplete(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	// Parse optional force_tool query param (format: "scenario:tool_name")
	forcedTool := r.URL.Query().Get("force_tool")

	// Parse optional skills from request body
	// Note: Don't check ContentLength - it may be -1 for chunked transfer encoding
	var skills []SkillPayload
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[WARN] Failed to read request body for skills: %v", err)
		} else if len(bodyBytes) > 0 {
			var reqBody struct {
				Skills []SkillPayload `json:"skills"`
			}
			if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
				log.Printf("[DEBUG] No skills in request body (parse error: %v)", err)
			} else {
				skills = reqBody.Skills
				if len(skills) > 0 {
					log.Printf("[DEBUG] Parsed %d skills from request body", len(skills))
				}
			}
		}
	}

	// Prepare completion request (validates chat exists and has messages)
	svc := h.NewCompletionService()
	svc.SetSkills(skills) // Pass skills to service for tool execution
	prepReq, err := svc.PrepareCompletionRequest(r.Context(), chatID, isStreamingRequest(r), forcedTool)
	if err != nil {
		statusCode := mapCompletionErrorToStatus(err)
		h.JSONError(w, err.Error(), statusCode)
		return
	}

	// Create OpenRouter client
	orClient, err := integrations.NewOpenRouterClient()
	if err != nil {
		h.JSONError(w, "OpenRouter API key not configured", http.StatusServiceUnavailable)
		return
	}

	// Build and execute request
	orReq := &integrations.OpenRouterRequest{
		Model:      prepReq.Model,
		Messages:   prepReq.Messages,
		Stream:     prepReq.Streaming,
		Tools:      prepReq.Tools,
		ToolChoice: prepReq.ToolChoice,
		Plugins:    prepReq.Plugins,
		Modalities: prepReq.Modalities,
	}

	resp, err := orClient.CreateCompletion(r.Context(), orReq)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Route to appropriate handler based on streaming decision
	if prepReq.Streaming {
		h.handleStreamingResponse(w, r, resp.Body, chatID, prepReq.Model, svc)
	} else {
		h.handleNonStreamingResponse(w, r, resp.Body, chatID, prepReq.Model, orClient, svc)
	}
}

// isStreamingRequest checks if the client requested streaming.
func isStreamingRequest(r *http.Request) bool {
	return r.URL.Query().Get("stream") == "true"
}

// mapCompletionErrorToStatus maps service-layer errors to HTTP status codes.
// Decision boundary: Which HTTP status represents this error?
func mapCompletionErrorToStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrChatNotFound):
		return http.StatusNotFound
	case errors.Is(err, services.ErrNoMessages):
		return http.StatusBadRequest
	case errors.Is(err, services.ErrDatabaseError), errors.Is(err, services.ErrMessagesFailed):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
