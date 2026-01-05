package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/services"
)

// fetchAndSaveGenerationStats asynchronously fetches usage/cost data from OpenRouter
// and saves it to the database. This is called after a completion request finishes.
// It runs in a background goroutine to not block the response to the client.
//
// The function uses a hybrid approach:
// 1. Retry fetching generation stats with exponential backoff (OpenRouter needs time to index)
// 2. If stats are available, save record with accurate cost from OpenRouter
// 3. If all retries fail, save fallback record with token counts from response (cost = 0)
//
// This ensures we always capture at least token usage, even when cost data is unavailable.
func (h *Handlers) fetchAndSaveGenerationStats(chatID, messageID, model, generationID string, fallbackUsage *domain.Usage) {
	// Skip if we have neither generation ID nor fallback usage data
	if generationID == "" && fallbackUsage == nil {
		return
	}

	go func() {
		// Use a fresh context with enough time for retries
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var stats *integrations.GenerationStats
		var fetchErr error

		// Try to fetch generation stats with exponential backoff
		if generationID != "" {
			orClient, err := integrations.NewOpenRouterClient()
			if err != nil {
				log.Printf("[WARN] Failed to create OpenRouter client for usage stats: %v", err)
			} else {
				// Exponential backoff: wait before each attempt since OpenRouter needs time to index
				// Delays: 2s, 4s, 8s (total wait ~14s before giving up)
				delays := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

				for attempt, delay := range delays {
					time.Sleep(delay)

					stats, fetchErr = orClient.FetchGenerationStats(ctx, generationID)
					if fetchErr == nil {
						break
					}

					log.Printf("[DEBUG] Generation stats attempt %d/%d for %s: %v",
						attempt+1, len(delays), generationID, fetchErr)
				}
			}
		}

		// If we got stats, use them (preferred - has accurate cost)
		if stats != nil {
			usageRecord := integrations.CreateUsageRecordFromStats(chatID, messageID, stats)
			if usageRecord != nil {
				if err := h.Repo.SaveUsageRecord(ctx, usageRecord); err != nil {
					log.Printf("[WARN] Failed to save usage record: %v", err)
					return
				}
				log.Printf("[INFO] Saved usage stats: model=%s, tokens=%d, cost=$%.4f",
					stats.Model, usageRecord.TotalTokens, stats.TotalCost)
			}
			return
		}

		// Fallback: use usage data from response if available (tokens only, no cost)
		if fallbackUsage != nil && messageID != "" {
			fallbackRecord := &domain.UsageRecord{
				ChatID:           chatID,
				MessageID:        messageID,
				Model:            model,
				PromptTokens:     fallbackUsage.PromptTokens,
				CompletionTokens: fallbackUsage.CompletionTokens,
				TotalTokens:      fallbackUsage.TotalTokens,
				TotalCost:        0, // Cost unknown - generation stats unavailable
			}
			if err := h.Repo.SaveUsageRecord(ctx, fallbackRecord); err != nil {
				log.Printf("[WARN] Failed to save fallback usage record: %v", err)
				return
			}
			log.Printf("[INFO] Saved fallback usage (no cost data): model=%s, tokens=%d",
				model, fallbackRecord.TotalTokens)
		}

		// Log final failure if we couldn't get generation stats
		if fetchErr != nil {
			log.Printf("[WARN] All attempts to fetch generation stats failed for %s: %v",
				generationID, fetchErr)
		}
	}()
}

// ListModels returns available AI models from OpenRouter.
// Uses cached model registry to avoid repeated CLI calls.
func (h *Handlers) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.ModelRegistry.GetModels(r.Context())
	if err != nil {
		h.JSONError(w, "Failed to fetch models", http.StatusInternalServerError)
		return
	}
	h.JSONResponse(w, models, http.StatusOK)
}

// ListTools returns available tools for AI.
// Uses the dynamic ToolRegistry to fetch tools from all configured scenarios.
func (h *Handlers) ListTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.ToolRegistry.GetEffectiveTools(r.Context(), "")
	if err != nil {
		log.Printf("warning: failed to get tools from registry: %v", err)
		// Return empty array on error for graceful degradation
		h.JSONResponse(w, []interface{}{}, http.StatusOK)
		return
	}

	// Convert to OpenAI format for backward compatibility
	openAITools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		if tool.Enabled {
			openAITools[i] = domain.ToOpenAIFunction(tool.Tool)
		}
	}

	h.JSONResponse(w, openAITools, http.StatusOK)
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
	chat, err := h.Repo.UpdateChat(r.Context(), chatID, &name, nil, nil)
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

	// Parse optional force_tool query param (format: "scenario:tool_name")
	forcedTool := r.URL.Query().Get("force_tool")

	// Prepare completion request (validates chat exists and has messages)
	svc := h.NewCompletionService()
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

// handleStreamingResponse processes a streaming AI response with auto-continue.
// After tool calls complete, it automatically makes follow-up requests so the AI
// can respond to tool results. This continues until the AI responds without tool calls
// or a maximum iteration limit is reached.
//
// The response arrives as Server-Sent Events (SSE) that must be parsed,
// accumulated, and forwarded to the client.
func (h *Handlers) handleStreamingResponse(w http.ResponseWriter, r *http.Request, body interface{ Read([]byte) (int, error) }, chatID, model string, svc *services.CompletionService) {
	// Setup SSE response
	sw := SetupSSEResponse(w, r)
	if sw == nil {
		h.JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Maximum iterations to prevent infinite loops (tool call -> response -> tool call -> ...)
	const maxIterations = 10
	iteration := 0

	// Current response body to process
	currentBody := body

	for iteration < maxIterations {
		iteration++
		log.Printf("[DEBUG] handleStreamingResponse iteration %d", iteration)

		// Parse and accumulate streaming chunks
		result := parseStreamingChunks(currentBody, sw)

		// Handle the completion result
		messageID, hasPendingApprovals := h.handleCompletionResultWithStatus(r, sw, svc, chatID, model, result)

		// Fetch and save generation stats asynchronously (non-blocking)
		h.fetchAndSaveGenerationStats(chatID, messageID, model, result.ResponseID, result.Usage)

		// If there are pending approvals, stop and wait for user action
		if hasPendingApprovals {
			log.Printf("[DEBUG] Stopping auto-continue: pending approvals")
			break
		}

		// If no tool calls were made, we're done
		if !result.RequiresToolExecution() {
			log.Printf("[DEBUG] Stopping auto-continue: no tool calls (finish_reason=%s)", result.FinishReason)
			break
		}

		// Tool calls were made and executed - make a follow-up request
		// to let the AI respond to the tool results
		log.Printf("[DEBUG] Auto-continuing after tool execution (iteration %d)", iteration)

		// Create a new OpenRouter client for the follow-up request
		orClient, err := integrations.NewOpenRouterClient()
		if err != nil {
			log.Printf("[ERROR] Failed to create OpenRouter client for auto-continue: %v", err)
			sw.WriteError(err)
			break
		}

		// Prepare a new completion request (will include tool results from DB)
		// Note: forcedTool is not passed on follow-up - we only force on the initial request
		prepReq, err := svc.PrepareCompletionRequest(r.Context(), chatID, true, "")
		if err != nil {
			log.Printf("[ERROR] Failed to prepare follow-up request: %v", err)
			sw.WriteError(err)
			break
		}

		// Build and execute the follow-up request
		orReq := &integrations.OpenRouterRequest{
			Model:      prepReq.Model,
			Messages:   prepReq.Messages,
			Stream:     true,
			Tools:      prepReq.Tools,
			ToolChoice: nil, // Auto for follow-up
			Plugins:    prepReq.Plugins,
			Modalities: prepReq.Modalities,
		}

		resp, err := orClient.CreateCompletion(r.Context(), orReq)
		if err != nil {
			log.Printf("[ERROR] Follow-up completion failed: %v", err)
			sw.WriteError(err)
			break
		}

		// Update currentBody for the next iteration
		currentBody = resp.Body
		// Note: resp.Body will be closed when we exit the loop or on next iteration
	}

	if iteration >= maxIterations {
		log.Printf("[WARN] Auto-continue reached max iterations (%d)", maxIterations)
	}

	sw.WriteDone()
}

// parseStreamingChunks reads and accumulates SSE chunks into a CompletionResult.
func parseStreamingChunks(body interface{ Read([]byte) (int, error) }, sw *StreamWriter) *domain.CompletionResult {
	acc := domain.NewStreamingAccumulator()
	scanner := bufio.NewScanner(body)
	// Increase buffer size to handle large SSE chunks (e.g., generated images as base64)
	// Default is 64KB, we increase to 16MB for large image data URLs
	const maxScanTokenSize = 16 * 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineCount := 0
	dataLineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		// Log first 5 lines and any line containing "data"
		if lineCount <= 5 {
			log.Printf("[DEBUG] SSE line %d: %s", lineCount, line[:min(len(line), 100)])
		}
		if strings.Contains(strings.ToLower(line), "data") {
			dataLineCount++
			log.Printf("[DEBUG] SSE data line %d: %s", dataLineCount, line[:min(len(line), 200)])
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// Log raw data for first few chunks to debug response format
		if dataLineCount <= 3 {
			log.Printf("[DEBUG] SSE raw chunk %d: %s", dataLineCount, data[:min(500, len(data))])
		}

		var chunk integrations.OpenRouterResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("[DEBUG] SSE parse error: %v, data: %s", err, data[:min(100, len(data))])
			continue
		}

		log.Printf("[DEBUG] SSE chunk: id=%s, choices=%d", chunk.ID, len(chunk.Choices))
		if len(chunk.Choices) > 0 {
			// Log detailed info about the first choice to see content structure
			choice := chunk.Choices[0]
			contentBytes, _ := json.Marshal(choice.Delta.Content)
			log.Printf("[DEBUG] SSE choice[0]: finish_reason=%s, delta.content=%s, delta.images=%d",
				choice.FinishReason, string(contentBytes)[:min(200, len(contentBytes))], len(choice.Delta.Images))
		}
		acc.SetResponseID(chunk.ID)

		// Capture usage data if present (typically in final chunk)
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			acc.SetUsage(chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens, chunk.Usage.TotalTokens)
		}

		if len(chunk.Choices) > 0 {
			processStreamingChoice(chunk.Choices[0], acc, sw)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[DEBUG] SSE scanner error: %v", err)
	}

	result := acc.ToResult()
	log.Printf("[DEBUG] SSE parsing complete: %d total lines, %d data lines, content length=%d, tool_calls=%d, images=%d",
		lineCount, dataLineCount, len(result.Content), len(result.ToolCalls), len(result.Images))
	return result
}

// processStreamingChoice processes a single choice from a streaming chunk.
func processStreamingChoice(choice integrations.OpenRouterChoice, acc *domain.StreamingAccumulator, sw *StreamWriter) {
	// Forward content to client and accumulate
	// Content can be either a string (normal text) or array of content parts (multimodal with images)
	log.Printf("[DEBUG] processStreamingChoice: delta.Content type=%T", choice.Delta.Content)

	switch c := choice.Delta.Content.(type) {
	case string:
		if c != "" {
			sw.WriteContentChunk(c)
			acc.AppendContent(c)
		}
	case []interface{}:
		// Multimodal streaming - extract text and images from content parts
		log.Printf("[DEBUG] Streaming multimodal content with %d parts", len(c))
		for _, part := range c {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := partMap["type"].(string)

			switch partType {
			case "text":
				if text, ok := partMap["text"].(string); ok && text != "" {
					sw.WriteContentChunk(text)
					acc.AppendContent(text)
				}
			case "image_url":
				// Extract image URL from image_url object
				if imgURL, ok := partMap["image_url"].(map[string]interface{}); ok {
					if url, ok := imgURL["url"].(string); ok && url != "" {
						log.Printf("[DEBUG] Streaming: found image in content part")
						acc.AppendImage(url)
						sw.WriteImageGenerated(url)
					}
				}
			}
		}
	}

	// Handle generated images in legacy Images field
	for _, img := range choice.Delta.Images {
		if img.ImageURL != nil && img.ImageURL.URL != "" {
			log.Printf("[DEBUG] Received generated image in delta.Images")
			acc.AppendImage(img.ImageURL.URL)
			sw.WriteImageGenerated(img.ImageURL.URL)
		}
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
// Returns the message ID of the saved message (empty if none saved).
func (h *Handlers) handleCompletionResult(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) string {
	messageID, _ := h.handleCompletionResultWithStatus(r, sw, svc, chatID, model, result)
	return messageID
}

// handleCompletionResultWithStatus is like handleCompletionResult but also returns
// whether there are pending approvals. This is used by the auto-continue loop.
func (h *Handlers) handleCompletionResultWithStatus(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) (string, bool) {
	ctx := r.Context()

	// Get the active leaf (the user message that triggered this completion)
	// This becomes the parent of the assistant message for branching support
	parentMessageID, _ := h.Repo.GetActiveLeaf(ctx, chatID)

	if result.RequiresToolExecution() {
		return h.handleToolCallsStreamingWithStatus(r, sw, svc, chatID, model, result, parentMessageID)
	} else if result.HasResponse() {
		// Save regular message (text and/or images)
		msg, _ := svc.SaveCompletionResult(ctx, chatID, model, result, parentMessageID)
		svc.UpdateChatPreview(ctx, chatID, result)
		if msg != nil {
			return msg.ID, false
		}
	}
	return "", false
}

// handleToolCallsStreaming executes tool calls during a streaming response.
// parentMessageID is the user message that triggered this completion (for branching support).
// Returns the message ID of the saved assistant message (empty if save failed).
//
// TEMPORAL FLOW NOTE: Tool calls are executed sequentially to maintain
// deterministic ordering. Errors are reported via SSE events but do not
// stop subsequent tool execution - this allows partial success scenarios.
func (h *Handlers) handleToolCallsStreaming(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult, parentMessageID string) string {
	messageID, _ := h.handleToolCallsStreamingWithStatus(r, sw, svc, chatID, model, result, parentMessageID)
	return messageID
}

// handleToolCallsStreamingWithStatus is like handleToolCallsStreaming but also returns
// whether there are pending approvals. This is used by the auto-continue loop.
func (h *Handlers) handleToolCallsStreamingWithStatus(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult, parentMessageID string) (string, bool) {
	ctx := r.Context()

	// Save the assistant message with tool calls (parented to the user message)
	msg, err := svc.SaveCompletionResult(ctx, chatID, model, result, parentMessageID)
	if err != nil {
		sw.WriteError(err)
		return "", false
	}

	svc.UpdateChatPreview(ctx, chatID, result)

	// Execute each tool call
	messageID := ""
	assistantMessageID := ""
	if msg != nil {
		messageID = msg.ID
		assistantMessageID = msg.ID // Tool responses are parented to the assistant message
	}

	var toolErrors []error
	var hasPendingApprovals bool

	for _, tc := range result.ToolCalls {
		sw.WriteToolCallStart(tc)

		outcome, err := svc.ExecuteToolCalls(ctx, chatID, messageID, []domain.ToolCall{tc}, assistantMessageID)
		if err != nil {
			toolErrors = append(toolErrors, err)
			log.Printf("tool call %s failed: %v", tc.Function.Name, err)
		}

		if outcome != nil {
			if outcome.HasPendingApprovals {
				hasPendingApprovals = true
				// Write pending approval event for each pending tool
				for _, pending := range outcome.PendingApprovals {
					sw.WriteToolCallPendingApproval(pending)
				}
			}

			if len(outcome.Results) > 0 {
				sw.WriteToolCallResult(outcome.Results[0])
			}
		}
	}

	// Report aggregated warning if any tools failed
	if len(toolErrors) > 0 {
		sw.WriteWarning(domain.ErrCodeToolExecutionFailed, fmt.Sprintf("%d tool(s) encountered errors", len(toolErrors)))
	}

	// Signal completion status
	if hasPendingApprovals {
		sw.WriteAwaitingApprovals()
	} else {
		sw.WriteToolCallsComplete()
	}

	return messageID, hasPendingApprovals
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

	// Extract content and images from message
	// Content can be either a string or an array of content parts (multimodal)
	var content string
	var images []string

	log.Printf("[DEBUG] convertToCompletionResult: Content type=%T", choice.Message.Content)

	switch c := choice.Message.Content.(type) {
	case string:
		content = c
	case []interface{}:
		// Multimodal response - extract text and images from content parts
		log.Printf("[DEBUG] Multimodal content with %d parts", len(c))
		for _, part := range c {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := partMap["type"].(string)
			log.Printf("[DEBUG] Content part type: %s", partType)

			switch partType {
			case "text":
				if text, ok := partMap["text"].(string); ok {
					if content != "" {
						content += "\n"
					}
					content += text
				}
			case "image_url":
				// Extract image URL from image_url object
				if imgURL, ok := partMap["image_url"].(map[string]interface{}); ok {
					if url, ok := imgURL["url"].(string); ok && url != "" {
						log.Printf("[DEBUG] Found image in content: %s...", url[:min(50, len(url))])
						images = append(images, url)
					}
				}
			}
		}
	default:
		log.Printf("[DEBUG] Unexpected content type: %T", choice.Message.Content)
	}

	result := &domain.CompletionResult{
		Content:      content,
		TokenCount:   resp.Usage.CompletionTokens,
		FinishReason: choice.FinishReason,
		ToolCalls:    choice.Message.ToolCalls,
		ResponseID:   resp.ID,
		Images:       images,
	}

	// Also extract from legacy Images field if present
	for _, img := range choice.Message.Images {
		if img.ImageURL != nil && img.ImageURL.URL != "" {
			result.Images = append(result.Images, img.ImageURL.URL)
		}
	}
	if len(result.Images) > 0 {
		log.Printf("[DEBUG] Total images extracted: %d", len(result.Images))
	}

	// Capture full usage data if available
	if resp.Usage.PromptTokens > 0 || resp.Usage.CompletionTokens > 0 {
		result.Usage = &domain.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return result
}

// handleToolCallsNonStreaming handles tool execution for non-streaming responses.
//
// TEMPORAL FLOW NOTE: Tool calls are executed sequentially. Errors are logged
// and individual tool results reflect their status. The overall response is
// still returned to allow partial success handling.
func (h *Handlers) handleToolCallsNonStreaming(w http.ResponseWriter, r *http.Request, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	// Get the active leaf (the user message that triggered this completion)
	parentMessageID, _ := h.Repo.GetActiveLeaf(r.Context(), chatID)

	msg, err := svc.SaveCompletionResult(r.Context(), chatID, model, result, parentMessageID)
	if err != nil {
		h.JSONError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	messageID := ""
	assistantMessageID := ""
	if msg != nil {
		messageID = msg.ID
		assistantMessageID = msg.ID // Tool responses are parented to the assistant message
	}

	// Execute all tool calls
	outcome, toolErr := svc.ExecuteToolCalls(r.Context(), chatID, messageID, result.ToolCalls, assistantMessageID)
	if toolErr != nil {
		log.Printf("tool execution error for chat %s: %v", chatID, toolErr)
	}

	// Convert to response format
	var resultsMap []map[string]interface{}
	var pendingApprovalsMap []map[string]interface{}

	if outcome != nil {
		for _, tr := range outcome.Results {
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

		for _, pending := range outcome.PendingApprovals {
			pendingApprovalsMap = append(pendingApprovalsMap, map[string]interface{}{
				"id":         pending.ID,
				"tool_name":  pending.ToolName,
				"arguments":  pending.Arguments,
				"status":     pending.Status,
				"started_at": pending.StartedAt,
			})
		}
	}

	response := map[string]interface{}{
		"message":            msg,
		"tool_results":       resultsMap,
		"needs_followup":     !outcome.HasPendingApprovals, // Only needs AI followup if no pending approvals
		"pending_approvals":  pendingApprovalsMap,
		"awaiting_approvals": outcome.HasPendingApprovals,
	}

	// Include error summary in response if any tools failed
	if toolErr != nil {
		response["tool_errors"] = toolErr.Error()
	}

	// Fetch and save generation stats asynchronously
	// Pass model and usage data for fallback if OpenRouter stats are unavailable
	h.fetchAndSaveGenerationStats(chatID, messageID, model, result.ResponseID, result.Usage)

	h.JSONResponse(w, response, http.StatusOK)
}

// handleRegularMessageNonStreaming handles a regular (non-tool) completion.
func (h *Handlers) handleRegularMessageNonStreaming(w http.ResponseWriter, r *http.Request, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	// Get the active leaf (the user message that triggered this completion)
	parentMessageID, _ := h.Repo.GetActiveLeaf(r.Context(), chatID)

	msg, err := svc.SaveCompletionResult(r.Context(), chatID, model, result, parentMessageID)
	if err != nil {
		h.JSONError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	svc.UpdateChatPreview(r.Context(), chatID, result)

	// Fetch and save generation stats asynchronously
	// Pass model and usage data for fallback if OpenRouter stats are unavailable
	messageID := ""
	if msg != nil {
		messageID = msg.ID
	}
	h.fetchAndSaveGenerationStats(chatID, messageID, model, result.ResponseID, result.Usage)

	h.JSONResponse(w, msg, http.StatusOK)
}
