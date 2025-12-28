// Package integrations provides clients for external services.
// This file contains model pricing data for cost calculation.
package integrations

import "agent-inbox/domain"

// ModelPricing contains pricing information for a model.
// Prices are in USD cents per million tokens.
type ModelPricing struct {
	Model              string  // Model ID
	PromptCostPerM     float64 // Cost per million prompt (input) tokens in cents
	CompletionCostPerM float64 // Cost per million completion (output) tokens in cents
}

// modelPricing contains current pricing for supported models.
// Source: https://openrouter.ai/docs#models (prices as of Dec 2024)
// Prices are in USD cents per million tokens.
var modelPricing = map[string]ModelPricing{
	// Anthropic
	"anthropic/claude-3.5-sonnet": {
		Model:              "anthropic/claude-3.5-sonnet",
		PromptCostPerM:     300,  // $0.003/1K = $3/1M = 300 cents
		CompletionCostPerM: 1500, // $0.015/1K = $15/1M = 1500 cents
	},
	"anthropic/claude-3-haiku": {
		Model:              "anthropic/claude-3-haiku",
		PromptCostPerM:     25,  // $0.00025/1K
		CompletionCostPerM: 125, // $0.00125/1K
	},
	"anthropic/claude-3-opus": {
		Model:              "anthropic/claude-3-opus",
		PromptCostPerM:     1500, // $0.015/1K
		CompletionCostPerM: 7500, // $0.075/1K
	},
	// OpenAI
	"openai/gpt-4o": {
		Model:              "openai/gpt-4o",
		PromptCostPerM:     250,  // $0.0025/1K
		CompletionCostPerM: 1000, // $0.01/1K
	},
	"openai/gpt-4o-mini": {
		Model:              "openai/gpt-4o-mini",
		PromptCostPerM:     15, // $0.00015/1K
		CompletionCostPerM: 60, // $0.0006/1K
	},
	"openai/gpt-4-turbo": {
		Model:              "openai/gpt-4-turbo",
		PromptCostPerM:     1000, // $0.01/1K
		CompletionCostPerM: 3000, // $0.03/1K
	},
	// Google
	"google/gemini-pro-1.5": {
		Model:              "google/gemini-pro-1.5",
		PromptCostPerM:     125, // $0.00125/1K
		CompletionCostPerM: 500, // $0.005/1K
	},
	"google/gemini-flash-1.5": {
		Model:              "google/gemini-flash-1.5",
		PromptCostPerM:     7.5, // $0.000075/1K
		CompletionCostPerM: 30,  // $0.0003/1K
	},
	// Meta
	"meta-llama/llama-3.1-70b-instruct": {
		Model:              "meta-llama/llama-3.1-70b-instruct",
		PromptCostPerM:     52, // $0.00052/1K
		CompletionCostPerM: 75, // $0.00075/1K
	},
	"meta-llama/llama-3.1-8b-instruct": {
		Model:              "meta-llama/llama-3.1-8b-instruct",
		PromptCostPerM:     5.5, // $0.000055/1K
		CompletionCostPerM: 8,   // $0.00008/1K
	},
	// Mistral
	"mistralai/mistral-large": {
		Model:              "mistralai/mistral-large",
		PromptCostPerM:     200, // $0.002/1K
		CompletionCostPerM: 600, // $0.006/1K
	},
	"mistralai/mistral-medium": {
		Model:              "mistralai/mistral-medium",
		PromptCostPerM:     270, // $0.0027/1K
		CompletionCostPerM: 810, // $0.0081/1K
	},
}

// GetModelPricing returns pricing for a model, or nil if unknown.
func GetModelPricing(model string) *ModelPricing {
	if p, ok := modelPricing[model]; ok {
		return &p
	}
	return nil
}

// CalculateUsageCost calculates the cost for a usage record.
// Returns prompt cost, completion cost, and total cost in cents.
func CalculateUsageCost(model string, promptTokens, completionTokens int) (promptCost, completionCost, totalCost float64) {
	pricing := GetModelPricing(model)
	if pricing == nil {
		// Unknown model - return zero costs
		return 0, 0, 0
	}

	// Calculate costs: (tokens / 1_000_000) * costPerMillion
	promptCost = float64(promptTokens) * pricing.PromptCostPerM / 1_000_000
	completionCost = float64(completionTokens) * pricing.CompletionCostPerM / 1_000_000
	totalCost = promptCost + completionCost

	return promptCost, completionCost, totalCost
}

// CreateUsageRecord creates a usage record from completion data.
func CreateUsageRecord(chatID, messageID, model string, usage *domain.Usage) *domain.UsageRecord {
	if usage == nil {
		return nil
	}

	promptCost, completionCost, totalCost := CalculateUsageCost(model, usage.PromptTokens, usage.CompletionTokens)

	return &domain.UsageRecord{
		ChatID:           chatID,
		MessageID:        messageID,
		Model:            model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		PromptCost:       promptCost,
		CompletionCost:   completionCost,
		TotalCost:        totalCost,
	}
}
