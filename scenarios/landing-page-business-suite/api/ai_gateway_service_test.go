package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
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
// The estimation includes a 1.5x safety margin to reduce underestimation.
func TestAIGatewayService_EstimateTokens(t *testing.T) {
	svc := &AIGatewayService{}

	tests := []struct {
		name             string
		messages         []AIMessage
		maxTokens        int
		expectPromptMin  int
		expectPromptMax  int
		expectCompletion int
	}{
		{
			name: "simple message with safety margin",
			messages: []AIMessage{
				// ~11 chars + 4 role + 10 overhead = 25, /4 = ~6, *1.5 = ~9
				{Role: "user", Content: "Hello world"},
			},
			maxTokens:        0,
			expectPromptMin:  7,
			expectPromptMax:  15,
			expectCompletion: 1500, // Default 1000 * 1.5 safety margin
		},
		{
			name: "with max_tokens (no margin applied to user-specified max)",
			messages: []AIMessage{
				{Role: "user", Content: "Hello"},
			},
			maxTokens:        500,
			expectPromptMin:  4,
			expectPromptMax:  15,
			expectCompletion: 500, // User-specified, no safety margin
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
			expectPromptMin:  45,
			expectPromptMax:  90,
			expectCompletion: 200, // User-specified, no safety margin
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

// TestAIGatewayService_EstimateTokens_SafetyMargin tests that the safety margin is applied correctly.
func TestAIGatewayService_EstimateTokens_SafetyMargin(t *testing.T) {
	svc := &AIGatewayService{}

	// Test that without max_tokens, we get 1.5x the base estimate
	messages := []AIMessage{
		{Role: "user", Content: "Test message with known length"},
	}

	// Calculate raw estimate: "Test message with known length" = 30 chars
	// + "user" = 4 chars + 10 overhead = 44 chars / 4 = 11 tokens raw
	// With 1.5x margin: ~16 tokens
	estimate := svc.estimateTokens(messages, 0)

	// Verify prompt tokens have safety margin applied (should be ~1.5x raw)
	rawPromptTokens := (30 + 4 + 10) / 4 // = 11
	expectedMin := int(float64(rawPromptTokens) * 1.4)
	expectedMax := int(float64(rawPromptTokens) * 1.6)

	if estimate.prompt < expectedMin || estimate.prompt > expectedMax {
		t.Errorf("expected prompt tokens with safety margin between %d and %d, got %d",
			expectedMin, expectedMax, estimate.prompt)
	}

	// Verify completion tokens have safety margin (1000 * 1.5 = 1500)
	if estimate.completion != 1500 {
		t.Errorf("expected completion tokens with safety margin to be 1500, got %d", estimate.completion)
	}

	// Verify that when max_tokens is specified, no margin is applied to completion
	estimateWithMax := svc.estimateTokens(messages, 800)
	if estimateWithMax.completion != 800 {
		t.Errorf("expected user-specified max_tokens 800 to be used directly, got %d", estimateWithMax.completion)
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

// TestIPRateLimiter_IndependentFromUserLimiter tests that the IP rate limiter
// and user rate limiter are independent - they track different keys and have
// different limits.
func TestIPRateLimiter_IndependentFromUserLimiter(t *testing.T) {
	// Reset the rate limiters to ensure clean state
	// (package-level vars persist between tests)
	userLimiter := NewRateLimiter(60, 1*time.Minute)
	ipLimiter := NewRateLimiter(120, 1*time.Minute)

	// User limiter tracks by email, IP limiter tracks by IP
	userKey := "user@example.com"
	ipKey := "192.168.1.1"

	// Verify limits are different
	t.Run("different limits", func(t *testing.T) {
		// Use up user limit
		for i := 0; i < 60; i++ {
			if !userLimiter.Allow(userKey) {
				t.Errorf("user request %d should be allowed", i+1)
			}
		}

		// 61st user request should be blocked
		if userLimiter.Allow(userKey) {
			t.Error("61st user request should be blocked")
		}

		// IP limiter should still allow requests (has 120 limit)
		for i := 0; i < 60; i++ {
			if !ipLimiter.Allow(ipKey) {
				t.Errorf("IP request %d should be allowed", i+1)
			}
		}

		// IP limiter should still have 60 more
		if ipLimiter.Remaining(ipKey) != 60 {
			t.Errorf("expected 60 remaining for IP, got %d", ipLimiter.Remaining(ipKey))
		}
	})

	t.Run("different key spaces", func(t *testing.T) {
		userLimiter2 := NewRateLimiter(5, 1*time.Minute)
		ipLimiter2 := NewRateLimiter(5, 1*time.Minute)

		// Using same string as key should work independently
		key := "shared-key"

		// Use up all on user limiter
		for i := 0; i < 5; i++ {
			userLimiter2.Allow(key)
		}

		// IP limiter with same key should still have full limit
		if ipLimiter2.Remaining(key) != 5 {
			t.Errorf("IP limiter should have full limit, got %d", ipLimiter2.Remaining(key))
		}
	})
}

// MockUsageService implements UsageServicer for testing credit flows.
type MockUsageService struct {
	ReserveAndChargeFn    func(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReportRequest) error
	ReserveCreditsFn      func(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error)
	FinalizeReservationFn func(ctx context.Context, reservationID string, actualAmount int64) error
	ReleaseReservationFn  func(ctx context.Context, reservationID string) error
	AdjustUsageFn         func(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error
	RecordUsageFn         func(ctx context.Context, req UsageReportRequest) error
	GetUsageSummaryFn     func(ctx context.Context, userIdentity, tier string) (*UsageSummary, error)
	Calls                 map[string][]interface{}
}

func NewMockUsageService() *MockUsageService {
	return &MockUsageService{Calls: make(map[string][]interface{})}
}

func (m *MockUsageService) ReserveAndCharge(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReportRequest) error {
	m.Calls["ReserveAndCharge"] = append(m.Calls["ReserveAndCharge"], []interface{}{userIdentity, tier, limitKey, amount, metadata})
	if m.ReserveAndChargeFn != nil {
		return m.ReserveAndChargeFn(ctx, userIdentity, tier, limitKey, amount, metadata)
	}
	return nil
}

func (m *MockUsageService) ReserveCredits(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
	m.Calls["ReserveCredits"] = append(m.Calls["ReserveCredits"], []interface{}{userIdentity, tier, limitKey, amount})
	if m.ReserveCreditsFn != nil {
		return m.ReserveCreditsFn(ctx, userIdentity, tier, limitKey, amount)
	}
	return "test-reservation-id", nil
}

func (m *MockUsageService) FinalizeReservation(ctx context.Context, reservationID string, actualAmount int64) error {
	m.Calls["FinalizeReservation"] = append(m.Calls["FinalizeReservation"], []interface{}{reservationID, actualAmount})
	if m.FinalizeReservationFn != nil {
		return m.FinalizeReservationFn(ctx, reservationID, actualAmount)
	}
	return nil
}

func (m *MockUsageService) ReleaseReservation(ctx context.Context, reservationID string) error {
	m.Calls["ReleaseReservation"] = append(m.Calls["ReleaseReservation"], []interface{}{reservationID})
	if m.ReleaseReservationFn != nil {
		return m.ReleaseReservationFn(ctx, reservationID)
	}
	return nil
}

func (m *MockUsageService) AdjustUsage(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error {
	m.Calls["AdjustUsage"] = append(m.Calls["AdjustUsage"], []interface{}{userIdentity, limitKey, adjustment, reason})
	if m.AdjustUsageFn != nil {
		return m.AdjustUsageFn(ctx, userIdentity, limitKey, adjustment, reason)
	}
	return nil
}

func (m *MockUsageService) RecordUsage(ctx context.Context, req UsageReportRequest) error {
	m.Calls["RecordUsage"] = append(m.Calls["RecordUsage"], []interface{}{req})
	if m.RecordUsageFn != nil {
		return m.RecordUsageFn(ctx, req)
	}
	return nil
}

func (m *MockUsageService) GetUsageSummary(ctx context.Context, userIdentity, tier string) (*UsageSummary, error) {
	m.Calls["GetUsageSummary"] = append(m.Calls["GetUsageSummary"], []interface{}{userIdentity, tier})
	if m.GetUsageSummaryFn != nil {
		return m.GetUsageSummaryFn(ctx, userIdentity, tier)
	}
	return &UsageSummary{}, nil
}

// Compile-time check
var _ UsageServicer = (*MockUsageService)(nil)

// MockAccountService implements AccountServicer for testing tier lookup.
type MockAccountService struct {
	GetSubscriptionFn func(userIdentity string) (*SubscriptionStatus, error)
}

type SubscriptionStatus = landing_page_business_suite_v1.SubscriptionStatus

func (m *MockAccountService) GetSubscriptionContext(_ context.Context, userIdentity string) (*SubscriptionStatus, error) {
	if m.GetSubscriptionFn != nil {
		return m.GetSubscriptionFn(userIdentity)
	}
	tier := "pro"
	return &SubscriptionStatus{PlanTier: &tier}, nil
}

// Compile-time check
var _ AccountServicer = (*MockAccountService)(nil)

// MockAPIKeyService implements APIKeyServicer for testing API key retrieval.
type MockAPIKeyService struct {
	GetFn func(ctx context.Context, provider string) (string, error)
}

func (m *MockAPIKeyService) Get(ctx context.Context, provider string) (string, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, provider)
	}
	return "test-api-key", nil
}

// Compile-time check
var _ APIKeyServicer = (*MockAPIKeyService)(nil)

// MockFlusherResponseWriter wraps httptest.ResponseRecorder with Flush capability.
type MockFlusherResponseWriter struct {
	*httptest.ResponseRecorder
	FlushCalled int
}

func NewMockFlusherResponseWriter() *MockFlusherResponseWriter {
	return &MockFlusherResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (m *MockFlusherResponseWriter) Flush() {
	m.FlushCalled++
}

// TestExecuteChat_Success tests a successful non-streaming chat request.
func TestExecuteChat_Success(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockAccount := &MockAccountService{}
	mockOpenRouter := &MockOpenRouterClient{
		ChatFn: func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
			return &OpenRouterChatResponse{
				ID:      "test-id",
				Model:   req.Model,
				Content: "Test response",
				Usage: OpenRouterUsage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			}, nil
		},
	}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   mockAccount,
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	resp, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Test response" {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if len(mockUsage.Calls["ReserveAndCharge"]) == 0 {
		t.Error("expected ReserveAndCharge to be called")
	}
}

// TestExecuteChat_InsufficientCredits tests credit check failure.
func TestExecuteChat_InsufficientCredits(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.ReserveAndChargeFn = func(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReportRequest) error {
		return ErrInsufficientCredits
	}

	svc := &AIGatewayService{
		usageService:   mockUsage,
		accountService: &MockAccountService{},
		modelPricing:   defaultModelPricing(),
		log:            func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})

	if !errors.Is(err, ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits, got: %v", err)
	}
}

// TestExecuteChat_ModelNotAllowed tests rejection of invalid models.
func TestExecuteChat_ModelNotAllowed(t *testing.T) {
	svc := &AIGatewayService{
		log: func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "unknown/invalid-model",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})

	if !errors.Is(err, ErrModelNotAllowed) {
		t.Errorf("expected ErrModelNotAllowed, got: %v", err)
	}
}

// TestExecuteChat_CostUnderEstimate_Refunds tests that a refund is issued when actual cost is less than estimate.
func TestExecuteChat_CostUnderEstimate_Refunds(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{
		ChatFn: func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
			// Return very small usage to trigger refund
			return &OpenRouterChatResponse{
				ID:      "test-id",
				Model:   req.Model,
				Content: "Short",
				Usage: OpenRouterUsage{
					PromptTokens:     5,
					CompletionTokens: 2,
					TotalTokens:      7,
				},
			}, nil
		},
	}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   &MockAccountService{},
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called AdjustUsage with negative amount (refund)
	if len(mockUsage.Calls["AdjustUsage"]) == 0 {
		t.Error("expected AdjustUsage to be called for refund")
	} else {
		args := mockUsage.Calls["AdjustUsage"][0].([]interface{})
		adjustment := args[2].(int64)
		if adjustment >= 0 {
			t.Errorf("expected negative adjustment for refund, got: %d", adjustment)
		}
	}
}

// TestExecuteChat_CostOverEstimate_Charges tests that additional charge is made when actual cost exceeds estimate.
func TestExecuteChat_CostOverEstimate_Charges(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{
		ChatFn: func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
			// Return very large usage to trigger additional charge
			return &OpenRouterChatResponse{
				ID:      "test-id",
				Model:   req.Model,
				Content: "Very long response",
				Usage: OpenRouterUsage{
					PromptTokens:     500,
					CompletionTokens: 5000, // Much more than estimated
					TotalTokens:      5500,
				},
			}, nil
		},
	}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   &MockAccountService{},
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called RecordUsage for additional charge
	if len(mockUsage.Calls["RecordUsage"]) == 0 {
		t.Error("expected RecordUsage to be called for additional charge")
	}
}

// TestExecuteChat_TierLookupFails_DefaultsFree tests fallback to free tier.
func TestExecuteChat_TierLookupFails_DefaultsFree(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockAccount := &MockAccountService{
		GetSubscriptionFn: func(userIdentity string) (*SubscriptionStatus, error) {
			return nil, errors.New("subscription service unavailable")
		},
	}
	mockOpenRouter := &MockOpenRouterClient{}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   mockAccount,
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify "free" tier was used in ReserveAndCharge
	if len(mockUsage.Calls["ReserveAndCharge"]) == 0 {
		t.Fatal("expected ReserveAndCharge to be called")
	}
	args := mockUsage.Calls["ReserveAndCharge"][0].([]interface{})
	tier := args[1].(string)
	if tier != "free" {
		t.Errorf("expected tier 'free', got: %s", tier)
	}
}

// TestExecuteChat_OpenRouterFails tests error propagation from OpenRouter.
func TestExecuteChat_OpenRouterFails(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{
		ChatFn: func(ctx context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
			return nil, errors.New("OpenRouter API error")
		},
	}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   &MockAccountService{},
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.ExecuteChat(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "openrouter") {
		t.Errorf("expected OpenRouter error, got: %v", err)
	}
}

// TestExecuteChatStream_Success tests successful streaming request.
func TestExecuteChatStream_Success(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   &MockAccountService{},
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	w := NewMockFlusherResponseWriter()

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Error("expected text/event-stream content type")
	}

	if len(mockUsage.Calls["ReserveCredits"]) == 0 {
		t.Error("expected ReserveCredits to be called")
	}

	if len(mockUsage.Calls["FinalizeReservation"]) == 0 {
		t.Error("expected FinalizeReservation to be called")
	}
}

// TestExecuteChatStream_InsufficientCredits tests reservation failure.
func TestExecuteChatStream_InsufficientCredits(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.ReserveCreditsFn = func(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
		return "", ErrInsufficientCredits
	}

	svc := &AIGatewayService{
		usageService:   mockUsage,
		accountService: &MockAccountService{},
		modelPricing:   defaultModelPricing(),
		log:            func(event string, fields map[string]interface{}) {},
	}

	w := NewMockFlusherResponseWriter()

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, w)

	if !errors.Is(err, ErrInsufficientCredits) {
		t.Errorf("expected ErrInsufficientCredits, got: %v", err)
	}
}

// TestExecuteChatStream_StreamFails_ReleasesReservation tests cleanup on stream failure.
func TestExecuteChatStream_StreamFails_ReleasesReservation(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{
		ChatStreamFn: func(ctx context.Context, req OpenRouterChatRequest, onChunk func(string)) (*OpenRouterUsage, error) {
			return nil, errors.New("stream failed")
		},
	}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   &MockAccountService{},
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	w := NewMockFlusherResponseWriter()

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, w)

	if err == nil {
		t.Fatal("expected error")
	}

	// Should have released the reservation
	if len(mockUsage.Calls["ReleaseReservation"]) == 0 {
		t.Error("expected ReleaseReservation to be called on failure")
	}
}

// TestExecuteChatStream_FinalizeFails_FallbackRecords tests fallback to RecordUsage.
func TestExecuteChatStream_FinalizeFails_FallbackRecords(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.FinalizeReservationFn = func(ctx context.Context, reservationID string, actualAmount int64) error {
		return errors.New("finalize failed")
	}
	mockOpenRouter := &MockOpenRouterClient{}

	svc := &AIGatewayService{
		usageService:     mockUsage,
		accountService:   &MockAccountService{},
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	w := NewMockFlusherResponseWriter()

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have fallen back to RecordUsage
	if len(mockUsage.Calls["RecordUsage"]) == 0 {
		t.Error("expected RecordUsage to be called as fallback")
	}
}

// nonFlusherWriter is a ResponseWriter that doesn't implement http.Flusher.
type nonFlusherWriter struct {
	headers    http.Header
	statusCode int
	body       []byte
}

func newNonFlusherWriter() *nonFlusherWriter {
	return &nonFlusherWriter{headers: make(http.Header)}
}

func (w *nonFlusherWriter) Header() http.Header {
	return w.headers
}

func (w *nonFlusherWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *nonFlusherWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// NOTE: Deliberately not implementing http.Flusher

// TestExecuteChatStream_NotFlusher_ReturnsError tests error when ResponseWriter doesn't support flushing.
func TestExecuteChatStream_NotFlusher_ReturnsError(t *testing.T) {
	mockUsage := NewMockUsageService()

	svc := &AIGatewayService{
		usageService:   mockUsage,
		accountService: &MockAccountService{},
		modelPricing:   defaultModelPricing(),
		log:            func(event string, fields map[string]interface{}) {},
	}

	// Use custom writer that doesn't implement Flusher
	w := newNonFlusherWriter()

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model:    "openai/gpt-4o-mini",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, w)

	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Errorf("expected ErrStreamingNotSupported, got: %v", err)
	}

	// Should have released the reservation
	if len(mockUsage.Calls["ReleaseReservation"]) == 0 {
		t.Error("expected ReleaseReservation to be called")
	}
}

// TestGetUserTier_ValidSubscription tests tier extraction from subscription.
func TestGetUserTier_ValidSubscription(t *testing.T) {
	tier := "enterprise"
	mockAccount := &MockAccountService{
		GetSubscriptionFn: func(userIdentity string) (*SubscriptionStatus, error) {
			return &SubscriptionStatus{PlanTier: &tier}, nil
		},
	}

	svc := &AIGatewayService{
		accountService: mockAccount,
		log:            func(event string, fields map[string]interface{}) {},
	}

	result, err := svc.getUserTier(context.Background(), "user@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "enterprise" {
		t.Errorf("expected tier 'enterprise', got: %s", result)
	}
}

// TestGetUserTier_NilSubscription_DefaultsFree tests fallback when subscription is nil.
func TestGetUserTier_NilSubscription_DefaultsFree(t *testing.T) {
	mockAccount := &MockAccountService{
		GetSubscriptionFn: func(userIdentity string) (*SubscriptionStatus, error) {
			return nil, nil
		},
	}

	svc := &AIGatewayService{
		accountService: mockAccount,
		log:            func(event string, fields map[string]interface{}) {},
	}

	result, err := svc.getUserTier(context.Background(), "user@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "free" {
		t.Errorf("expected tier 'free', got: %s", result)
	}
}

// TestGetUserTier_ErrorLogsWarning tests that errors are logged but don't fail.
func TestGetUserTier_ErrorLogsWarning(t *testing.T) {
	var loggedEvents []string
	mockAccount := &MockAccountService{
		GetSubscriptionFn: func(userIdentity string) (*SubscriptionStatus, error) {
			return nil, errors.New("database error")
		},
	}

	svc := &AIGatewayService{
		accountService: mockAccount,
		log: func(event string, fields map[string]interface{}) {
			loggedEvents = append(loggedEvents, event)
		},
	}

	result, err := svc.getUserTier(context.Background(), "user@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "free" {
		t.Errorf("expected tier 'free', got: %s", result)
	}

	// Verify warning was logged
	found := false
	for _, event := range loggedEvents {
		if event == "tier_lookup_failed_defaulting_to_free" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning to be logged")
	}
}

// TestGetUserTier_NoAccountService tests behavior when account service is nil.
func TestGetUserTier_NoAccountService(t *testing.T) {
	var loggedEvents []string
	svc := &AIGatewayService{
		accountService: nil,
		log: func(event string, fields map[string]interface{}) {
			loggedEvents = append(loggedEvents, event)
		},
	}

	result, err := svc.getUserTier(context.Background(), "user@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "free" {
		t.Errorf("expected tier 'free', got: %s", result)
	}
}

// TestGetOpenRouterClient_Injected tests using injected client.
func TestGetOpenRouterClient_Injected(t *testing.T) {
	injectedClient := &MockOpenRouterClient{}

	svc := &AIGatewayService{
		openRouterClient: injectedClient,
		log:              func(event string, fields map[string]interface{}) {},
	}

	client, err := svc.getOpenRouterClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != injectedClient {
		t.Error("expected injected client to be returned")
	}
}

// TestGetOpenRouterClient_CreatesNew tests creating a new client from API key.
func TestGetOpenRouterClient_CreatesNew(t *testing.T) {
	mockAPIKey := &MockAPIKeyService{
		GetFn: func(ctx context.Context, provider string) (string, error) {
			return "test-api-key", nil
		},
	}

	svc := &AIGatewayService{
		apiKeyService:    mockAPIKey,
		openRouterClient: nil, // No injected client
		log:              func(event string, fields map[string]interface{}) {},
	}

	client, err := svc.getOpenRouterClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Error("expected client to be created")
	}
}

// TestGetOpenRouterClient_NoKey tests error when no API key is configured.
func TestGetOpenRouterClient_NoKey(t *testing.T) {
	mockAPIKey := &MockAPIKeyService{
		GetFn: func(ctx context.Context, provider string) (string, error) {
			return "", nil // Empty key
		},
	}

	svc := &AIGatewayService{
		apiKeyService:    mockAPIKey,
		openRouterClient: nil,
		log:              func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.getOpenRouterClient(context.Background())

	if !errors.Is(err, ErrNoAPIKeyConfigured) {
		t.Errorf("expected ErrNoAPIKeyConfigured, got: %v", err)
	}
}
