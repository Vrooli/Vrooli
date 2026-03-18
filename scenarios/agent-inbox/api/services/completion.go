// Package services contains business logic orchestration.
// Services coordinate between handlers, persistence, and integrations.
//
// The completion service is split across multiple files:
//   - completion.go: Core types, constructor, configuration, and skills
//   - completion_save.go: Saving completion results, images, and async tracking
//   - completion_tools.go: Tool call execution flows
//   - completion_tools_approval.go: Tool call approval and rejection flows
//   - completion_prepare.go: Completion request preparation and context building
//   - completion_manual.go: Manual tool execution and async result saving
package services

import (
	"context"
	"encoding/json"
	"log"

	"agent-inbox/config"
	"agent-inbox/domain"

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
// - ToolPersistence: atomic tool result saving
type CompletionService struct {
	repo             CompletionRepository
	executor         ToolExecutorInterface
	toolRegistry     ToolRegistryInterface
	contextManager   *ContextManager
	messageConverter *MessageConverter
	storage          StorageService
	asyncTracker     AsyncTrackerInterface
	toolPersistence  *ToolPersistence
	skills           []SkillPayload // Skills to inject into tool calls as context
}

// CompletionServiceDeps contains all dependencies for CompletionService.
// Used by NewCompletionServiceWithDeps for full dependency injection in tests.
type CompletionServiceDeps struct {
	Repo            CompletionRepository
	Executor        ToolExecutorInterface
	Registry        ToolRegistryInterface
	AsyncTracker    AsyncTrackerInterface
	Storage         StorageService
	ModelRegistry   *ModelRegistry   // Optional: if nil, a new one is created
	ToolPersistence *ToolPersistence // Optional: for atomic tool saves
}

// NewCompletionServiceWithDeps creates a completion service with all dependencies injected.
// This is the primary constructor for production use and unit testing.
// If deps.ModelRegistry is nil, a new one is created.
func NewCompletionServiceWithDeps(deps CompletionServiceDeps) *CompletionService {
	modelRegistry := deps.ModelRegistry
	if modelRegistry == nil {
		modelRegistry = NewModelRegistry()
	}
	return &CompletionService{
		repo:             deps.Repo,
		executor:         deps.Executor,
		toolRegistry:     deps.Registry,
		contextManager:   NewContextManager(modelRegistry, config.Default()),
		messageConverter: NewMessageConverter(deps.Storage),
		storage:          deps.Storage,
		asyncTracker:     deps.AsyncTracker,
		toolPersistence:  deps.ToolPersistence,
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
	if skills == nil {
		s.skills = nil
		return
	}

	// Use JSON marshal/unmarshal for type conversion to avoid circular imports
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

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		log.Printf("[WARN] Failed to parse tool arguments for skill injection: %v", err)
		return arguments
	}

	// Filter skills by targetToolId
	var contextAttachments []map[string]interface{}
	for _, skill := range s.skills {
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

	args["_context_attachments"] = contextAttachments
	log.Printf("[DEBUG] Injected %d skills as context_attachments into %s", len(contextAttachments), toolName)

	enhanced, err := json.Marshal(args)
	if err != nil {
		log.Printf("[WARN] Failed to re-serialize tool arguments: %v", err)
		return arguments
	}

	return string(enhanced)
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
