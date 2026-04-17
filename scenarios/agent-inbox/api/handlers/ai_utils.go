package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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
			stats, fetchErr = fetchGenerationStatsWithRetry(ctx, generationID)
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
			saveFallbackUsageRecord(ctx, h, chatID, messageID, model, fallbackUsage)
		}

		// Log final failure if we couldn't get generation stats
		if fetchErr != nil {
			log.Printf("[WARN] All attempts to fetch generation stats failed for %s: %v",
				generationID, fetchErr)
		}
	}()
}

// fetchGenerationStatsWithRetry attempts to fetch generation stats from OpenRouter
// with exponential backoff. Returns nil stats if all attempts fail.
func fetchGenerationStatsWithRetry(ctx context.Context, generationID string) (*integrations.GenerationStats, error) {
	orClient, err := integrations.NewOpenRouterClient()
	if err != nil {
		log.Printf("[WARN] Failed to create OpenRouter client for usage stats: %v", err)
		return nil, err
	}

	// Exponential backoff: wait before each attempt since OpenRouter needs time to index
	// Delays: 2s, 4s, 8s (total wait ~14s before giving up)
	delays := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

	var fetchErr error
	for attempt, delay := range delays {
		time.Sleep(delay)

		stats, err := orClient.FetchGenerationStats(ctx, generationID)
		if err == nil {
			return stats, nil
		}
		fetchErr = err

		log.Printf("[DEBUG] Generation stats attempt %d/%d for %s: %v",
			attempt+1, len(delays), generationID, err)
	}

	return nil, fetchErr
}

// saveFallbackUsageRecord saves a usage record with token counts but no cost data.
func saveFallbackUsageRecord(ctx context.Context, h *Handlers, chatID, messageID, model string, usage *domain.Usage) {
	fallbackRecord := &domain.UsageRecord{
		ChatID:           chatID,
		MessageID:        messageID,
		Model:            model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		TotalCost:        0, // Cost unknown - generation stats unavailable
	}
	if err := h.Repo.SaveUsageRecord(ctx, fallbackRecord); err != nil {
		log.Printf("[WARN] Failed to save fallback usage record: %v", err)
		return
	}
	log.Printf("[INFO] Saved fallback usage (no cost data): model=%s, tokens=%d",
		model, fallbackRecord.TotalTokens)
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

// checkAndDeactivateTemplate checks if a tool call matches an active template's
// suggested tool and deactivates the template if so. Returns true if template was deactivated.
func checkAndDeactivateTemplate(
	ctx context.Context,
	repo interface {
		ClearActiveTemplate(ctx context.Context, chatID string) error
	},
	chatID, toolName string,
	activeTemplateToolIDs []string,
	toolResult *domain.ToolExecutionResult,
) bool {
	for _, templateToolID := range activeTemplateToolIDs {
		if strings.HasSuffix(templateToolID, ":"+toolName) || templateToolID == toolName {
			toolResult.DeactivateTemplate = true
			if clearErr := repo.ClearActiveTemplate(ctx, chatID); clearErr != nil {
				log.Printf("warning: failed to clear active template: %v", clearErr)
			}
			return true
		}
	}
	return false
}
