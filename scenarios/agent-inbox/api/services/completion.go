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

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
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

// ToolRegistryInterface defines the methods needed by CompletionService.
// This interface enables dependency injection for testing.
type ToolRegistryInterface interface {
	// GetToolByName looks up a tool by name (across all scenarios).
	GetToolByName(ctx context.Context, toolName string) (*toolspb.ToolDefinition, string, error)
	// GetToolsForOpenAI returns enabled tools in OpenAI function-calling format.
	GetToolsForOpenAI(ctx context.Context, chatID string) ([]map[string]interface{}, error)
	// GetToolApprovalRequired checks if a tool requires approval before execution.
	GetToolApprovalRequired(ctx context.Context, chatID, toolName string) (bool, domain.ToolConfigurationScope, error)
}

// CompletionService orchestrates AI chat completion.
// It handles the decision flow for completing a chat with an AI model,
// including tool execution when requested by the model.
//
// Dependencies are injected via interfaces to enable testing:
// - CompletionRepository: database operations
// - ToolExecutorInterface: tool execution
// - ToolRegistryInterface: tool discovery and configuration
// - AsyncTrackerInterface: async operation tracking
type CompletionService struct {
	repo             CompletionRepository
	executor         ToolExecutorInterface
	toolRegistry     ToolRegistryInterface
	contextManager   *ContextManager
	messageConverter *MessageConverter
	storage          StorageService
	asyncTracker     AsyncTrackerInterface
	skills           []SkillPayload // Skills to inject into tool calls as context
}

// NewCompletionService creates a new completion service.
func NewCompletionService(repo *persistence.Repository, storage StorageService) *CompletionService {
	modelRegistry := NewModelRegistry()
	executor := integrations.NewToolExecutor()
	return &CompletionService{
		repo:             repo,
		executor:         executor,
		toolRegistry:     NewToolRegistry(repo, executor),
		contextManager:   NewContextManager(modelRegistry, config.Default()),
		messageConverter: NewMessageConverter(storage),
		storage:          storage,
	}
}

// NewCompletionServiceWithRegistry creates a completion service with an injected registry.
// This is the constructor for testing scenarios that only need registry injection.
func NewCompletionServiceWithRegistry(repo *persistence.Repository, registry ToolRegistryInterface, storage StorageService) *CompletionService {
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

// CompletionServiceDeps contains all dependencies for CompletionService.
// Used by NewCompletionServiceWithDeps for full dependency injection in tests.
type CompletionServiceDeps struct {
	Repo         CompletionRepository
	Executor     ToolExecutorInterface
	Registry     ToolRegistryInterface
	AsyncTracker AsyncTrackerInterface
	Storage      StorageService
}

// NewCompletionServiceWithDeps creates a completion service with all dependencies injected.
// This is the primary constructor for unit testing.
func NewCompletionServiceWithDeps(deps CompletionServiceDeps) *CompletionService {
	modelRegistry := NewModelRegistry()
	return &CompletionService{
		repo:             deps.Repo,
		executor:         deps.Executor,
		toolRegistry:     deps.Registry,
		contextManager:   NewContextManager(modelRegistry, config.Default()),
		messageConverter: NewMessageConverter(deps.Storage),
		storage:          deps.Storage,
		asyncTracker:     deps.AsyncTracker,
	}
}

// SetAsyncTracker sets the async tracker for tracking long-running tool operations.
// This is called after construction to avoid circular dependencies.
func (s *CompletionService) SetAsyncTracker(tracker AsyncTrackerInterface) {
	s.asyncTracker = tracker
}

// SetSkills sets the skills to inject into tool calls as context attachments.
// Skills are converted to context attachments when passed to external tools like agent-manager.
func (s *CompletionService) SetSkills(skills interface{}) {
	// Convert from handlers.SkillPayload to services.SkillPayload
	// This avoids circular imports between handlers and services
	if skills == nil {
		s.skills = nil
		return
	}

	// Use JSON marshal/unmarshal for type conversion
	jsonBytes, err := json.Marshal(skills)
	if err != nil {
		log.Printf("[WARN] Failed to marshal skills: %v", err)
		return
	}

	var parsed []SkillPayload
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		log.Printf("[WARN] Failed to unmarshal skills: %v", err)
		return
	}

	s.skills = parsed
	log.Printf("[DEBUG] CompletionService: set %d skills for context injection", len(s.skills))
}

// injectSkillsIntoArgs injects skills as context_attachments into tool arguments.
// Skills are filtered by targetToolId if specified - only skills targeting this tool
// or skills with no target (apply to all tools) are included.
// Returns the original arguments if no skills are set or on any error.
func (s *CompletionService) injectSkillsIntoArgs(toolName, arguments string) string {
	if len(s.skills) == 0 {
		return arguments
	}

	// Parse existing arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		log.Printf("[WARN] Failed to parse tool arguments for skill injection: %v", err)
		return arguments
	}

	// Filter skills by targetToolId
	var contextAttachments []map[string]interface{}
	for _, skill := range s.skills {
		// Include skill if it targets this tool or has no target (applies to all)
		if skill.TargetToolID == "" || skill.TargetToolID == toolName {
			attachment := map[string]interface{}{
				"type":    "skill",
				"key":     skill.Key,
				"label":   skill.Label,
				"content": skill.Content,
			}
			if len(skill.Tags) > 0 {
				attachment["tags"] = skill.Tags
			}
			contextAttachments = append(contextAttachments, attachment)
		}
	}

	if len(contextAttachments) == 0 {
		return arguments
	}

	// Inject context attachments
	args["_context_attachments"] = contextAttachments
	log.Printf("[DEBUG] Injected %d skills as context_attachments into %s", len(contextAttachments), toolName)

	// Re-serialize
	enhanced, err := json.Marshal(args)
	if err != nil {
		log.Printf("[WARN] Failed to re-serialize tool arguments: %v", err)
		return arguments
	}

	return string(enhanced)
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
// Some tools may execute immediately, others may require approval, and some may run asynchronously.
type ToolExecutionOutcome struct {
	// Results contains execution results for tools that ran immediately.
	Results []domain.ToolExecutionResult
	// PendingApprovals contains tool calls that require user approval.
	PendingApprovals []*domain.ToolCallRecord
	// HasPendingApprovals indicates if any tools are waiting for approval.
	HasPendingApprovals bool
	// AsyncOperations contains info about tools running asynchronously.
	AsyncOperations []domain.AsyncOperationInfo
	// HasAsyncOperations indicates if any tools are running in the background.
	HasAsyncOperations bool
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

		// Inject skills as context attachments into tool arguments
		enhancedArgs := s.injectSkillsIntoArgs(tc.Function.Name, tc.Function.Arguments)

		// Execute immediately
		record, err := s.executor.ExecuteTool(ctx, chatID, tc.ID, tc.Function.Name, enhancedArgs)
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
		var asyncOpInfo *domain.AsyncOperationInfo
		if err == nil && s.asyncTracker != nil {
			asyncOpInfo = s.maybeStartAsyncTracking(ctx, chatID, tc.ID, tc.Function.Name, record)
			if asyncOpInfo != nil {
				outcome.AsyncOperations = append(outcome.AsyncOperations, *asyncOpInfo)
				outcome.HasAsyncOperations = true
			}
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

		// Build result using centralized factory (with async info if applicable)
		execResult := NewToolExecutionResult(tc.ID, tc.Function.Name, record, err)
		if asyncOpInfo != nil {
			execResult.IsAsync = true
			execResult.AsyncRunID = asyncOpInfo.RunID
		}
		outcome.Results = append(outcome.Results, execResult)
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

// SaveToolResult saves a tool execution result as a tool response message.
// This is used for async tool results that need to be injected into the conversation
// after the tool completes in the background.
func (s *CompletionService) SaveToolResult(ctx context.Context, chatID string, result domain.ToolExecutionResult, parentMessageID string) error {
	// Convert result to JSON
	var resultJSON string
	if result.Error != "" {
		resultJSON = fmt.Sprintf(`{"status": %q, "error": %q}`, result.Status, result.Error)
	} else {
		resultBytes, err := json.Marshal(result.Result)
		if err != nil {
			resultJSON = fmt.Sprintf(`{"status": %q, "result": "marshal error: %s"}`, result.Status, err.Error())
		} else {
			resultJSON = fmt.Sprintf(`{"status": %q, "result": %s}`, result.Status, string(resultBytes))
		}
	}

	// Save tool response message
	toolMsg, err := s.repo.SaveToolResponseMessage(ctx, chatID, result.ToolCallID, resultJSON, parentMessageID)
	if err != nil {
		return fmt.Errorf("failed to save tool response message: %w", err)
	}

	// Update active leaf to the new tool response message
	if toolMsg != nil {
		if err := s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID); err != nil {
			log.Printf("[WARN] Failed to set active leaf for async tool result: %v", err)
		}
	}

	return nil
}

// maybeStartAsyncTracking checks if a tool has AsyncBehavior and starts tracking if so.
// Returns AsyncOperationInfo if tracking started successfully, nil otherwise.
func (s *CompletionService) maybeStartAsyncTracking(ctx context.Context, chatID, toolCallID, toolName string, record *domain.ToolCallRecord) *domain.AsyncOperationInfo {
	// Look up the tool to check for async behavior
	tool, scenario, err := s.toolRegistry.GetToolByName(ctx, toolName)
	if err != nil {
		log.Printf("[DEBUG] Could not look up tool %s for async tracking: %v", toolName, err)
		return nil
	}

	// Check if the tool has async behavior defined
	if tool.Metadata == nil || tool.Metadata.AsyncBehavior == nil {
		return nil
	}

	asyncBehavior := tool.Metadata.AsyncBehavior
	if asyncBehavior.StatusPolling == nil {
		log.Printf("[DEBUG] Tool %s has AsyncBehavior but no StatusPolling configured", toolName)
		return nil
	}

	// Parse the result to extract operation ID
	var resultData interface{}
	if err := json.Unmarshal([]byte(record.Result), &resultData); err != nil {
		log.Printf("[WARN] Failed to parse tool result for async tracking: %v", err)
		return nil
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
		return nil
	}

	// Get the operation to retrieve the extracted run ID
	op := s.asyncTracker.GetOperation(toolCallID)
	if op == nil {
		log.Printf("[WARN] Async tracking started but operation not found: %s", toolCallID)
		return nil
	}

	log.Printf("[DEBUG] Async operation started: tool=%s, toolCallID=%s, runID=%s", toolName, toolCallID, op.ExternalRunID)

	return &domain.AsyncOperationInfo{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		RunID:      op.ExternalRunID,
		Scenario:   scenario,
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

	// Inject async guidance if operations are active
	// This tells the AI not to call status/polling tools because results will arrive automatically
	if s.asyncTracker != nil {
		activeOps := s.asyncTracker.GetActiveOperations(chatID)
		if len(activeOps) > 0 {
			guidance := s.buildAsyncGuidanceMessage(activeOps)
			orMessages = append([]integrations.OpenRouterMessage{
				{Role: "system", Content: guidance},
			}, orMessages...)
			log.Printf("[DEBUG] Injected async guidance for %d active operations", len(activeOps))
		}
	}

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

	// Check for forced tool FIRST, before tools_enabled check.
	// This allows forcing a tool even when tools_enabled=false or the tool is disabled.
	var forcedToolDef map[string]interface{}
	if forcedTool != "" && !isImageGen {
		var err error
		forcedToolDef, err = s.getForcedToolDefinition(ctx, forcedTool)
		if err != nil {
			log.Printf("[WARN] Forced tool '%s' not found or invalid: %v", forcedTool, err)
			// Continue without forcing - fall through to normal tool handling
		}
	}

	// Determine if we should include tools:
	// - If forcing a specific tool, include it (bypasses tools_enabled)
	// - If tools_enabled, include all enabled tools
	// - Never include tools for image generation models
	shouldIncludeTools := (settings.ToolsEnabled || forcedToolDef != nil) && !isImageGen

	if shouldIncludeTools {
		if forcedToolDef != nil {
			// Force mode: only send the forced tool
			// Extract tool name for ToolChoice
			toolName := ""
			if fn, ok := forcedToolDef["function"].(map[string]interface{}); ok {
				if name, ok := fn["name"].(string); ok {
					toolName = name
				}
			}

			req.Tools = []map[string]interface{}{forcedToolDef}
			req.ToolChoice = integrations.ToolChoiceFunction{
				Type:     "function",
				Function: integrations.ToolChoiceFunctionName{Name: toolName},
			}
			log.Printf("[DEBUG] Forced tool mode: sending only tool '%s' (tools_enabled=%v)", toolName, settings.ToolsEnabled)
		} else if settings.ToolsEnabled {
			// Normal mode: send all enabled tools
			tools, err := s.toolRegistry.GetToolsForOpenAI(ctx, chatID)
			if err != nil {
				// Log but don't fail - tools are optional enhancement
				log.Printf("warning: failed to get tools from registry: %v", err)
			} else {
				req.Tools = tools
			}
		}
	}

	// Validation: log if forced tool was specified but couldn't be set
	if forcedTool != "" && req.ToolChoice == nil {
		log.Printf("[ERROR] Forced tool '%s' was specified but not set. Check format: 'scenario:tool_name'", forcedTool)
	}

	return req, nil
}

// Note: isImageGenerationModel logic moved to ContextManager.IsImageGenerationModel

// getForcedToolDefinition retrieves a tool by name, bypassing enabled filters.
// This allows forcing a tool even when tools_enabled=false or the tool is disabled.
// The forcedTool format is "scenario:tool_name".
func (s *CompletionService) getForcedToolDefinition(ctx context.Context, forcedTool string) (map[string]interface{}, error) {
	parts := strings.SplitN(forcedTool, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid forced tool format: expected 'scenario:tool_name', got '%s'", forcedTool)
	}

	toolName := parts[1]

	// Get the tool directly by name (bypasses enabled filter)
	toolDef, _, err := s.toolRegistry.GetToolByName(ctx, toolName)
	if err != nil {
		return nil, fmt.Errorf("tool '%s' not found: %w", toolName, err)
	}

	// Convert to OpenAI format
	openAITool := domain.ToOpenAIFunction(toolDef)
	return openAITool, nil
}

// buildAsyncGuidanceMessage creates a system message that instructs the AI
// about active async operations. This prevents the AI from calling status/polling
// tools repeatedly, as results will be delivered automatically when ready.
func (s *CompletionService) buildAsyncGuidanceMessage(ops []*AsyncOperation) string {
	var toolNames []string
	for _, op := range ops {
		toolNames = append(toolNames, op.ToolName)
	}

	return fmt.Sprintf(
		"IMPORTANT: The following tools are currently executing asynchronously: %s. "+
			"You will receive their results automatically when they complete. "+
			"DO NOT call any status-checking or polling tools - the results will be delivered to you without any action on your part. "+
			"Please wait patiently or continue with other tasks while these operations complete.",
		strings.Join(toolNames, ", "),
	)
}

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
