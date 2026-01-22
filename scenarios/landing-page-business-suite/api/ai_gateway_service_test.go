package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
)

// TestAIGatewayService_CalculateCost tests cost calculation.
func TestAIGatewayService_CalculateCost(t *testing.T) {
	svc := &AIGatewayService{
		modelPricing: defaultModelPricing(),
	}

	tests := []struct {
		name             string
		model            string
		promptTokens     int
		completionTokens int
		expectCost       int64
	}{
		{
			name:             "gpt-4o-mini standard request",
			model:            "openai/gpt-4o-mini",
			promptTokens:     1000,
			completionTokens: 500,
			// prompt: 1000 * 150000 / 1000 = 150000
			// completion: 500 * 600000 / 1000 = 300000
			// total: 450000
			expectCost: 450000,
		},
		{
			name:             "gpt-4o more expensive",
			model:            "openai/gpt-4o",
			promptTokens:     1000,
			completionTokens: 500,
			// prompt: 1000 * 2500000 / 1000 = 2500000
			// completion: 500 * 10000000 / 1000 = 5000000
			// total: 7500000
			expectCost: 7500000,
		},
		{
			name:             "unknown model uses default pricing",
			model:            "unknown/model",
			promptTokens:     1000,
			completionTokens: 500,
			// prompt: 1000 * 1000000 / 1000 = 1000000
			// completion: 500 * 2000000 / 1000 = 1000000
			// total: 2000000
			expectCost: 2000000,
		},
		{
			name:             "claude-3.5-sonnet pricing",
			model:            "anthropic/claude-3.5-sonnet",
			promptTokens:     1000,
			completionTokens: 500,
			// prompt: 1000 * 3000000 / 1000 = 3000000
			// completion: 500 * 15000000 / 1000 = 7500000
			// total: 10500000
			expectCost: 10500000,
		},
		{
			name:             "gemini-flash cheap model",
			model:            "google/gemini-flash-1.5",
			promptTokens:     1000,
			completionTokens: 500,
			// prompt: 1000 * 75000 / 1000 = 75000
			// completion: 500 * 300000 / 1000 = 150000
			// total: 225000
			expectCost: 225000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := svc.calculateCost(tt.model, tt.promptTokens, tt.completionTokens)
			if cost != tt.expectCost {
				t.Errorf("expected cost %d, got %d", tt.expectCost, cost)
			}
		})
	}
}

// TestAIGatewayService_EstimateTokens tests token estimation.
func TestAIGatewayService_EstimateTokens(t *testing.T) {
	svc := &AIGatewayService{}

	tests := []struct {
		name               string
		messages           []AIMessage
		maxTokens          int
		expectPromptMin    int
		expectPromptMax    int
		expectCompletion   int
	}{
		{
			name: "simple message",
			messages: []AIMessage{
				{Role: "user", Content: "Hello world"}, // ~11 chars + 4 role + 10 overhead = 25, /4 = ~6
			},
			maxTokens:        0,
			expectPromptMin:  5,
			expectPromptMax:  10,
			expectCompletion: 1000, // Default
		},
		{
			name: "with max_tokens",
			messages: []AIMessage{
				{Role: "user", Content: "Hello"},
			},
			maxTokens:        500,
			expectPromptMin:  3,
			expectPromptMax:  10,
			expectCompletion: 500, // Uses max_tokens
		},
		{
			name: "longer conversation",
			messages: []AIMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "What is the capital of France?"},
				{Role: "assistant", Content: "The capital of France is Paris."},
				{Role: "user", Content: "What about Germany?"},
			},
			maxTokens:        200,
			expectPromptMin:  30,
			expectPromptMax:  60,
			expectCompletion: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate := svc.estimateTokens(tt.messages, tt.maxTokens)

			if estimate.prompt < tt.expectPromptMin || estimate.prompt > tt.expectPromptMax {
				t.Errorf("expected prompt tokens between %d and %d, got %d",
					tt.expectPromptMin, tt.expectPromptMax, estimate.prompt)
			}

			if estimate.completion != tt.expectCompletion {
				t.Errorf("expected completion tokens %d, got %d", tt.expectCompletion, estimate.completion)
			}
		})
	}
}

// TestAIGatewayService_GetAvailableModels tests model listing.
func TestAIGatewayService_GetAvailableModels(t *testing.T) {
	svc := &AIGatewayService{}
	models := svc.GetAvailableModels()

	if len(models) == 0 {
		t.Error("expected at least one model")
	}

	// Check that known models are in the list
	expectedModels := []string{
		"openai/gpt-4o",
		"openai/gpt-4o-mini",
		"anthropic/claude-3.5-sonnet",
		"anthropic/claude-3-haiku",
		"google/gemini-pro-1.5",
		"google/gemini-flash-1.5",
	}

	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected model %s not found in list", expected)
		}
	}

	// Should have exactly 6 models
	if len(models) != 6 {
		t.Errorf("expected 6 models, got %d", len(models))
	}
}

// TestAIGatewayService_HealthCheck tests health check with mocked client.
func TestAIGatewayService_HealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		mockClient  *MockOpenRouterClient
		expectError bool
	}{
		{
			name: "healthy",
			mockClient: &MockOpenRouterClient{
				VerifyFn: func(ctx context.Context) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			name: "unhealthy - verification fails",
			mockClient: &MockOpenRouterClient{
				VerifyFn: func(ctx context.Context) error {
					return errors.New("key verification failed")
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &AIGatewayService{
				openRouterClient: tt.mockClient,
				log:              func(event string, fields map[string]interface{}) {},
			}

			err := svc.HealthCheck(context.Background())

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestAIGatewayService_ModelValidation tests that invalid models are rejected.
func TestAIGatewayService_ModelValidation(t *testing.T) {
	// Test the allowedModels map directly since ExecuteChat requires full service setup
	tests := []struct {
		name    string
		model   string
		allowed bool
	}{
		{"valid model openai/gpt-4o-mini", "openai/gpt-4o-mini", true},
		{"valid model openai/gpt-4o", "openai/gpt-4o", true},
		{"valid model anthropic/claude-3.5-sonnet", "anthropic/claude-3.5-sonnet", true},
		{"valid model anthropic/claude-3-haiku", "anthropic/claude-3-haiku", true},
		{"valid model google/gemini-pro-1.5", "google/gemini-pro-1.5", true},
		{"valid model google/gemini-flash-1.5", "google/gemini-flash-1.5", true},
		{"invalid model unknown/model", "unknown/model", false},
		{"empty model", "", false},
		{"similar but wrong model", "openai/gpt-4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if allowedModels[tt.model] != tt.allowed {
				t.Errorf("expected model %q allowed=%v, got %v", tt.model, tt.allowed, allowedModels[tt.model])
			}
		})
	}
}

// TestMockOpenRouterClient tests the mock client behavior.
func TestMockOpenRouterClient(t *testing.T) {
	t.Run("default chat response", func(t *testing.T) {
		mock := &MockOpenRouterClient{}
		resp, err := mock.Chat(context.Background(), OpenRouterChatRequest{
			Model: "test/model",
			Messages: []OpenRouterMessage{
				{Role: "user", Content: "Hello"},
			},
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "Mock response" {
			t.Errorf("expected 'Mock response', got %q", resp.Content)
		}
	})

	t.Run("custom chat response", func(t *testing.T) {
		mock := &MockOpenRouterClient{
			ChatFn: func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
				return &OpenRouterChatResponse{
					ID:      "custom-id",
					Content: "Custom response for " + req.Model,
				}, nil
			},
		}

		resp, err := mock.Chat(context.Background(), OpenRouterChatRequest{
			Model: "my/model",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "Custom response for my/model" {
			t.Errorf("unexpected content: %s", resp.Content)
		}
	})

	t.Run("streaming sends chunks", func(t *testing.T) {
		mock := &MockOpenRouterClient{}
		var chunks []string

		usage, err := mock.ChatStream(context.Background(), OpenRouterChatRequest{}, func(content string) {
			chunks = append(chunks, content)
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(chunks) != 3 {
			t.Errorf("expected 3 chunks, got %d", len(chunks))
		}
		if usage.CompletionTokens != 3 {
			t.Errorf("expected 3 completion tokens, got %d", usage.CompletionTokens)
		}
	})
}

// TestOpenRouterChatRequest_JSON tests JSON serialization.
func TestOpenRouterChatRequest_JSON(t *testing.T) {
	req := OpenRouterChatRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []OpenRouterMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello!"},
		},
		Stream: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed OpenRouterChatRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Model != req.Model {
		t.Errorf("model mismatch: %s != %s", parsed.Model, req.Model)
	}
	if len(parsed.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(parsed.Messages))
	}
	if !parsed.Stream {
		t.Error("stream flag should be true")
	}
}

// TestAIStreamEvent_JSON tests SSE event JSON serialization.
func TestAIStreamEvent_JSON(t *testing.T) {
	t.Run("chunk event", func(t *testing.T) {
		event := AIStreamEvent{
			Type:    "chunk",
			Content: "Hello world",
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if !containsStr(string(data), `"type":"chunk"`) {
			t.Error("missing type field")
		}
		if !containsStr(string(data), `"content":"Hello world"`) {
			t.Error("missing content field")
		}
	})

	t.Run("done event", func(t *testing.T) {
		event := AIStreamEvent{
			Type: "done",
			Usage: &struct {
				PromptTokens     int   `json:"prompt_tokens"`
				CompletionTokens int   `json:"completion_tokens"`
				TotalTokens      int   `json:"total_tokens"`
				CreditsCharged   int64 `json:"credits_charged"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
				CreditsCharged:   1000,
			},
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if !containsStr(string(data), `"type":"done"`) {
			t.Error("missing type field")
		}
		if !containsStr(string(data), `"credits_charged":1000`) {
			t.Error("missing credits_charged field")
		}
	})

	t.Run("error event", func(t *testing.T) {
		event := AIStreamEvent{
			Type:  "error",
			Error: "Something went wrong",
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if !containsStr(string(data), `"type":"error"`) {
			t.Error("missing type field")
		}
		if !containsStr(string(data), `"error":"Something went wrong"`) {
			t.Error("missing error field")
		}
	})
}

// Helper function for string contains
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Compile-time check that MockAIGateway implements AIGateway
var _ AIGateway = (*MockAIGateway)(nil)

// TestMockAIGateway tests the handler mock.
func TestMockAIGateway(t *testing.T) {
	t.Run("default ExecuteChat", func(t *testing.T) {
		mock := &MockAIGateway{}
		resp, err := mock.ExecuteChat(context.Background(), "test@example.com", AIRequest{
			Model: "test/model",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "Mock response content" {
			t.Errorf("unexpected content: %s", resp.Content)
		}
	})

	t.Run("default GetAvailableModels", func(t *testing.T) {
		mock := &MockAIGateway{}
		models := mock.GetAvailableModels()

		if len(models) != 2 {
			t.Errorf("expected 2 models, got %d", len(models))
		}
	})

	t.Run("default HealthCheck", func(t *testing.T) {
		mock := &MockAIGateway{}
		err := mock.HealthCheck(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("custom ExecuteChat", func(t *testing.T) {
		mock := &MockAIGateway{
			ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
				return &AIResponse{
					Content: "Custom: " + userIdentity,
				}, nil
			},
		}

		resp, err := mock.ExecuteChat(context.Background(), "alice@test.com", AIRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "Custom: alice@test.com" {
			t.Errorf("unexpected content: %s", resp.Content)
		}
	})

	t.Run("streaming writes SSE", func(t *testing.T) {
		mock := &MockAIGateway{}
		recorder := httptest.NewRecorder()

		err := mock.ExecuteChatStream(context.Background(), "test@example.com", AIRequest{}, recorder)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if recorder.Header().Get("Content-Type") != "text/event-stream" {
			t.Error("expected text/event-stream content type")
		}

		body := recorder.Body.String()
		if !containsStr(body, "chunk") {
			t.Error("expected chunk event in response")
		}
	})
}
