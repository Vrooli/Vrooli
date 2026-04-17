package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/services"
	"log"
	"net/http"
)

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
		content, images = extractMultimodalContent(c)
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

// extractMultimodalContent extracts text and images from multimodal content parts.
func extractMultimodalContent(parts []interface{}) (string, []string) {
	var content string
	var images []string

	for _, part := range parts {
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
			if imgURL, ok := partMap["image_url"].(map[string]interface{}); ok {
				if url, ok := imgURL["url"].(string); ok && url != "" {
					log.Printf("[DEBUG] Found image in content: %s...", url[:min(50, len(url))])
					images = append(images, url)
				}
			}
		}
	}

	return content, images
}

// handleToolCallsNonStreaming handles tool execution for non-streaming responses.
//
// TEMPORAL FLOW NOTE: Tool calls are executed sequentially. Errors are logged
// and individual tool results reflect their status. The overall response is
// still returned to allow partial success handling.
func (h *Handlers) handleToolCallsNonStreaming(w http.ResponseWriter, r *http.Request, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) {
	ctx := r.Context()

	// Get the active leaf (the user message that triggered this completion)
	parentMessageID, _ := h.Repo.GetActiveLeaf(ctx, chatID)

	msg, err := svc.SaveCompletionResult(ctx, chatID, model, result, parentMessageID)
	if err != nil {
		h.JSONError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	messageID := ""
	assistantMessageID := ""
	if msg != nil {
		messageID = msg.ID
		assistantMessageID = msg.ID
	}

	// Fetch active template tool IDs for template deactivation detection
	activeTemplateToolIDs, _ := h.Repo.GetActiveTemplateToolIDs(ctx, chatID)
	templateDeactivated := false

	// Execute all tool calls
	outcome, toolErr := svc.ExecuteToolCalls(ctx, chatID, messageID, result.ToolCalls, assistantMessageID)
	if toolErr != nil {
		log.Printf("tool execution error for chat %s: %v", chatID, toolErr)
	}

	// Convert to response format
	var resultsMap []map[string]interface{}
	var pendingApprovalsMap []map[string]interface{}

	if outcome != nil {
		for i, tr := range outcome.Results {
			// Check if this tool matches an active template's suggested tool
			if !templateDeactivated && len(activeTemplateToolIDs) > 0 {
				templateDeactivated = checkAndDeactivateTemplate(
					ctx, h.Repo, chatID, tr.ToolName,
					activeTemplateToolIDs, &outcome.Results[i],
				)
			}

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
			if outcome.Results[i].DeactivateTemplate {
				m["deactivate_template"] = true
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
		"needs_followup":     !outcome.HasPendingApprovals,
		"pending_approvals":  pendingApprovalsMap,
		"awaiting_approvals": outcome.HasPendingApprovals,
	}

	if toolErr != nil {
		response["tool_errors"] = toolErr.Error()
	}

	// Fetch and save generation stats asynchronously
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

	_ = svc.UpdateChatPreview(r.Context(), chatID, result)

	// Fetch and save generation stats asynchronously
	messageID := ""
	if msg != nil {
		messageID = msg.ID
	}
	h.fetchAndSaveGenerationStats(chatID, messageID, model, result.ResponseID, result.Usage)

	h.JSONResponse(w, msg, http.StatusOK)
}
