package intelligence

const (
	charsPerToken               = 4
	tokenEstimationSafetyMargin = 1.5
	defaultCompletionTokens     = 1000
	messageFramingOverhead      = 10
)

// AllowedModels returns a copy so callers cannot mutate the gateway policy.
func AllowedModels() map[string]bool {
	return map[string]bool{
		"openai/gpt-4o":               true,
		"openai/gpt-4o-mini":          true,
		"anthropic/claude-3.5-sonnet": true,
		"anthropic/claude-3-haiku":    true,
		"google/gemini-pro-1.5":       true,
		"google/gemini-flash-1.5":     true,
	}
}

// DefaultModelPricing returns the provider pricing policy in internal credit
// units per 1K tokens. It returns a fresh map for safe test customization.
func DefaultModelPricing() map[string]ModelPricing {
	return map[string]ModelPricing{
		"openai/gpt-4o":               {PromptCostPer1K: 2500000, CompletionCostPer1K: 10000000},
		"openai/gpt-4o-mini":          {PromptCostPer1K: 150000, CompletionCostPer1K: 600000},
		"anthropic/claude-3.5-sonnet": {PromptCostPer1K: 3000000, CompletionCostPer1K: 15000000},
		"anthropic/claude-3-haiku":    {PromptCostPer1K: 250000, CompletionCostPer1K: 1250000},
		"google/gemini-pro-1.5":       {PromptCostPer1K: 1250000, CompletionCostPer1K: 5000000},
		"google/gemini-flash-1.5":     {PromptCostPer1K: 75000, CompletionCostPer1K: 300000},
		"default":                     {PromptCostPer1K: 1000000, CompletionCostPer1K: 2000000},
	}
}

type TokenEstimate struct{ Prompt, Completion int }

func EstimateTokens(messages []AIMessage, maxTokens int) TokenEstimate {
	promptChars := 0
	for _, message := range messages {
		promptChars += len(message.Content) + len(message.Role) + messageFramingOverhead
	}
	prompt := int(float64(promptChars/charsPerToken) * tokenEstimationSafetyMargin)
	completion := defaultCompletionTokens
	if maxTokens > 0 {
		completion = maxTokens
	} else {
		completion = int(float64(completion) * tokenEstimationSafetyMargin)
	}
	return TokenEstimate{Prompt: prompt, Completion: completion}
}

func CalculateCost(pricing map[string]ModelPricing, model string, promptTokens, completionTokens int) int64 {
	modelPricing, ok := pricing[model]
	if !ok {
		modelPricing = pricing["default"]
	}
	return int64(promptTokens)*modelPricing.PromptCostPer1K/1000 + int64(completionTokens)*modelPricing.CompletionCostPer1K/1000
}
