// Package integrations provides clients for external services.
// This file contains model fetching and message conversion utilities for OpenRouter.
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agent-inbox/domain"
)

// ConvertMessages converts domain messages to OpenRouter format.
// Messages with missing or invalid "role" fields are skipped.
func ConvertMessages(messages []map[string]interface{}) []OpenRouterMessage {
	result := make([]OpenRouterMessage, 0, len(messages))
	for _, m := range messages {
		role, ok := m["role"].(string)
		if !ok {
			continue
		}
		msg := OpenRouterMessage{
			Role: role,
		}
		if content, ok := m["content"].(string); ok {
			msg.Content = content
		}
		if tcid, ok := m["tool_call_id"].(string); ok {
			msg.ToolCallID = tcid
		}
		if tcs, ok := m["tool_calls"].([]domain.ToolCall); ok {
			msg.ToolCalls = tcs
		}
		result = append(result, msg)
	}
	return result
}

// FetchModels fetches available models from the resource-openrouter CLI.
func FetchModels(ctx context.Context) ([]ModelInfo, error) {
	// Check if resource-openrouter is available
	path, err := exec.LookPath("resource-openrouter")
	if err != nil {
		return nil, fmt.Errorf("resource-openrouter CLI not found: %w", err)
	}

	// Set timeout for the command
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, path, "content", "models", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models from resource-openrouter: %w", err)
	}

	// Parse JSON response
	var resp ModelsResponse
	if err := json.Unmarshal(trimToJSON(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	if len(resp.Models) == 0 {
		return nil, fmt.Errorf("no models returned from resource-openrouter")
	}

	return resp.Models, nil
}

// trimToJSON removes leading non-JSON lines (warnings/logs) to allow parsing.
func trimToJSON(raw []byte) []byte {
	data := strings.TrimSpace(string(raw))
	if data == "" {
		return raw
	}

	// Find the first '{' or '[' which should start the JSON payload.
	idxObj := strings.IndexRune(data, '{')
	idxArr := strings.IndexRune(data, '[')

	start := -1
	if idxObj >= 0 && idxArr >= 0 {
		start = idxObj
		if idxArr < idxObj {
			start = idxArr
		}
	} else if idxObj >= 0 {
		start = idxObj
	} else if idxArr >= 0 {
		start = idxArr
	}

	if start > 0 {
		return []byte(data[start:])
	}
	return []byte(data)
}

// CreateUsageRecordFromStats creates a UsageRecord from OpenRouter generation stats.
// Converts cost from USD to cents for consistency with existing schema.
func CreateUsageRecordFromStats(chatID, messageID string, stats *GenerationStats) *domain.UsageRecord {
	if stats == nil {
		return nil
	}

	// Use native token counts for accuracy, fallback to normalized if not available
	promptTokens := stats.NativeTokensPrompt
	completionTokens := stats.NativeTokensCompletion
	if promptTokens == 0 && completionTokens == 0 {
		promptTokens = stats.TokensPrompt
		completionTokens = stats.TokensCompletion
	}

	// Convert USD to cents (* 100)
	totalCostCents := stats.TotalCost * 100

	return &domain.UsageRecord{
		ChatID:           chatID,
		MessageID:        messageID,
		Model:            stats.Model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptCost:       0, // OpenRouter only provides total cost
		CompletionCost:   0, // OpenRouter only provides total cost
		TotalCost:        totalCostCents,
	}
}
