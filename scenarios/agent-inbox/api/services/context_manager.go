// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the ContextManager service for context window management.
// The manager validates that message history fits within a model's context limit
// and truncates oldest messages when necessary.
//
// DESIGN PRINCIPLES:
// - Use actual token counts from OpenRouter when available
// - Fall back to tiktoken-go estimation only when necessary
// - Preserve system messages and the last user message
// - Remove oldest messages first (most recent context is most relevant)
package services

import (
	"context"
	"log"
	"strings"

	"agent-inbox/config"
	"agent-inbox/domain"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// ContextManager handles context window validation and message truncation.
type ContextManager struct {
	modelRegistry *ModelRegistry
	cfg           *config.Config
	tokenizer     *tiktoken.Tiktoken
}

// ContextBudget contains the calculated token budget for a request.
type ContextBudget struct {
	ModelContextLength   int  // Total context length from model
	ReservedForResponse  int  // Tokens reserved for completion
	AvailableForMessages int  // Budget available for input messages
	CurrentUsage         int  // Current total of message tokens
	ExceedsLimit         bool // Whether current usage exceeds available
	TruncationNeeded     int  // How many tokens need to be removed
}

// NewContextManager creates a new ContextManager with the given registry.
func NewContextManager(registry *ModelRegistry, cfg *config.Config) *ContextManager {
	// Initialize tiktoken with cl100k_base encoding (used by GPT-4, Claude, etc.)
	// This is a fallback for when we don't have actual token counts.
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("[WARN] Failed to initialize tiktoken: %v (will use character-based estimation)", err)
	}

	return &ContextManager{
		modelRegistry: registry,
		cfg:           cfg,
		tokenizer:     enc,
	}
}

// CalculateBudget returns the token budget for a model and the current usage.
func (cm *ContextManager) CalculateBudget(ctx context.Context, modelID string, messages []domain.Message) (*ContextBudget, error) {
	// Get model's context length
	contextLength, err := cm.modelRegistry.GetContextLength(ctx, modelID)
	if err != nil {
		log.Printf("[WARN] Failed to get context length for %s: %v (using default)", modelID, err)
	}

	// Calculate reserved portion for response
	reservePercent := cm.cfg.AI.CompletionReservePercent
	if reservePercent <= 0 || reservePercent >= 100 {
		reservePercent = 25
	}
	reservedForResponse := contextLength * reservePercent / 100
	availableForMessages := contextLength - reservedForResponse

	// Sum up token counts from messages
	currentUsage := cm.SumTokens(messages)

	budget := &ContextBudget{
		ModelContextLength:   contextLength,
		ReservedForResponse:  reservedForResponse,
		AvailableForMessages: availableForMessages,
		CurrentUsage:         currentUsage,
		ExceedsLimit:         currentUsage > availableForMessages,
	}

	if budget.ExceedsLimit {
		budget.TruncationNeeded = currentUsage - availableForMessages
	}

	return budget, nil
}

// SumTokens calculates the total tokens across all messages.
// Uses stored token counts when available, falls back to tiktoken estimation.
func (cm *ContextManager) SumTokens(messages []domain.Message) int {
	total := 0
	for _, msg := range messages {
		if msg.TokenCount > 0 {
			// Use stored token count (from OpenRouter)
			total += msg.TokenCount
		} else {
			// Fall back to tiktoken estimation
			total += cm.CountTokens(msg.Content)
		}
	}
	return total
}

// CountTokens estimates token count for a string using tiktoken.
// Falls back to character-based estimation if tiktoken is unavailable.
func (cm *ContextManager) CountTokens(text string) int {
	if cm.tokenizer != nil {
		tokens := cm.tokenizer.Encode(text, nil, nil)
		return len(tokens)
	}
	// Fallback: rough estimate of 1 token per 4 characters
	return len(text) / 4
}

// ValidateAndTruncate validates that messages fit within context and truncates if needed.
// Returns the (possibly truncated) message list.
//
// Truncation rules:
// 1. Never remove system messages
// 2. Never remove the last user message (the current request)
// 3. Remove oldest messages first
func (cm *ContextManager) ValidateAndTruncate(ctx context.Context, modelID string, messages []domain.Message) ([]domain.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	budget, err := cm.CalculateBudget(ctx, modelID, messages)
	if err != nil {
		return messages, err
	}

	// If within budget, no truncation needed
	if !budget.ExceedsLimit {
		log.Printf("[DEBUG] Context within budget: %d/%d tokens used",
			budget.CurrentUsage, budget.AvailableForMessages)
		return messages, nil
	}

	log.Printf("[INFO] Context exceeds limit: %d/%d tokens, truncating %d tokens",
		budget.CurrentUsage, budget.AvailableForMessages, budget.TruncationNeeded)

	// Truncate messages
	return cm.truncateMessages(messages, budget)
}

// truncateMessages removes oldest non-protected messages until within budget.
func (cm *ContextManager) truncateMessages(messages []domain.Message, budget *ContextBudget) ([]domain.Message, error) {
	if len(messages) <= 1 {
		// Can't truncate if only one message
		return messages, nil
	}

	// Identify protected messages (system and last user message)
	protected := make(map[int]bool)
	lastUserIdx := -1

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == domain.RoleSystem {
			protected[i] = true
		}
		if messages[i].Role == domain.RoleUser && lastUserIdx == -1 {
			lastUserIdx = i
			protected[i] = true
		}
	}

	// Calculate tokens to remove
	tokensToRemove := budget.TruncationNeeded
	removed := 0

	// Build result list, skipping oldest non-protected messages until under budget
	result := make([]domain.Message, 0, len(messages))
	for i, msg := range messages {
		if protected[i] {
			// Always include protected messages
			result = append(result, msg)
			continue
		}

		// Calculate this message's token count
		msgTokens := msg.TokenCount
		if msgTokens == 0 {
			msgTokens = cm.CountTokens(msg.Content)
		}

		if tokensToRemove > 0 {
			// Skip this message (remove it)
			tokensToRemove -= msgTokens
			removed++
			log.Printf("[DEBUG] Truncated message %d (%s): %d tokens", i, msg.Role, msgTokens)
		} else {
			// Keep this message
			result = append(result, msg)
		}
	}

	log.Printf("[INFO] Truncated %d messages to fit context window", removed)
	return result, nil
}

// GetMessageTokenCount returns the token count for a message, using stored value or estimation.
func (cm *ContextManager) GetMessageTokenCount(msg *domain.Message) int {
	if msg.TokenCount > 0 {
		return msg.TokenCount
	}
	return cm.CountTokens(msg.Content)
}

// EstimateRequestTokens estimates the total tokens for a completion request.
// This includes the messages plus a base overhead for the request format.
func (cm *ContextManager) EstimateRequestTokens(messages []domain.Message) int {
	total := cm.SumTokens(messages)
	// Add estimated overhead for message formatting (role labels, etc.)
	// Approximately 4 tokens per message for formatting overhead
	overhead := len(messages) * 4
	return total + overhead
}

// IsImageGenerationModel checks if a model supports image generation.
// Delegates to the model registry.
func (cm *ContextManager) IsImageGenerationModel(ctx context.Context, modelID string) bool {
	supports, err := cm.modelRegistry.SupportsImageGeneration(ctx, modelID)
	if err != nil {
		// Fall back to pattern matching
		return isImageGenerationModelByPattern(modelID)
	}
	return supports
}

// isImageGenerationModelByPattern uses pattern matching as a fallback.
var imageGenerationPatterns = []string{
	"gemini-2.5-flash-image",
	"gemini-2.0-flash-image",
	"flux",
	"dall-e",
	"stable-diffusion",
	"sdxl",
	"imagen",
}

func isImageGenerationModelByPattern(modelID string) bool {
	modelIDLower := strings.ToLower(modelID)
	for _, pattern := range imageGenerationPatterns {
		if strings.Contains(modelIDLower, pattern) {
			return true
		}
	}
	return false
}
