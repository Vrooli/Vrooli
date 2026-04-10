package handlers

import (
	"fmt"
	"log"
	"net/http"

	"agent-inbox/domain"
	"agent-inbox/services"
)

// handleCompletionResultFull processes a completed AI response and returns full execution info.
// This includes async operation tracking for tools that run in the background.
func (h *Handlers) handleCompletionResultFull(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult) ToolExecutionStreamResult {
	ctx := r.Context()

	// Get the active leaf (the user message that triggered this completion)
	parentMessageID, _ := h.Repo.GetActiveLeaf(ctx, chatID)

	if result.RequiresToolExecution() {
		return h.handleToolCallsStreamingFull(r, sw, svc, chatID, model, result, parentMessageID)
	} else if result.HasResponse() {
		msg, _ := svc.SaveCompletionResult(ctx, chatID, model, result, parentMessageID)
		_ = svc.UpdateChatPreview(ctx, chatID, result)
		if msg != nil {
			return ToolExecutionStreamResult{MessageID: msg.ID}
		}
	}
	return ToolExecutionStreamResult{}
}

// handleToolCallsStreamingFull executes tool calls and returns full result including async info.
// parentMessageID is the user message that triggered this completion (for branching support).
//
// TEMPORAL FLOW NOTE: Tool calls are executed sequentially to maintain
// deterministic ordering. Errors are reported via SSE events but do not
// stop subsequent tool execution - this allows partial success scenarios.
func (h *Handlers) handleToolCallsStreamingFull(r *http.Request, sw *StreamWriter, svc *services.CompletionService, chatID, model string, result *domain.CompletionResult, parentMessageID string) ToolExecutionStreamResult {
	ctx := r.Context()
	res := ToolExecutionStreamResult{}

	// Save the assistant message with tool calls (parented to the user message)
	msg, err := svc.SaveCompletionResult(ctx, chatID, model, result, parentMessageID)
	if err != nil {
		sw.WriteError(err)
		return res
	}

	_ = svc.UpdateChatPreview(ctx, chatID, result)

	messageID := ""
	assistantMessageID := ""
	if msg != nil {
		messageID = msg.ID
		assistantMessageID = msg.ID
	}
	res.MessageID = messageID

	// Fetch active template tool IDs for template deactivation detection
	activeTemplateToolIDs, _ := h.Repo.GetActiveTemplateToolIDs(ctx, chatID)
	templateDeactivated := false

	var toolErrors []error
	var allAsyncOps []domain.AsyncOperationInfo

	for _, tc := range result.ToolCalls {
		sw.WriteToolCallStart(tc)

		outcome, err := svc.ExecuteToolCalls(ctx, chatID, messageID, []domain.ToolCall{tc}, assistantMessageID)
		if err != nil {
			toolErrors = append(toolErrors, err)
			log.Printf("tool call %s failed: %v", tc.Function.Name, err)
		}

		if outcome != nil {
			if outcome.HasPendingApprovals {
				res.HasPendingApprovals = true
				for _, pending := range outcome.PendingApprovals {
					sw.WriteToolCallPendingApproval(pending)
				}
			}

			if outcome.HasAsyncOperations {
				allAsyncOps = append(allAsyncOps, outcome.AsyncOperations...)
			}

			if len(outcome.Results) > 0 {
				toolResult := outcome.Results[0]

				// Check if this tool matches an active template's suggested tool
				if !templateDeactivated && len(activeTemplateToolIDs) > 0 {
					templateDeactivated = checkAndDeactivateTemplate(
						ctx, h.Repo, chatID, tc.Function.Name,
						activeTemplateToolIDs, &toolResult,
					)
				}

				sw.WriteToolCallResult(toolResult)
			}
		}
	}

	if len(allAsyncOps) > 0 {
		res.HasAsyncOperations = true
		res.AsyncOperations = allAsyncOps
	}

	if len(toolErrors) > 0 {
		sw.WriteWarning(domain.ErrCodeToolExecutionFailed, fmt.Sprintf("%d tool(s) encountered errors", len(toolErrors)))
	}

	if res.HasPendingApprovals {
		sw.WriteAwaitingApprovals()
	} else {
		sw.WriteToolCallsComplete()
	}

	return res
}
