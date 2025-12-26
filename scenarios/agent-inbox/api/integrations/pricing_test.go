package integrations

import (
	"testing"

	"agent-inbox/domain"
)

func TestGetModelPricing(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"anthropic/claude-3.5-sonnet", true},
		{"openai/gpt-4o", true},
		{"unknown/model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricing := GetModelPricing(tt.model)
			if tt.expected && pricing == nil {
				t.Errorf("expected pricing for %s, got nil", tt.model)
			}
			if !tt.expected && pricing != nil {
				t.Errorf("expected nil for %s, got pricing", tt.model)
			}
		})
	}
}

func TestCalculateUsageCost(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		promptTokens     int
		completionTokens int
		expectNonZero    bool
	}{
		{
			name:             "claude-3.5-sonnet with tokens",
			model:            "anthropic/claude-3.5-sonnet",
			promptTokens:     1000,
			completionTokens: 500,
			expectNonZero:    true,
		},
		{
			name:             "unknown model",
			model:            "unknown/model",
			promptTokens:     1000,
			completionTokens: 500,
			expectNonZero:    false,
		},
		{
			name:             "zero tokens",
			model:            "anthropic/claude-3.5-sonnet",
			promptTokens:     0,
			completionTokens: 0,
			expectNonZero:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptCost, completionCost, totalCost := CalculateUsageCost(tt.model, tt.promptTokens, tt.completionTokens)

			if tt.expectNonZero {
				if promptCost <= 0 || completionCost <= 0 || totalCost <= 0 {
					t.Errorf("expected non-zero costs, got prompt=%f, completion=%f, total=%f", promptCost, completionCost, totalCost)
				}
				if totalCost != promptCost+completionCost {
					t.Errorf("total cost (%f) should equal prompt (%f) + completion (%f)", totalCost, promptCost, completionCost)
				}
			} else {
				if promptCost != 0 || completionCost != 0 || totalCost != 0 {
					t.Errorf("expected zero costs for %s, got prompt=%f, completion=%f, total=%f", tt.name, promptCost, completionCost, totalCost)
				}
			}
		})
	}
}

func TestCalculateUsageCost_Accuracy(t *testing.T) {
	// Claude 3.5 Sonnet: 300 cents per million prompt, 1500 cents per million completion
	// 1000 prompt tokens = 300 * 1000 / 1,000,000 = 0.3 cents
	// 500 completion tokens = 1500 * 500 / 1,000,000 = 0.75 cents
	promptCost, completionCost, _ := CalculateUsageCost("anthropic/claude-3.5-sonnet", 1000, 500)

	expectedPromptCost := 0.3
	expectedCompletionCost := 0.75

	if promptCost != expectedPromptCost {
		t.Errorf("prompt cost = %f, want %f", promptCost, expectedPromptCost)
	}
	if completionCost != expectedCompletionCost {
		t.Errorf("completion cost = %f, want %f", completionCost, expectedCompletionCost)
	}
}

func TestCreateUsageRecord(t *testing.T) {
	usage := &domain.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	record := CreateUsageRecord("chat-123", "msg-456", "anthropic/claude-3.5-sonnet", usage)

	if record == nil {
		t.Fatal("expected usage record, got nil")
	}

	if record.ChatID != "chat-123" {
		t.Errorf("ChatID = %s, want chat-123", record.ChatID)
	}
	if record.MessageID != "msg-456" {
		t.Errorf("MessageID = %s, want msg-456", record.MessageID)
	}
	if record.Model != "anthropic/claude-3.5-sonnet" {
		t.Errorf("Model = %s, want anthropic/claude-3.5-sonnet", record.Model)
	}
	if record.PromptTokens != 1000 {
		t.Errorf("PromptTokens = %d, want 1000", record.PromptTokens)
	}
	if record.CompletionTokens != 500 {
		t.Errorf("CompletionTokens = %d, want 500", record.CompletionTokens)
	}
	if record.TotalTokens != 1500 {
		t.Errorf("TotalTokens = %d, want 1500", record.TotalTokens)
	}
	if record.TotalCost <= 0 {
		t.Errorf("TotalCost = %f, want > 0", record.TotalCost)
	}
}

func TestCreateUsageRecord_NilUsage(t *testing.T) {
	record := CreateUsageRecord("chat-123", "msg-456", "anthropic/claude-3.5-sonnet", nil)

	if record != nil {
		t.Errorf("expected nil for nil usage, got %+v", record)
	}
}
