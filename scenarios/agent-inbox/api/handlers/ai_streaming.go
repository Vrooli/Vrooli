package handlers

import (
	"context"
	"log"
	"net/http"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/services"
)

// handleStreamingResponse processes a streaming AI response with auto-continue.
//
// # Auto-Continue Loop
//
// After tool calls complete, the handler automatically makes follow-up requests
// so the AI can respond to tool results. This continues until:
//   - AI responds without tool calls (finish_reason="stop")
//   - Pending approvals are waiting (awaiting user action)
//   - Async operations are running (will resume after completion)
//   - Maximum iteration limit is reached (safety valve)
//
// # Async Tool Handling
//
// For async tools (long-running operations), the loop:
// 1. Sends `async_waiting` event to client with operation info
// 2. Registers a completion callback channel with AsyncTracker
// 3. Waits for all async operations to complete
// 4. Sends `async_completed` events as operations finish
// 5. Continues the loop to let AI respond to results
//
// # SSE Protocol
//
// The response uses Server-Sent Events (SSE) format:
//   - Each event is: "data: {json}\n\n"
//   - Final event is: "data: [DONE]\n\n"
//   - Events are forwarded to client as they arrive
//
// See docs/SEAMS.md for complete streaming protocol specification.
//
// # Error Handling
//
// Errors during follow-up requests are sent via SSE error events rather than
// failing the entire response. This allows partial success scenarios.
func (h *Handlers) handleStreamingResponse(w http.ResponseWriter, r *http.Request, body interface{ Read([]byte) (int, error) }, chatID, model string, svc *services.CompletionService) {
	// Setup SSE response
	sw := SetupSSEResponse(w, r)
	if sw == nil {
		h.JSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	iteration := 0

	// Current response body to process
	currentBody := body

	for iteration < services.MaxAutoContinueIterations {
		iteration++
		log.Printf("[DEBUG] handleStreamingResponse iteration %d", iteration)

		// Parse and accumulate streaming chunks
		result := parseStreamingChunks(currentBody, sw)

		// Handle the completion result (returns full info including async operations)
		execResult := h.handleCompletionResultFull(r, sw, svc, chatID, model, result)

		// Fetch and save generation stats asynchronously (non-blocking)
		h.fetchAndSaveGenerationStats(chatID, execResult.MessageID, model, result.ResponseID, result.Usage)

		// If there are pending approvals, stop and wait for user action
		if execResult.HasPendingApprovals {
			log.Printf("[DEBUG] Stopping auto-continue: pending approvals")
			break
		}

		// If there are async operations, wait for them to complete
		if execResult.HasAsyncOperations {
			log.Printf("[DEBUG] Async operations detected, waiting for completion (count=%d)", len(execResult.AsyncOperations))

			// Notify client that we're waiting for async operations
			sw.WriteAsyncWaiting(execResult.AsyncOperations)

			// Wait for async operations to complete
			completed := h.waitForAsyncCompletion(r.Context(), chatID, sw, svc, execResult.AsyncOperations)
			if !completed {
				log.Printf("[DEBUG] Async wait interrupted or timed out")
				break
			}

			// After async completion, continue the loop to let AI respond to results
			log.Printf("[DEBUG] Async operations completed, continuing loop")
			// Fall through to make follow-up request with async results
		}

		// If no tool calls were made, we're done
		if !result.RequiresToolExecution() {
			log.Printf("[DEBUG] Stopping auto-continue: no tool calls (finish_reason=%s)", result.FinishReason)
			break
		}

		// Tool calls were made and executed - make a follow-up request
		// to let the AI respond to the tool results
		log.Printf("[DEBUG] Auto-continuing after tool execution (iteration %d)", iteration)

		followUpBody, err := h.makeFollowUpRequest(r.Context(), svc, chatID)
		if err != nil {
			sw.WriteError(err)
			break
		}

		// Update currentBody for the next iteration
		currentBody = followUpBody
		// Note: resp.Body will be closed when we exit the loop or on next iteration
	}

	if iteration >= services.MaxAutoContinueIterations {
		log.Printf("[WARN] Auto-continue reached max iterations (%d)", services.MaxAutoContinueIterations)
	}

	sw.WriteDone()
}

// makeFollowUpRequest creates a new streaming completion request for auto-continue.
// Returns the response body for the next iteration of the streaming loop.
func (h *Handlers) makeFollowUpRequest(ctx context.Context, svc *services.CompletionService, chatID string) (interface{ Read([]byte) (int, error) }, error) {
	orClient, err := integrations.NewOpenRouterClient()
	if err != nil {
		log.Printf("[ERROR] Failed to create OpenRouter client for auto-continue: %v", err)
		return nil, err
	}

	// Prepare a new completion request (will include tool results from DB)
	// Note: forcedTool is not passed on follow-up - we only force on the initial request
	prepReq, err := svc.PrepareCompletionRequest(ctx, chatID, true, "")
	if err != nil {
		log.Printf("[ERROR] Failed to prepare follow-up request: %v", err)
		return nil, err
	}

	orReq := &integrations.OpenRouterRequest{
		Model:      prepReq.Model,
		Messages:   prepReq.Messages,
		Stream:     true,
		Tools:      prepReq.Tools,
		ToolChoice: nil, // Auto for follow-up
		Plugins:    prepReq.Plugins,
		Modalities: prepReq.Modalities,
	}

	resp, err := orClient.CreateCompletion(ctx, orReq)
	if err != nil {
		log.Printf("[ERROR] Follow-up completion failed: %v", err)
		return nil, err
	}

	return resp.Body, nil
}

// waitForAsyncCompletion waits for all async operations to complete.
// Returns true if all operations completed successfully, false if interrupted or timed out.
// Sends progress updates to the client via SSE as operations complete.
func (h *Handlers) waitForAsyncCompletion(ctx context.Context, chatID string, sw *StreamWriter, svc *services.CompletionService, asyncOps []domain.AsyncOperationInfo) bool {
	if len(asyncOps) == 0 {
		return true
	}

	// Register completion callback to receive async results
	completionCh := h.AsyncTracker.RegisterCompletionCallback(chatID)
	defer h.AsyncTracker.UnregisterCompletionCallback(chatID)

	// Track which operations we're waiting for
	pendingOps := make(map[string]bool)
	for _, op := range asyncOps {
		pendingOps[op.ToolCallID] = true
	}

	log.Printf("[DEBUG] Waiting for %d async operations", len(pendingOps))

	// Wait for all operations to complete
	for len(pendingOps) > 0 {
		select {
		case <-ctx.Done():
			log.Printf("[DEBUG] Context cancelled while waiting for async operations")
			return false

		case event, ok := <-completionCh:
			if !ok {
				log.Printf("[DEBUG] Completion channel closed")
				return false
			}

			// Only process events for operations we're tracking
			if !pendingOps[event.ToolCallID] {
				log.Printf("[DEBUG] Received completion for untracked operation: %s", event.ToolCallID)
				continue
			}

			// Mark this operation as complete
			delete(pendingOps, event.ToolCallID)

			// Notify client of completion
			sw.WriteAsyncCompleted(event.ToolCallID, event.ToolName, event.Status, event.Result, event.Error)

			// Save the async result as a tool response message for the AI conversation
			if err := h.saveAsyncResultAsToolResponse(ctx, chatID, event, svc); err != nil {
				log.Printf("[WARN] Failed to save async result as tool response: %v", err)
			}

			log.Printf("[DEBUG] Async operation completed: %s (status=%s, remaining=%d)",
				event.ToolCallID, event.Status, len(pendingOps))
		}
	}

	log.Printf("[DEBUG] All async operations completed")
	return true
}

// saveAsyncResultAsToolResponse saves the async operation result as a tool response message.
// This ensures the AI sees the result in subsequent completions.
func (h *Handlers) saveAsyncResultAsToolResponse(ctx context.Context, chatID string, event services.AsyncCompletionEvent, svc *services.CompletionService) error {
	// Build the tool result to save
	var resultContent interface{}
	if event.Error != "" {
		resultContent = map[string]interface{}{
			"status": event.Status,
			"error":  event.Error,
		}
	} else {
		resultContent = map[string]interface{}{
			"status": event.Status,
			"result": event.Result,
		}
	}

	// Save as a tool response message
	toolResult := domain.ToolExecutionResult{
		ToolCallID: event.ToolCallID,
		ToolName:   event.ToolName,
		Status:     event.Status,
		Result:     resultContent,
	}
	if event.Error != "" {
		toolResult.Error = event.Error
	}

	// Get the assistant message that contains the tool call to use as parent
	parentMessageID, err := h.Repo.GetActiveLeaf(ctx, chatID)
	if err != nil {
		log.Printf("[WARN] Failed to get active leaf for async result: %v", err)
	}

	return svc.SaveToolResult(ctx, chatID, toolResult, parentMessageID)
}
