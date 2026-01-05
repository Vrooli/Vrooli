// Package services contains business logic orchestration.
// Services coordinate between handlers, persistence, and integrations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
)

// NewToolExecutionResult creates a ToolExecutionResult from a record and optional error.
// This centralizes the decision of how to populate the result based on success/failure.
func NewToolExecutionResult(toolCallID, toolName string, record *domain.ToolCallRecord, err error) domain.ToolExecutionResult {
	result := domain.ToolExecutionResult{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Status:     record.Status,
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Result = record.Result
	}
	return result
}

// CompletionService orchestrates AI chat completion.
// It handles the decision flow for completing a chat with an AI model,
// including tool execution when requested by the model.
type CompletionService struct {
	repo             *persistence.Repository
	executor         *integrations.ToolExecutor
	toolRegistry     *ToolRegistry
	contextManager   *ContextManager
	messageConverter *MessageConverter
	storage          StorageService
	asyncTracker     *AsyncTrackerService
}

// NewCompletionService creates a new completion service.
func NewCompletionService(repo *persistence.Repository, storage StorageService) *CompletionService {
	modelRegistry := NewModelRegistry()
	return &CompletionService{
		repo:             repo,
		executor:         integrations.NewToolExecutor(),
		toolRegistry:     NewToolRegistry(repo),
		contextManager:   NewContextManager(modelRegistry, config.Default()),
		messageConverter: NewMessageConverter(storage),
		storage:          storage,
	}
}

// NewCompletionServiceWithRegistry creates a completion service with an injected registry.
// This is the constructor for testing.
func NewCompletionServiceWithRegistry(repo *persistence.Repository, registry *ToolRegistry, storage StorageService) *CompletionService {
	modelRegistry := NewModelRegistry()
	return &CompletionService{
		repo:             repo,
		executor:         integrations.NewToolExecutor(),
		toolRegistry:     registry,
		contextManager:   NewContextManager(modelRegistry, config.Default()),
		messageConverter: NewMessageConverter(storage),
		storage:          storage,
	}
}

// SetAsyncTracker sets the async tracker for tracking long-running tool operations.
// This is called after construction to avoid circular dependencies.
func (s *CompletionService) SetAsyncTracker(tracker *AsyncTrackerService) {
	s.asyncTracker = tracker
}

// SaveCompletionResult persists a completion result to the database.
// This handles the decision of whether to save as a regular message or
// as a message with tool calls.
// parentMessageID is used for branching support (ChatGPT-style regeneration).
func (s *CompletionService) SaveCompletionResult(ctx context.Context, chatID, model string, result *domain.CompletionResult, parentMessageID string) (*domain.Message, error) {
	var msg *domain.Message
	var err error

	if result.RequiresToolExecution() {
		msg, err = s.repo.SaveAssistantMessageWithToolCalls(
			ctx, chatID, model, result.Content, result.ToolCalls,
			result.ResponseID, result.FinishReason, result.TokenCount, parentMessageID,
		)
	} else {
		msg, err = s.repo.SaveAssistantMessage(ctx, chatID, model, result.Content, result.TokenCount, parentMessageID)
	}

	if err != nil {
		return msg, err
	}

	// Update active leaf to point to the new message
	if msg != nil {
		s.repo.SetActiveLeaf(ctx, chatID, msg.ID)
	}

	// Save generated images as attachments
	if len(result.Images) > 0 && msg != nil {
		for i, imageDataURL := range result.Images {
			att, saveErr := s.storage.SaveBase64Image(ctx, imageDataURL, fmt.Sprintf("generated_%d", i+1))
			if saveErr != nil {
				log.Printf("warning: failed to save generated image %d: %v", i+1, saveErr)
				continue
			}
			// Create attachment record in database
			if createErr := s.repo.CreateAttachment(ctx, att); createErr != nil {
				log.Printf("warning: failed to create attachment record for generated image: %v", createErr)
				continue
			}
			// Link attachment to message
			if linkErr := s.repo.AttachToMessage(ctx, att.ID, msg.ID); linkErr != nil {
				log.Printf("warning: failed to link generated image to message: %v", linkErr)
			}
		}
		log.Printf("[DEBUG] Saved %d generated images as attachments for message %s", len(result.Images), msg.ID)
	}

	// Note: Usage record saving is now handled asynchronously by the handler
	// using OpenRouter's generation stats API for accurate cost data.

	return msg, nil
}

// ToolExecutionOutcome represents the result of attempting to execute tool calls.
// Some tools may execute immediately, others may require approval.
type ToolExecutionOutcome struct {
	// Results contains execution results for tools that ran immediately.
	Results []domain.ToolExecutionResult
	// PendingApprovals contains tool calls that require user approval.
	PendingApprovals []*domain.ToolCallRecord
	// HasPendingApprovals indicates if any tools are waiting for approval.
	HasPendingApprovals bool
}

// ExecuteToolCalls runs all tool calls from a completion result.
// Returns results for each tool call in order.
// parentMessageID is the assistant message that made the tool calls (for branching support).
//
// APPROVAL FLOW: If a tool requires approval (based on YOLO mode, user config, or metadata),
// it will be saved as pending_approval and not executed. The caller should check
// HasPendingApprovals to determine if the UI needs to show approval prompts.
//
// Error Handling:
//   - Individual tool errors are captured in each ToolExecutionResult
//   - The returned error is non-nil if ANY tool call failed
//   - Callers can inspect individual results for partial success scenarios
func (s *CompletionService) ExecuteToolCalls(ctx context.Context, chatID, messageID string, toolCalls []domain.ToolCall, parentMessageID string) (*ToolExecutionOutcome, error) {
	outcome := &ToolExecutionOutcome{
		Results:          make([]domain.ToolExecutionResult, 0, len(toolCalls)),
		PendingApprovals: make([]*domain.ToolCallRecord, 0),
	}
	var executionErrors []error
	var lastToolMsgID string

	for _, tc := range toolCalls {
		// Check if this tool requires approval
		requiresApproval, _, err := s.toolRegistry.GetToolApprovalRequired(ctx, chatID, tc.Function.Name)
		if err != nil {
			log.Printf("warning: failed to check approval requirement for %s: %v", tc.Function.Name, err)
			// Default to not requiring approval on error
			requiresApproval = false
		}

		if requiresApproval {
			// Create pending approval record instead of executing
			record := s.createPendingApprovalRecord(chatID, messageID, tc)
			if messageID != "" {
				if saveErr := s.repo.SaveToolCallRecord(ctx, messageID, record); saveErr != nil {
					log.Printf("[ERROR] Failed to save pending approval record for %s: %v", tc.Function.Name, saveErr)
				}
			}
			outcome.PendingApprovals = append(outcome.PendingApprovals, record)
			outcome.HasPendingApprovals = true

			// Add a result indicating pending approval
			outcome.Results = append(outcome.Results, domain.ToolExecutionResult{
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				Status:     domain.StatusPendingApproval,
			})
			continue
		}

		// Execute immediately
		record, err := s.executor.ExecuteTool(ctx, chatID, tc.ID, tc.Function.Name, tc.Function.Arguments)
		// Track errors for aggregated reporting
		if err != nil {
			executionErrors = append(executionErrors, fmt.Errorf("tool %s failed: %w", tc.Function.Name, err))
		}

		// Save the execution record
		if messageID != "" {
			if saveErr := s.repo.SaveToolCallRecord(ctx, messageID, record); saveErr != nil {
				log.Printf("[ERROR] Failed to save tool call record for %s: %v", tc.Function.Name, saveErr)
			}
		} else {
			log.Printf("[WARN] No messageID for tool call %s, skipping record save", tc.Function.Name)
		}

		// Check for async behavior and start tracking if applicable
		if err == nil && s.asyncTracker != nil {
			s.maybeStartAsyncTracking(ctx, chatID, tc.ID, tc.Function.Name, record)
		}

		// Save tool response message (parented to the assistant message)
		// This is CRITICAL - if we fail to save the tool result, subsequent completions
		// will fail with "No tool output found for function call" errors
		toolMsg, toolMsgErr := s.repo.SaveToolResponseMessage(ctx, chatID, tc.ID, record.Result, parentMessageID)
		if toolMsgErr != nil {
			log.Printf("[ERROR] Failed to save tool response message for %s (tool_call_id=%s): %v",
				tc.Function.Name, tc.ID, toolMsgErr)
			// Add to execution errors so caller knows something went wrong
			executionErrors = append(executionErrors, fmt.Errorf("failed to save tool result for %s: %w", tc.Function.Name, toolMsgErr))
		} else if toolMsg != nil {
			lastToolMsgID = toolMsg.ID
			log.Printf("[DEBUG] Saved tool response message: id=%s, tool_call_id=%s, parent=%s",
				toolMsg.ID, tc.ID, parentMessageID)
		}

		// Build result using centralized factory
		outcome.Results = append(outcome.Results, NewToolExecutionResult(tc.ID, tc.Function.Name, record, err))
	}

	// Update active leaf to the last tool message
	if lastToolMsgID != "" {
		s.repo.SetActiveLeaf(ctx, chatID, lastToolMsgID)
	}

	// Return aggregated error if any tool failed
	if len(executionErrors) > 0 {
		return outcome, fmt.Errorf("%d of %d tool calls failed: %v", len(executionErrors), len(toolCalls), executionErrors[0])
	}

	return outcome, nil
}

// createPendingApprovalRecord creates a ToolCallRecord for a tool awaiting approval.
func (s *CompletionService) createPendingApprovalRecord(chatID, messageID string, tc domain.ToolCall) *domain.ToolCallRecord {
	return &domain.ToolCallRecord{
		ID:        tc.ID,
		MessageID: messageID,
		ChatID:    chatID,
		ToolName:  tc.Function.Name,
		Arguments: tc.Function.Arguments,
		Status:    domain.StatusPendingApproval,
		StartedAt: time.Now(),
	}
}

// ApprovalResult contains the result of approving a tool call.
type ApprovalResult struct {
	// ToolResult is the execution result after approval.
	ToolResult *domain.ToolCallRecord
	// PendingApprovals remaining after this approval.
	PendingApprovals []*domain.ToolCallRecord
	// AutoContinued indicates if all approvals are resolved and continuation was triggered.
	AutoContinued bool
}

// ApproveToolCall approves and executes a pending tool call.
// Returns the execution result and whether auto-continuation should occur.
func (s *CompletionService) ApproveToolCall(ctx context.Context, chatID, toolCallID string) (*ApprovalResult, error) {
	// Get the pending tool call
	record, err := s.repo.GetToolCallByID(ctx, toolCallID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool call: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("tool call not found: %s", toolCallID)
	}
	if record.Status != domain.StatusPendingApproval {
		return nil, fmt.Errorf("tool call is not pending approval: status=%s", record.Status)
	}
	if record.ChatID != chatID {
		return nil, fmt.Errorf("tool call does not belong to chat")
	}

	// Update status to approved
	if err := s.repo.UpdateToolCallStatus(ctx, toolCallID, domain.StatusApproved, ""); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// Execute the tool
	executedRecord, err := s.executor.ExecuteTool(ctx, chatID, toolCallID, record.ToolName, record.Arguments)
	if err != nil {
		log.Printf("warning: tool execution failed after approval: %v", err)
	}

	// Update the record with execution results
	s.repo.SaveToolCallRecord(ctx, record.MessageID, executedRecord)

	// Save tool response message
	toolMsg, _ := s.repo.SaveToolResponseMessage(ctx, chatID, toolCallID, executedRecord.Result, record.MessageID)
	if toolMsg != nil {
		s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID)
	}

	// Check for remaining pending approvals
	pending, _ := s.repo.GetPendingApprovals(ctx, chatID)

	return &ApprovalResult{
		ToolResult:       executedRecord,
		PendingApprovals: pending,
		AutoContinued:    len(pending) == 0, // Auto-continue when all approvals resolved
	}, nil
}

// RejectToolCall rejects a pending tool call.
func (s *CompletionService) RejectToolCall(ctx context.Context, chatID, toolCallID, reason string) error {
	// Get the pending tool call
	record, err := s.repo.GetToolCallByID(ctx, toolCallID)
	if err != nil {
		return fmt.Errorf("failed to get tool call: %w", err)
	}
	if record == nil {
		return fmt.Errorf("tool call not found: %s", toolCallID)
	}
	if record.Status != domain.StatusPendingApproval {
		return fmt.Errorf("tool call is not pending approval: status=%s", record.Status)
	}
	if record.ChatID != chatID {
		return fmt.Errorf("tool call does not belong to chat")
	}

	// Update status to rejected
	errorMsg := "Rejected by user"
	if reason != "" {
		errorMsg = reason
	}
	if err := s.repo.UpdateToolCallStatus(ctx, toolCallID, domain.StatusRejected, errorMsg); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Save tool response message with rejection info
	rejectionResult := fmt.Sprintf(`{"rejected": true, "reason": %q}`, reason)
	toolMsg, _ := s.repo.SaveToolResponseMessage(ctx, chatID, toolCallID, rejectionResult, record.MessageID)
	if toolMsg != nil {
		s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID)
	}

	return nil
}

// GetPendingApprovals returns all pending tool call approvals for a chat.
func (s *CompletionService) GetPendingApprovals(ctx context.Context, chatID string) ([]*domain.ToolCallRecord, error) {
	return s.repo.GetPendingApprovals(ctx, chatID)
}

// maybeStartAsyncTracking checks if a tool has AsyncBehavior and starts tracking if so.
func (s *CompletionService) maybeStartAsyncTracking(ctx context.Context, chatID, toolCallID, toolName string, record *domain.ToolCallRecord) {
	// Look up the tool to check for async behavior
	tool, scenario, err := s.toolRegistry.GetToolByName(ctx, toolName)
	if err != nil {
		log.Printf("[DEBUG] Could not look up tool %s for async tracking: %v", toolName, err)
		return
	}

	// Check if the tool has async behavior defined
	if tool.Metadata == nil || tool.Metadata.AsyncBehavior == nil {
		return
	}

	asyncBehavior := tool.Metadata.AsyncBehavior
	if asyncBehavior.StatusPolling == nil {
		log.Printf("[DEBUG] Tool %s has AsyncBehavior but no StatusPolling configured", toolName)
		return
	}

	// Parse the result to extract operation ID
	var resultData interface{}
	if err := json.Unmarshal([]byte(record.Result), &resultData); err != nil {
		log.Printf("[WARN] Failed to parse tool result for async tracking: %v", err)
		return
	}

	// Start async tracking
	if err := s.asyncTracker.StartTracking(
		context.Background(), // Use background context for long-running polling
		toolCallID,
		chatID,
		toolName,
		scenario,
		resultData,
		asyncBehavior,
	); err != nil {
		log.Printf("[WARN] Failed to start async tracking for %s: %v", toolName, err)
	}
}

// UpdateChatPreview updates the chat's preview text based on completion result.
func (s *CompletionService) UpdateChatPreview(ctx context.Context, chatID string, result *domain.CompletionResult) error {
	preview := result.PreviewText()
	return s.repo.UpdateChatPreview(ctx, chatID, preview, true)
}

// ChatSettings contains the settings needed for chat completion.
type ChatSettings struct {
	Model            string
	ToolsEnabled     bool
	WebSearchEnabled bool
}

// GetChatSettings retrieves settings for a chat completion.
// Returns nil if chat doesn't exist.
func (s *CompletionService) GetChatSettings(ctx context.Context, chatID string) (*ChatSettings, error) {
	model, toolsEnabled, webSearchEnabled, err := s.repo.GetChatSettingsWithWebSearch(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, nil // Chat not found
	}
	return &ChatSettings{
		Model:            model,
		ToolsEnabled:     toolsEnabled,
		WebSearchEnabled: webSearchEnabled,
	}, nil
}

// CompletionRequest contains validated data needed to make a completion.
type CompletionRequest struct {
	ChatID     string
	Model      string
	Messages   []integrations.OpenRouterMessage
	Tools      []map[string]interface{}
	ToolChoice interface{} // nil for auto, ToolChoiceFunction for forced tool
	Plugins    []integrations.OpenRouterPlugin
	Modalities []string // ["image", "text"] for image generation models
	Streaming  bool
}

// ShouldIncludeTools returns true if tools should be sent with the request.
func (r *CompletionRequest) ShouldIncludeTools() bool {
	return len(r.Tools) > 0
}

// ShouldIncludePlugins returns true if plugins should be sent with the request.
func (r *CompletionRequest) ShouldIncludePlugins() bool {
	return len(r.Plugins) > 0
}

// ShouldIncludeModalities returns true if modalities should be sent with the request.
func (r *CompletionRequest) ShouldIncludeModalities() bool {
	return len(r.Modalities) > 0
}

// PrepareCompletionRequest builds a validated completion request.
// Returns an error if the chat doesn't exist or has no messages.
// Uses sentinel errors (ErrChatNotFound, ErrNoMessages) for type-safe error handling.
// forcedTool is an optional parameter in format "scenario:tool_name" to force the AI to use a specific tool.
func (s *CompletionService) PrepareCompletionRequest(ctx context.Context, chatID string, streaming bool, forcedTool string) (*CompletionRequest, error) {
	// Get chat settings
	settings, err := s.GetChatSettings(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	if settings == nil {
		return nil, ErrChatNotFound
	}

	// Get messages
	messages, err := s.repo.GetMessagesForCompletion(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMessagesFailed, err)
	}
	if len(messages) == 0 {
		return nil, ErrNoMessages
	}

	// Get message IDs to fetch attachments
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	// Fetch attachments for all messages
	attachmentsByMsgID, err := s.repo.GetAttachmentsForMessages(ctx, messageIDs)
	if err != nil {
		log.Printf("warning: failed to fetch attachments: %v", err)
		attachmentsByMsgID = make(map[string][]domain.Attachment) // Continue without attachments
	}

	// Check if this is an image generation model
	isImageGen := s.contextManager.IsImageGenerationModel(ctx, settings.Model)

	// Validate and truncate messages to fit context window
	// This handles all models (including image generation) automatically
	messages, err = s.contextManager.ValidateAndTruncate(ctx, settings.Model, messages)
	if err != nil {
		log.Printf("warning: context validation failed: %v", err)
		// Continue with original messages on error
	}

	// Update attachment map to only include messages that survived truncation
	messageIDSet := make(map[string]bool, len(messages))
	for _, msg := range messages {
		messageIDSet[msg.ID] = true
	}
	filteredAttachments := make(map[string][]domain.Attachment)
	for msgID, atts := range attachmentsByMsgID {
		if messageIDSet[msgID] {
			filteredAttachments[msgID] = atts
		}
	}

	// Convert messages with multimodal support
	orMessages := s.messageConverter.ConvertToOpenRouter(ctx, messages, filteredAttachments)

	// Determine effective web search setting
	// Check if any user message has web_search enabled
	webSearchEnabled := settings.WebSearchEnabled
	log.Printf("[DEBUG] web search: chat default=%v, checking %d messages", webSearchEnabled, len(messages))
	for _, msg := range messages {
		if msg.Role == "user" {
			log.Printf("[DEBUG] user message %s: web_search=%v", msg.ID, msg.WebSearch)
			if msg.WebSearch != nil && *msg.WebSearch {
				webSearchEnabled = true
				break
			}
		}
	}
	log.Printf("[DEBUG] effective web search enabled=%v", webSearchEnabled)

	// Check for PDF attachments
	hasPDF := false
	for _, attachments := range attachmentsByMsgID {
		if HasPDFAttachment(attachments) {
			hasPDF = true
			break
		}
	}

	req := &CompletionRequest{
		ChatID:    chatID,
		Model:     settings.Model,
		Messages:  orMessages,
		Plugins:   s.messageConverter.BuildPlugins(webSearchEnabled, hasPDF),
		Streaming: streaming,
	}

	// Debug: Log messages being sent to OpenRouter to diagnose tool_call issues
	for i, msg := range orMessages {
		toolCallsStr := ""
		if len(msg.ToolCalls) > 0 {
			ids := make([]string, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				ids[j] = tc.ID
			}
			toolCallsStr = fmt.Sprintf(", tool_calls=%v", ids)
		}
		toolCallIDStr := ""
		if msg.ToolCallID != "" {
			toolCallIDStr = fmt.Sprintf(", tool_call_id=%s", msg.ToolCallID)
		}
		contentPreview := ""
		if s, ok := msg.Content.(string); ok && len(s) > 50 {
			contentPreview = s[:50] + "..."
		} else if s, ok := msg.Content.(string); ok {
			contentPreview = s
		} else {
			contentPreview = "[multimodal]"
		}
		log.Printf("[DEBUG] PrepareCompletionRequest msg[%d]: role=%s, content=%q%s%s",
			i, msg.Role, contentPreview, toolCallsStr, toolCallIDStr)
	}

	// Enable image generation modalities if the model supports it
	// Image generation models typically don't support tool use, so skip tools
	if isImageGen {
		req.Modalities = []string{"image", "text"}
		log.Printf("[DEBUG] Image generation enabled for model: %s (tools disabled)", settings.Model)
	}

	// Only add tools for non-image-generation models that have tools enabled
	if settings.ToolsEnabled && !isImageGen {
		tools, err := s.toolRegistry.GetToolsForOpenAI(ctx, chatID)
		if err != nil {
			// Log but don't fail - tools are optional enhancement
			log.Printf("warning: failed to get tools from registry: %v", err)
		} else {
			req.Tools = tools
		}

		// Handle forced tool selection
		if forcedTool != "" && len(req.Tools) > 0 {
			parts := strings.SplitN(forcedTool, ":", 2)
			if len(parts) == 2 {
				toolName := parts[1]
				// Validate tool exists in the tools list
				for _, t := range req.Tools {
					if fn, ok := t["function"].(map[string]interface{}); ok {
						if fn["name"] == toolName {
							req.ToolChoice = integrations.ToolChoiceFunction{
								Type:     "function",
								Function: integrations.ToolChoiceFunctionName{Name: toolName},
							}
							log.Printf("[DEBUG] Forcing tool: %s", toolName)
							break
						}
					}
				}
			}
		}
	}

	return req, nil
}

// Note: isImageGenerationModel logic moved to ContextManager.IsImageGenerationModel

// ManualExecutionResult contains the result of manually executing a tool.
type ManualExecutionResult struct {
	// Result is the tool's return value (JSON encoded).
	Result interface{} `json:"result"`
	// Status is "completed" or "failed".
	Status string `json:"status"`
	// Error contains the error message if failed.
	Error string `json:"error,omitempty"`
	// ExecutionTimeMs is how long execution took.
	ExecutionTimeMs int64 `json:"execution_time_ms"`
	// ToolCallRecord is set if chat_id was provided (tool call added to chat).
	ToolCallRecord *domain.ToolCallRecord `json:"tool_call_record,omitempty"`
}

// ExecuteToolManually executes a tool directly without going through the AI.
// This is used when the user fills in tool parameters manually.
// If chatID is provided, the tool call and result are added to the chat history.
// If chatID is empty, the tool is executed standalone without persistence.
func (s *CompletionService) ExecuteToolManually(ctx context.Context, chatID, scenario, toolName string, arguments map[string]interface{}) (*ManualExecutionResult, error) {
	startTime := time.Now()

	// Convert arguments to JSON string for the executor
	argsJSON, err := json.Marshal(arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize arguments: %w", err)
	}

	// Generate a unique tool call ID
	toolCallID := fmt.Sprintf("manual_%s_%d", toolName, time.Now().UnixNano())

	// If chatID is provided, we need to create a synthetic message for the tool call
	var messageID string
	if chatID != "" {
		// Create a synthetic assistant message indicating manual tool invocation
		syntheticContent := fmt.Sprintf("[Manual tool execution: %s]", toolName)
		toolCall := domain.ToolCall{
			ID:   toolCallID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      toolName,
				Arguments: string(argsJSON),
			},
		}
		msg, err := s.repo.SaveAssistantMessageWithToolCalls(
			ctx, chatID, "", syntheticContent, []domain.ToolCall{toolCall},
			"", "manual_tool_call", 0, "",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to save tool call message: %w", err)
		}
		messageID = msg.ID
	}

	// Execute the tool
	record, execErr := s.executor.ExecuteTool(ctx, chatID, toolCallID, toolName, string(argsJSON))

	executionTime := time.Since(startTime).Milliseconds()

	// Build result
	result := &ManualExecutionResult{
		ExecutionTimeMs: executionTime,
	}

	if execErr != nil {
		result.Status = domain.StatusFailed
		result.Error = execErr.Error()
		result.Result = map[string]string{"error": execErr.Error()}
	} else {
		result.Status = domain.StatusCompleted
		// Parse the JSON result back to interface{}
		var parsedResult interface{}
		if err := json.Unmarshal([]byte(record.Result), &parsedResult); err != nil {
			result.Result = record.Result // Return as string if not valid JSON
		} else {
			result.Result = parsedResult
		}

		// Start async tracking if applicable
		if chatID != "" && s.asyncTracker != nil {
			s.maybeStartAsyncTracking(ctx, chatID, toolCallID, toolName, record)
		}
	}

	// If chatID was provided, save the record and response message
	if chatID != "" && messageID != "" {
		record.MessageID = messageID
		if saveErr := s.repo.SaveToolCallRecord(ctx, messageID, record); saveErr != nil {
			log.Printf("warning: failed to save manual tool call record: %v", saveErr)
		}

		// Save tool response message
		toolMsg, toolMsgErr := s.repo.SaveToolResponseMessage(ctx, chatID, toolCallID, record.Result, messageID)
		if toolMsgErr != nil {
			log.Printf("warning: failed to save manual tool response message: %v", toolMsgErr)
		} else if toolMsg != nil {
			s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID)
		}

		result.ToolCallRecord = record
	}

	return result, nil
}
