package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/services"
)

// ListModels returns available AI models.
func (h *Handlers) ListModels(w http.ResponseWriter, r *http.Request) {
	h.JSONResponse(w, integrations.AvailableModels(), http.StatusOK)
}

// ListTools returns available tools for AI.
func (h *Handlers) ListTools(w http.ResponseWriter, r *http.Request) {
	h.JSONResponse(w, integrations.AvailableTools(), http.StatusOK)
}

// ListChatToolCalls returns tool calls for a chat.
func (h *Handlers) ListChatToolCalls(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	records, err := h.Repo.ListToolCallsForChat(r.Context(), chatID)
	if err != nil {
		h.JSONError(w, "Failed to list tool calls", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, records, http.StatusOK)
}

// AutoName generates a name for a chat using Ollama.
// Implements graceful degradation: falls back to default name on Ollama failure.
func (h *Handlers) AutoName(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	messages, err := h.Repo.GetMessages(r.Context(), chatID)
	if err != nil {
		h.JSONError(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	if len(messages) == 0 {
		h.JSONError(w, "No messages in chat to generate name from", http.StatusBadRequest)
		return
	}

	// Build conversation summary using configured limits
	maxMessages, maxContentLen := h.OllamaClient.SummaryLimits()
	summary := buildConversationSummary(messages, maxMessages, maxContentLen)

	// Generate name using Ollama with graceful degradation
	name, err := h.OllamaClient.GenerateChatName(r.Context(), summary)
	if err != nil {
		log.Printf("auto-name failed, using fallback | error=%s", err.Error())
		name = h.OllamaClient.FallbackName()
	}

	// Update the chat
	chat, err := h.Repo.UpdateChat(r.Context(), chatID, &name, nil)
	if err != nil {
		h.JSONError(w, "Failed to update chat name", http.StatusInternalServerError)
		return
	}
	if chat == nil {
		h.JSONError(w, "Chat not found", http.StatusNotFound)
		return
	}

	h.JSONResponse(w, chat, http.StatusOK)
}

// buildConversationSummary creates a text summary of messages for naming.
func buildConversationSummary(messages []domain.Message, maxMessages, maxContentLen int) string {
	var summary strings.Builder
	for i, m := range messages {
		if i >= maxMessages {
			break
		}
		content := m.Content
		if len(content) > maxContentLen {
			content = content[:maxContentLen] + "..."
		}
		summary.WriteString(fmt.Sprintf("%s: %s\n", m.Role, content))
	}
	return summary.String()
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

	// Prepare completion request (validates chat exists and has messages)
	svc := services.NewCompletionService(h.Repo)
	prepReq, err := svc.PrepareCompletionRequest(r.Context(), chatID, isStreamingRequest(r))
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
		Model:    prepReq.Model,
		Messages: prepReq.Messages,
		Stream:   prepReq.Streaming,
		Tools:    prepReq.Tools,
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

// handleStreamingResponse processes a streaming AI response.
// The response arrives as Server-Sent Events (SSE) that must be parsed,
// accumulated, and forwarded to the client.
func (h *Handlers) handleStreamingResponse(w http.ResponseWriter, r *http.Request, body interface{ Read([]byte) (int, error) }, chatID, model string, svc *services.CompletionService) {
	// Setup SSE response
	sw := SetupSSEResponse(w, r)
	if sw == nil {
		h.JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Parse and accumulate streaming chunks
	result := parseStreamingChunks(body, sw)

	// Handle the completion result
	h.handleCompletionResult(r, sw, svc, chatID, model, result)

	sw.WriteDone()
}

// parseStreamingChunks reads and accumulates SSE chunks into a CompletionResult.
func parseStreamingChunks(body interface{ Read([]byte) (int, error) }, sw *StreamWriter) *domain.CompletionResult {
	acc := domain.NewStreamingAccumulator()
	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk integrations.OpenRouterResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		acc.SetResponseID(chunk.ID)

		if len(chunk.Choices) > 0 {
			processStreamingChoice(chunk.Choices[0], acc, sw)
		}
	}

	return acc.ToResult()
}

// processStreamingChoice processes a single choice from a streaming chunk.
func processStreamingChoice(choice integrations.OpenRouterChoice, acc *domain.StreamingAccumulator, sw *StreamWriter) {
	// Forward content to client and accumulate
	if choice.Delta.Content != "" {
		sw.WriteContentChunk(choice.Delta.Content)
		acc.AppendContent(choice.Delta.Content)
	}

	// Accumulate tool calls
	for _, tc := range choice.Delta.ToolCalls {
		acc.AppendToolCallDelta(tc)
	}

	acc.SetFinishReason(choice.FinishReason)
}

// handleCompletionResult processes a completed AI response.
// Decision: Tool execution required vs regular message
// This is the key decision point - if the model requested tool calls,
// we execute them and signal the client to continue. Otherwise,
// we simply save the message and update the preview.
func (h *Handlers) handleCompletionResult(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	ctx := r.Context()

	if result.RequiresToolExecution() {
		h.handleToolCallsStreaming(r, sw, svc, chatID, model, result)
	} else if result.HasContent() {
		// Save regular message
		svc.SaveCompletionResult(ctx, chatID, model, result)
		svc.UpdateChatPreview(ctx, chatID, result)
	}
}

// handleToolCallsStreaming executes tool calls during a streaming response.
func (h *Handlers) handleToolCallsStreaming(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	ctx := r.Context()

	// Save the assistant message with tool calls
	msg, err := svc.SaveCompletionResult(ctx, chatID, model, result)
	if err != nil {
		sw.WriteError(err)
		return
	}

	svc.UpdateChatPreview(ctx, chatID, result)

	// Execute each tool call
	messageID := ""
	if msg != nil {
		messageID = msg.ID
	}

	for _, tc := range result.ToolCalls {
		sw.WriteToolCallStart(tc)

		results, _ := svc.ExecuteToolCalls(ctx, chatID, messageID, []domain.ToolCall{tc})

		if len(results) > 0 {
			sw.WriteToolCallResult(results[0])
		}
	}

	// Signal that tools were executed and continuation is needed
	sw.WriteToolCallsComplete()
}

// handleNonStreamingResponse processes a non-streaming AI response.
func (h *Handlers) handleNonStreamingResponse(w http.ResponseWriter, r *http.Request, body interface{ Read([]byte) (int, error) }, chatID, model string, orClient *integrations.OpenRouterClient, svc *services.CompletionService) {
	orResp, err := orClient.ParseNonStreamingResponse(body)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orResp.Choices) == 0 {
		h.JSONError(w, "No response from model", http.StatusInternalServerError)
		return
	}

	// Convert to domain type
	result := convertToCompletionResult(orResp)

	if result.RequiresToolExecution() {
		h.handleToolCallsNonStreaming(w, r, svc, chatID, model, result)
	} else {
		h.handleRegularMessageNonStreaming(w, r, svc, chatID, model, result)
	}
}

// convertToCompletionResult converts an OpenRouter response to domain type.
func convertToCompletionResult(resp *integrations.OpenRouterResponse) *domain.CompletionResult {
	choice := resp.Choices[0]
	return &domain.CompletionResult{
		Content:      choice.Message.Content,
		TokenCount:   resp.Usage.CompletionTokens,
		FinishReason: choice.FinishReason,
		ToolCalls:    choice.Message.ToolCalls,
		ResponseID:   resp.ID,
	}
}

// handleToolCallsNonStreaming handles tool execution for non-streaming responses.
func (h *Handlers) handleToolCallsNonStreaming(w http.ResponseWriter, r *http.Request, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	msg, err := svc.SaveCompletionResult(r.Context(), chatID, model, result)
	if err != nil {
		h.JSONError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	messageID := ""
	if msg != nil {
		messageID = msg.ID
	}

	// Execute all tool calls
	toolResults, _ := svc.ExecuteToolCalls(r.Context(), chatID, messageID, result.ToolCalls)

	// Convert to response format
	var resultsMap []map[string]interface{}
	for _, tr := range toolResults {
		m := map[string]interface{}{
			"tool_id":   tr.ToolCallID,
			"tool_name": tr.ToolName,
			"status":    tr.Status,
		}
		if tr.Error != "" {
			m["error"] = tr.Error
		} else {
			m["result"] = tr.Result
		}
		resultsMap = append(resultsMap, m)
	}

	h.JSONResponse(w, map[string]interface{}{
		"message":        msg,
		"tool_results":   resultsMap,
		"needs_followup": true,
	}, http.StatusOK)
}

// handleRegularMessageNonStreaming handles a regular (non-tool) completion.
func (h *Handlers) handleRegularMessageNonStreaming(w http.ResponseWriter, r *http.Request, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	msg, err := svc.SaveCompletionResult(r.Context(), chatID, model, result)
	if err != nil {
		h.JSONError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	svc.UpdateChatPreview(r.Context(), chatID, result)

	h.JSONResponse(w, msg, http.StatusOK)
}
