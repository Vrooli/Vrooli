package intelligence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/monetization-go"
	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// TestMeteredInferenceService_CalculateCost tests cost calculation.
func TestMeteredInferenceService_CalculateCost(t *testing.T) {
	svc := &MeteredInferenceService{
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

// TestMeteredInferenceService_EstimateTokens tests token estimation.
// The estimation includes a 1.5x safety margin to reduce underestimation.
func TestMeteredInferenceService_EstimateTokens(t *testing.T) {
	svc := &MeteredInferenceService{}

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

// TestMeteredInferenceService_EstimateTokens_SafetyMargin tests that the safety margin is applied correctly.
func TestMeteredInferenceService_EstimateTokens_SafetyMargin(t *testing.T) {
	svc := &MeteredInferenceService{}

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

// TestMeteredInferenceService_GetAvailableModels tests model listing.
func TestMeteredInferenceService_GetAvailableModels(t *testing.T) {
	svc := &MeteredInferenceService{}
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

// TestMeteredInferenceService_HealthCheck tests health check with mocked client.
func TestMeteredInferenceService_HealthCheck(t *testing.T) {
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
			svc := &MeteredInferenceService{
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

// TestMeteredInferenceService_ModelValidation tests that invalid models are rejected.
func TestMeteredInferenceService_ModelValidation(t *testing.T) {
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
				PromptTokens     int          `json:"prompt_tokens"`
				CompletionTokens int          `json:"completion_tokens"`
				TotalTokens      int          `json:"total_tokens"`
				CreditsCharged   int64        `json:"credits_charged"`
				BrowserUsage     BrowserUsage `json:"browser_usage,omitempty"`
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

type MockUsageService struct {
	ReserveAndChargeFn             func(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReport) error
	ReserveCreditsFn               func(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error)
	FinalizeReservationFn          func(ctx context.Context, reservationID string, actualAmount int64) error
	ReleaseReservationFn           func(ctx context.Context, reservationID string) error
	RetainReservationFn            func(ctx context.Context, reservationID string) error
	ReserveBusinessAccountWithIDFn func(ctx context.Context, accountID, actor, tier, limitKey string, amount int64, operationID string) (string, bool, error)
	RetainBusinessAccountFn        func(ctx context.Context, accountID, actor, reservationID string) error
	ReleaseBusinessAccountFn       func(ctx context.Context, accountID, actor, reservationID string) error
	FinalizeBusinessAccountFn      func(ctx context.Context, accountID, actor, reservationID string, amount int64) error
	AdjustUsageFn                  func(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error
	RecordUsageFn                  func(ctx context.Context, req UsageReport) error
	Calls                          map[string][]interface{}
}

func NewMockUsageService() *MockUsageService {
	return &MockUsageService{Calls: make(map[string][]interface{})}
}

func (m *MockUsageService) ReserveAndCharge(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReport) error {
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
	if m.ReserveAndChargeFn != nil {
		if err := m.ReserveAndChargeFn(ctx, userIdentity, tier, limitKey, amount, UsageReport{UserIdentity: userIdentity, LimitKey: limitKey, Amount: amount}); err != nil {
			return "", err
		}
	}
	return "test-reservation-id", nil
}

func (m *MockUsageService) ReserveBusinessAccountCredits(_ context.Context, accountID, actor, tier, limitKey string, amount int64) (string, error) {
	m.Calls["ReserveBusinessAccountCredits"] = append(m.Calls["ReserveBusinessAccountCredits"], []interface{}{accountID, actor, tier, limitKey, amount})
	return "account-test-reservation-id", nil
}

func (m *MockUsageService) ReserveBusinessAccountCreditsWithID(ctx context.Context, accountID, actor, tier, limitKey string, amount int64, operationID string) (string, bool, error) {
	m.Calls["ReserveBusinessAccountCreditsWithID"] = append(m.Calls["ReserveBusinessAccountCreditsWithID"], []interface{}{accountID, actor, tier, limitKey, amount, operationID})
	if m.ReserveBusinessAccountWithIDFn != nil {
		return m.ReserveBusinessAccountWithIDFn(ctx, accountID, actor, tier, limitKey, amount, operationID)
	}
	return "account-test-reservation-id", true, nil
}

func (m *MockUsageService) ReserveBusinessAccountInferenceCreditsWithID(ctx context.Context, accountID, actor, tier, appBundleKey string, amount int64, operationID string) (string, bool, error) {
	m.Calls["ReserveBusinessAccountInferenceCreditsWithID"] = append(m.Calls["ReserveBusinessAccountInferenceCreditsWithID"], []interface{}{accountID, actor, tier, appBundleKey, amount, operationID})
	if m.ReserveBusinessAccountWithIDFn != nil {
		return m.ReserveBusinessAccountWithIDFn(ctx, accountID, actor, tier, "ai_credits", amount, operationID)
	}
	return "account-test-reservation-id", true, nil
}

func (m *MockUsageService) BindBusinessAccountBrowserEvidence(_ context.Context, accountID, actor, reservationID, runID, fingerprint string) error {
	m.Calls["BindBusinessAccountBrowserEvidence"] = append(m.Calls["BindBusinessAccountBrowserEvidence"], []interface{}{accountID, actor, reservationID, runID, fingerprint})
	return nil
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

func (m *MockUsageService) RetainUncertainReservation(ctx context.Context, reservationID string) error {
	m.Calls["RetainUncertainReservation"] = append(m.Calls["RetainUncertainReservation"], []interface{}{reservationID})
	if m.RetainReservationFn != nil {
		return m.RetainReservationFn(ctx, reservationID)
	}
	return nil
}

func (m *MockUsageService) FinalizeBusinessAccountReservationWithCost(ctx context.Context, accountID, actor, reservationID string, amount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	m.Calls["FinalizeBusinessAccountReservationWithCost"] = append(m.Calls["FinalizeBusinessAccountReservationWithCost"], []interface{}{accountID, actor, reservationID, amount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, int64(0), int64(0), int64(0)})
	if m.FinalizeBusinessAccountFn != nil {
		return m.FinalizeBusinessAccountFn(ctx, accountID, actor, reservationID, amount)
	}
	return nil
}

func (m *MockUsageService) FinalizeBusinessAccountReservationWithBrowserUsage(ctx context.Context, accountID, actor, reservationID string, amount int64, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string, sessionMinutes, actions, bytes int64, runID, sessionID, evidenceFingerprint string) error {
	m.Calls["FinalizeBusinessAccountReservationWithCost"] = append(m.Calls["FinalizeBusinessAccountReservationWithCost"], []interface{}{accountID, actor, reservationID, amount, appBundleKey, model, costMicros, promptTokens, completionTokens, provider, sessionMinutes, actions, bytes, runID, sessionID, evidenceFingerprint})
	if m.FinalizeBusinessAccountFn != nil {
		return m.FinalizeBusinessAccountFn(ctx, accountID, actor, reservationID, amount)
	}
	return nil
}

func (m *MockUsageService) ReleaseBusinessAccountReservation(ctx context.Context, accountID, actor, reservationID string) error {
	m.Calls["ReleaseBusinessAccountReservation"] = append(m.Calls["ReleaseBusinessAccountReservation"], []interface{}{accountID, actor, reservationID})
	if m.ReleaseBusinessAccountFn != nil {
		return m.ReleaseBusinessAccountFn(ctx, accountID, actor, reservationID)
	}
	return nil
}

func (m *MockUsageService) RetainUncertainBusinessAccountReservation(ctx context.Context, accountID, actor, reservationID string) error {
	m.Calls["RetainUncertainBusinessAccountReservation"] = append(m.Calls["RetainUncertainBusinessAccountReservation"], []interface{}{accountID, actor, reservationID})
	if m.RetainBusinessAccountFn != nil {
		return m.RetainBusinessAccountFn(ctx, accountID, actor, reservationID)
	}
	return nil
}

func (m *MockUsageService) RecordBusinessAccountUsage(_ context.Context, accountID, actor, tier, limitKey string, amount int64, operationID, appBundleKey, model string, costMicros int64, promptTokens, completionTokens int, provider string) error {
	m.Calls["RecordBusinessAccountUsage"] = append(m.Calls["RecordBusinessAccountUsage"], []interface{}{accountID, actor, tier, limitKey, amount, operationID, appBundleKey, model, costMicros, promptTokens, completionTokens, provider})
	return nil
}

func (m *MockUsageService) AdjustBusinessAccountUsage(_ context.Context, accountID, limitKey string, adjustment int64) error {
	m.Calls["AdjustBusinessAccountUsage"] = append(m.Calls["AdjustBusinessAccountUsage"], []interface{}{accountID, limitKey, adjustment})
	return nil
}

func (m *MockUsageService) AdjustUsage(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error {
	m.Calls["AdjustUsage"] = append(m.Calls["AdjustUsage"], []interface{}{userIdentity, limitKey, adjustment, reason})
	if m.AdjustUsageFn != nil {
		return m.AdjustUsageFn(ctx, userIdentity, limitKey, adjustment, reason)
	}
	return nil
}

func (m *MockUsageService) RecordUsage(ctx context.Context, req UsageReport) error {
	m.Calls["RecordUsage"] = append(m.Calls["RecordUsage"], []interface{}{req})
	if m.RecordUsageFn != nil {
		return m.RecordUsageFn(ctx, req)
	}
	return nil
}

// Compile-time check
var _ UsageServicer = (*MockUsageService)(nil)

// MockAccountService implements AccountServicer for testing tier lookup.
type MockAccountService struct {
	GetSubscriptionFn         func(userIdentity string) (*landing_page_business_suite_v1.SubscriptionStatus, error)
	GetBusinessSubscriptionFn func(userIdentity, businessAccountID string) (*landing_page_business_suite_v1.SubscriptionStatus, error)
}

func (m *MockAccountService) GetSubscriptionContext(_ context.Context, userIdentity string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
	if m.GetSubscriptionFn != nil {
		return m.GetSubscriptionFn(userIdentity)
	}
	tier := "pro"
	return &landing_page_business_suite_v1.SubscriptionStatus{PlanTier: &tier}, nil
}

func (m *MockAccountService) GetSubscriptionForBusinessAccountContext(_ context.Context, userIdentity, businessAccountID string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
	if m.GetBusinessSubscriptionFn != nil {
		return m.GetBusinessSubscriptionFn(userIdentity, businessAccountID)
	}
	return m.GetSubscriptionContext(context.Background(), userIdentity)
}

// Compile-time check
var _ AccountServicer = (*MockAccountService)(nil)

func TestBillingIdentityAndTierScopesSelectedBusinessAccount(t *testing.T) {
	accounts := map[string]string{"acct-one": "pro", "acct-two": "free"}
	accountService := &MockAccountService{GetBusinessSubscriptionFn: func(_ string, id string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
		tier := accounts[id]
		return &landing_page_business_suite_v1.SubscriptionStatus{PlanTier: &tier}, nil
	}}
	svc := NewMeteredInferenceService(MeteredInferenceServiceOptions{AccountService: accountService})

	firstIdentity, firstTier, err := svc.billingIdentityAndTier(context.Background(), "buyer@example.com", "acct-one")
	if err != nil {
		t.Fatal(err)
	}
	secondIdentity, secondTier, err := svc.billingIdentityAndTier(context.Background(), "buyer@example.com", "acct-two")
	if err != nil {
		t.Fatal(err)
	}
	if firstIdentity != "buyer@example.com" || secondIdentity != "buyer@example.com" {
		t.Fatalf("actor attribution was not preserved: %q, %q", firstIdentity, secondIdentity)
	}
	if firstTier != "pro" || secondTier != "free" {
		t.Fatalf("account-specific tiers not selected: %q, %q", firstTier, secondTier)
	}
}

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
func TestBrowserUsageRejectsNegativeCounters(t *testing.T) {
	for _, usage := range []BrowserUsage{{SessionMinutes: -1}, {Actions: -1}, {Bytes: -1}} {
		if err := usage.validate(); err == nil {
			t.Fatalf("negative browser usage accepted: %+v", usage)
		}
	}
	if err := (BrowserUsage{SessionMinutes: 3, Actions: 12, Bytes: 4096}).validate(); err != nil {
		t.Fatalf("valid browser usage rejected: %v", err)
	}
}

type fixedBrowserEvidenceAttester struct {
	got      BrowserEvidenceProof
	result   AttestedBrowserUsage
	err      error
	verified int
}

type browserSessionSettlementStub struct {
	*MockUsageService
	aggregate commerce.BrowserGatewayAggregate
	finalized []interface{}
}

func (s *browserSessionSettlementStub) BusinessAccountBrowserGatewayAggregate(context.Context, string, string, string) (commerce.BrowserGatewayAggregate, error) {
	return s.aggregate, nil
}

func (*browserSessionSettlementStub) ReserveBusinessAccountBrowserSession(context.Context, string, string, string, string, int64, string) (string, bool, error) {
	return "", false, errors.New("unexpected reservation during reconciliation")
}

func (*browserSessionSettlementStub) BindBusinessAccountBrowserSession(context.Context, string, string, string, string, string, string, string) error {
	return errors.New("unexpected bind during reconciliation")
}

func (*browserSessionSettlementStub) BeginBusinessAccountBrowserGatewayReceipt(context.Context, string, string, string, string, string, string, string, string, string, string, int64) (bool, error) {
	return false, errors.New("unexpected dispatch during reconciliation")
}

func (*browserSessionSettlementStub) RecordBusinessAccountBrowserGatewayReceipt(context.Context, string, string, string, commerce.BrowserGatewayReceipt) error {
	return errors.New("unexpected receipt write during reconciliation")
}

func (*browserSessionSettlementStub) MarkBusinessAccountBrowserGatewayUncertain(context.Context, string, string, string, string) error {
	return errors.New("unexpected uncertainty update during reconciliation")
}

func (s *browserSessionSettlementStub) FinalizeBusinessAccountBrowserSession(_ context.Context, account, actor, reservation, run, session, grantFingerprint, proofFingerprint, provider string, aggregate commerce.BrowserGatewayAggregate, minutes, actions, bytes int64) error {
	s.finalized = []interface{}{account, actor, reservation, run, session, grantFingerprint, proofFingerprint, provider, aggregate, minutes, actions, bytes}
	return nil
}

type unavailableBrowserEvidenceAttester struct{ readyCalls int }

func (a *unavailableBrowserEvidenceAttester) Ready(context.Context) error {
	a.readyCalls++
	return errors.New("operator-issued LPBS service credential is absent")
}

func (*unavailableBrowserEvidenceAttester) VerifyAndConsumeBrowserEvidence(context.Context, BrowserEvidenceProof) (AttestedBrowserUsage, error) {
	return AttestedBrowserUsage{}, errors.New("unreachable: attestation must be preflighted")
}

func TestBrowserEvidenceIdentityReadinessBlocksBeforeReservationAndInference(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(map[bool]string{false: "chat", true: "stream"}[streaming], func(t *testing.T) {
			usage := NewMockUsageService()
			attester := &unavailableBrowserEvidenceAttester{}
			service := &MeteredInferenceService{usageService: usage, browserEvidenceAttester: attester}
			request := AIRequest{Model: "openai/gpt-4o-mini", Messages: []AIMessage{{Role: "user", Content: "hello"}}, BusinessAccountID: "acct-test", Metadata: AIMetadata{BrowserEvidence: &BrowserEvidenceProof{RunID: "run-1", GrantReceipt: "grant", Reference: "proof"}}}
			var err error
			if streaming {
				err = service.ExecuteChatStream(context.Background(), "actor", request, httptest.NewRecorder())
			} else {
				_, err = service.ExecuteChat(context.Background(), "actor", request)
			}
			if err == nil || !strings.Contains(err.Error(), "verifier identity is unavailable") {
				t.Fatalf("unavailable attester error=%v", err)
			}
			if attester.readyCalls != 1 || len(usage.Calls["ReserveBusinessAccountCreditsWithID"]) != 0 || len(usage.Calls["ReserveCredits"]) != 0 {
				t.Fatalf("not ready attester reached billing/provider path: ready=%d calls=%#v", attester.readyCalls, usage.Calls)
			}
		})
	}
}

func (a *fixedBrowserEvidenceAttester) VerifyAndConsumeBrowserEvidence(_ context.Context, proof BrowserEvidenceProof) (AttestedBrowserUsage, error) {
	a.got = proof
	a.verified++
	return a.result, a.err
}

func TestClosedBrowserSessionReconcilesImmutableGatewayAggregate(t *testing.T) {
	const account, actor, reservation, run, session, grant, proofRef = "acct-7", "operator@example.test", "7e026a72-90ba-4e0b-9897-b2b09dc8a750", "87026a72-90ba-4e0b-9897-b2b09dc8a750", "bas-session-7", "grant-7", "opaque-proof-7"
	aggregate := commerce.BrowserGatewayAggregate{Credits: 15, CostMicros: 820, PromptTokens: 21, CompletionTokens: 9, Operations: 2}
	usage := &browserSessionSettlementStub{MockUsageService: NewMockUsageService(), aggregate: aggregate}
	attester := &fixedBrowserEvidenceAttester{result: AttestedBrowserUsage{RunID: run, ReservationID: reservation, SessionID: session, State: "closed", SessionMinutes: 4, Actions: 12, Bytes: 4096}}
	service := &MeteredInferenceService{usageService: usage, browserEvidenceAttester: attester}
	binding := BrowserSessionBinding{ReservationID: reservation, RunID: run, SessionID: session, GrantReceipt: grant}
	proof := BrowserEvidenceProof{RunID: run, GrantReceipt: grant, Reference: proofRef}
	if err := service.reconcileClosedBrowserSessionAuthorized(context.Background(), actor, account, binding, proof); err != nil {
		t.Fatal(err)
	}
	if attester.verified != 1 || attester.got.ReservationID != reservation || attester.got.Reference != proofRef {
		t.Fatalf("attester received wrong proof binding: %+v (calls=%d)", attester.got, attester.verified)
	}
	if len(usage.finalized) != 12 {
		t.Fatalf("finalization fields = %#v", usage.finalized)
	}
	if usage.finalized[0] != account || usage.finalized[1] != actor || usage.finalized[2] != reservation || usage.finalized[3] != run || usage.finalized[4] != session {
		t.Fatalf("finalization scope = %#v", usage.finalized[:5])
	}
	if usage.finalized[6] != browserEvidenceFingerprint(proofRef) || usage.finalized[7] != "" || usage.finalized[8] != aggregate {
		t.Fatalf("settlement did not bind proof and persisted Gateway aggregate: %#v", usage.finalized[6:9])
	}
	if usage.finalized[9] != int64(4) || usage.finalized[10] != int64(12) || usage.finalized[11] != int64(4096) {
		t.Fatalf("caller/browser counters replaced attested measurements: %#v", usage.finalized[9:])
	}
}

func TestClosedBrowserSessionPublicReconciliationKeepsBillingScopeFailClosed(t *testing.T) {
	attester := &fixedBrowserEvidenceAttester{result: AttestedBrowserUsage{RunID: "87026a72-90ba-4e0b-9897-b2b09dc8a750", ReservationID: "7e026a72-90ba-4e0b-9897-b2b09dc8a750", SessionID: "bas-session-7", State: "closed"}}
	usage := &browserSessionSettlementStub{MockUsageService: NewMockUsageService(), aggregate: commerce.BrowserGatewayAggregate{Operations: 1}}
	service := &MeteredInferenceService{usageService: usage, browserEvidenceAttester: attester}
	binding := BrowserSessionBinding{ReservationID: "7e026a72-90ba-4e0b-9897-b2b09dc8a750", RunID: "87026a72-90ba-4e0b-9897-b2b09dc8a750", SessionID: "bas-session-7", GrantReceipt: "grant-7"}
	err := service.ReconcileClosedBrowserSession(context.Background(), "operator@example.test", "acct-7", binding, BrowserEvidenceProof{RunID: binding.RunID, GrantReceipt: binding.GrantReceipt, Reference: "opaque-proof-7"})
	if !errors.Is(err, ErrBillingScopeUnavailable) {
		t.Fatalf("public reconciliation error = %v, want fail-closed billing scope", err)
	}
	if attester.verified != 0 || usage.finalized != nil {
		t.Fatalf("closed proof or settlement reached without billing scope: verified=%d finalized=%#v", attester.verified, usage.finalized)
	}
}

func TestClosedBrowserSessionWithoutGatewayChildrenDoesNotConsumeProof(t *testing.T) {
	attester := &fixedBrowserEvidenceAttester{}
	usage := &browserSessionSettlementStub{MockUsageService: NewMockUsageService()}
	service := &MeteredInferenceService{usageService: usage, browserEvidenceAttester: attester}
	binding := BrowserSessionBinding{ReservationID: "7e026a72-90ba-4e0b-9897-b2b09dc8a750", RunID: "87026a72-90ba-4e0b-9897-b2b09dc8a750", SessionID: "bas-session-7", GrantReceipt: "grant-7"}
	err := service.reconcileClosedBrowserSessionAuthorized(context.Background(), "operator@example.test", "acct-7", binding, BrowserEvidenceProof{RunID: binding.RunID, GrantReceipt: binding.GrantReceipt, Reference: "opaque-proof-7"})
	if err == nil || !strings.Contains(err.Error(), "no completed Gateway operation receipt") {
		t.Fatalf("empty session aggregate error = %v", err)
	}
	if attester.verified != 0 {
		t.Fatalf("proof was consumed for a session without model operations: %d", attester.verified)
	}
}

func TestCallerReportedBrowserUsageCannotReachSettlement(t *testing.T) {
	err := validateBrowserEvidenceInput(AIMetadata{BrowserUsage: BrowserUsage{SessionMinutes: 2, Actions: 9, Bytes: 1024}})
	if err == nil || !strings.Contains(err.Error(), "caller-reported browser usage") {
		t.Fatalf("caller supplied browser measurements were not rejected: %v", err)
	}
}

func TestBrowserInferenceRequiresRunGrantProofBoundToReservation(t *testing.T) {
	if err := validateInferenceBrowserEvidence("browser.plan", AIMetadata{}); err == nil || !strings.Contains(err.Error(), "BAS run, grant and session binding") {
		t.Fatalf("browser inference without evidence error=%v, want fail-closed run/grant requirement", err)
	}
	proof := &BrowserEvidenceProof{RunID: "run-1", GrantReceipt: "grant-1", Reference: "opaque-proof-1"}
	if err := validateInferenceBrowserEvidence("browser.plan", AIMetadata{BrowserEvidence: proof}); err == nil || !strings.Contains(err.Error(), "only after provider operations finish") {
		t.Fatalf("pre-close BAS proof must not authorize inference dispatch: %v", err)
	}
	binding := &BrowserSessionBinding{RunID: "87026a72-90ba-4e0b-9897-b2b09dc8a750", SessionID: "bas-session-1", GrantReceipt: "grant-1"}
	if err := validateInferenceBrowserEvidence("browser.plan", AIMetadata{BrowserSession: binding}); err != nil {
		t.Fatalf("valid BAS open-session binding rejected: %v", err)
	}
	if err := validateInferenceBrowserEvidence("classify.fast", AIMetadata{}); err != nil {
		t.Fatalf("non-browser inference unexpectedly requires browser evidence: %v", err)
	}
}

func TestBrowserInferenceRequiresBoundedGatewayRootOperation(t *testing.T) {
	if err := validateInferenceRootOperationID("browser.plan", ""); err == nil || !strings.Contains(err.Error(), "requires gateway_root_operation_id") {
		t.Fatalf("browser inference without recovery identity error=%v", err)
	}
	if err := validateInferenceRootOperationID("browser.plan", "short"); err == nil || !strings.Contains(err.Error(), "bounded") {
		t.Fatalf("malformed browser root operation accepted: %v", err)
	}
	root := strings.Repeat("d", 64)
	if err := validateInferenceRootOperationID("browser.plan", root); err != nil {
		t.Fatalf("valid browser root operation rejected: %v", err)
	}
	if err := validateInferenceRootOperationID("chat.default", ""); err != nil {
		t.Fatalf("legacy non-browser request unexpectedly requires a root: %v", err)
	}
}

func TestBASProofResolutionRejectsWrongRunOrUnclosedSession(t *testing.T) {
	proof := &BrowserEvidenceProof{RunID: "run-17", GrantReceipt: "grant-exact", Reference: "opaque-proof"}
	for _, observation := range []AttestedBrowserUsage{
		{RunID: "run-other", ReservationID: "reservation-1", SessionID: "session-9", State: "closed", Actions: 2},
		{RunID: "run-17", ReservationID: "reservation-1", SessionID: "session-9", State: "open", Actions: 2},
	} {
		service := &MeteredInferenceService{browserEvidenceAttester: &fixedBrowserEvidenceAttester{result: observation}}
		if _, err := service.resolveBrowserUsage(context.Background(), proof, "reservation-1"); err == nil {
			t.Fatalf("unbound BAS observation accepted: %+v", observation)
		}
	}
}

func TestBASProofUsesExistingIdempotentLPBSReserveAndRejectsReplayBeforeDispatch(t *testing.T) {
	usage := NewMockUsageService()
	service := &MeteredInferenceService{usageService: usage}
	proof := &BrowserEvidenceProof{RunID: "run-17", GrantReceipt: "grant-exact", Reference: "opaque-proof"}
	rootOperationID := strings.Repeat("a", 64)
	operationID := browserEvidenceOperationID(rootOperationID)
	if len(operationID) != 36 || operationID[14] != '5' || !strings.ContainsRune("89ab", rune(operationID[19])) {
		t.Fatalf("proof-derived operation id is not a PostgreSQL UUID: %q", operationID)
	}
	first, err := service.reserveCredits(context.Background(), "buyer@example.com", "buyer@example.com", "acct-fixture", "pro", "fixture-app", 50, proof, rootOperationID)
	if err != nil || first != "account-test-reservation-id" {
		t.Fatalf("first browser reservation id=%q err=%v", first, err)
	}
	calls := usage.Calls["ReserveBusinessAccountInferenceCreditsWithID"]
	if len(calls) != 1 || calls[0].([]interface{})[3] != "fixture-app" || calls[0].([]interface{})[5] != operationID {
		t.Fatalf("browser reservation did not use stable proof idempotency: %#v", calls)
	}
	usage.ReserveBusinessAccountWithIDFn = func(context.Context, string, string, string, string, int64, string) (string, bool, error) {
		return "existing-reservation", false, nil
	}
	if _, err := service.reserveCredits(context.Background(), "buyer@example.com", "buyer@example.com", "acct-fixture", "pro", "fixture-app", 50, proof, rootOperationID); !errors.Is(err, ErrBrowserEvidenceReplay) {
		t.Fatalf("replayed browser proof reserve error=%v, want replay denial", err)
	}
}

func TestGatewayRootOperationReplayRefusedBeforeSecondInferenceDispatch(t *testing.T) {
	usage := NewMockUsageService()
	service := &MeteredInferenceService{usageService: usage}
	rootOperationID := strings.Repeat("b", 64)
	first, err := service.reserveCredits(context.Background(), "buyer@example.com", "buyer@example.com", "acct-fixture", "pro", "fixture-app", 50, nil, rootOperationID)
	if err != nil || first != "account-test-reservation-id" {
		t.Fatalf("first root-bound reservation id=%q err=%v", first, err)
	}
	operationID := browserEvidenceOperationID(rootOperationID)
	calls := usage.Calls["ReserveBusinessAccountInferenceCreditsWithID"]
	if len(calls) != 1 || calls[0].([]interface{})[3] != "fixture-app" || calls[0].([]interface{})[5] != operationID {
		t.Fatalf("Gateway root operation was not used as LPBS idempotency input: %#v", calls)
	}
	usage.ReserveBusinessAccountWithIDFn = func(context.Context, string, string, string, string, int64, string) (string, bool, error) {
		return "existing-reservation", false, nil
	}
	if _, err := service.reserveCredits(context.Background(), "buyer@example.com", "buyer@example.com", "acct-fixture", "pro", "fixture-app", 50, nil, rootOperationID); !errors.Is(err, ErrInferenceOperationReplay) {
		t.Fatalf("replayed Gateway root operation error=%v, want dispatch refusal", err)
	}
}

func TestClosedBASProofBindsBrowserMeasurementsToTokenSettlement(t *testing.T) {
	usage := NewMockUsageService()
	attester := &fixedBrowserEvidenceAttester{
		result: AttestedBrowserUsage{RunID: "run-17", ReservationID: "reservation-1", SessionID: "session-9", State: "closed", SessionMinutes: 4, Actions: 13, Bytes: 8192},
	}
	service := &MeteredInferenceService{usageService: usage, browserEvidenceAttester: attester}
	proof := &BrowserEvidenceProof{RunID: "run-17", GrantReceipt: "grant-exact", Reference: "opaque-proof"}
	if err := validateBrowserEvidenceInput(AIMetadata{BrowserEvidence: proof}); err != nil {
		t.Fatal(err)
	}
	if err := service.bindBrowserEvidence(context.Background(), "buyer@example.com", "acct-fixture", "reservation-1", proof); err != nil {
		t.Fatal(err)
	}
	bindings := usage.Calls["BindBusinessAccountBrowserEvidence"]
	if len(bindings) != 1 || bindings[0].([]interface{})[2] != "reservation-1" || bindings[0].([]interface{})[3] != "run-17" {
		t.Fatalf("proof was not bound to the exact LPBS reservation and run: %#v", bindings)
	}
	browserUsage, err := service.resolveBrowserUsage(context.Background(), proof, "reservation-1")
	if err != nil {
		t.Fatal(err)
	}
	boundProof := *proof
	boundProof.ReservationID = "reservation-1"
	if attester.verified != 1 || attester.got != boundProof {
		t.Fatalf("attester request=%+v calls=%d", attester.got, attester.verified)
	}
	proofDigest := sha256.Sum256([]byte(proof.Reference))
	wantBrowserUsage := BrowserUsage{SessionMinutes: 4, Actions: 13, Bytes: 8192, RunID: "run-17", SessionID: "session-9", EvidenceFingerprint: hex.EncodeToString(proofDigest[:])}
	if browserUsage != wantBrowserUsage {
		t.Fatalf("attested usage=%+v", browserUsage)
	}
	if err := service.finalizeReservationWithCost(context.Background(), "buyer@example.com", "acct-fixture", "reservation-1", 123, "browser-automation-studio", "openai/gpt-4o-mini", 77, 21, 8, "openrouter", browserUsage); err != nil {
		t.Fatal(err)
	}
	settlements := usage.Calls["FinalizeBusinessAccountReservationWithCost"]
	if len(settlements) != 1 {
		t.Fatalf("settlements=%d, want 1", len(settlements))
	}
	settlement := settlements[0].([]interface{})
	if settlement[5] != "openai/gpt-4o-mini" || settlement[6] != int64(77) || settlement[7] != 21 || settlement[8] != 8 || settlement[9] != "openrouter" || settlement[10] != int64(4) || settlement[11] != int64(13) || settlement[12] != int64(8192) || settlement[13] != "run-17" || settlement[14] != "session-9" || settlement[15] != hex.EncodeToString(proofDigest[:]) {
		t.Fatalf("single LPBS receipt did not reconcile model/token/browser evidence: %#v", settlement)
	}
}

func TestFinalizeReservationWithBrowserUsageUsesAccountReceiptAdapter(t *testing.T) {
	usage := NewMockUsageService()
	service := &MeteredInferenceService{usageService: usage}
	err := service.finalizeReservationWithCost(context.Background(), "actor@example.com", "acct-1", "reservation-1", 23, "browser-automation-studio", "provider/model", 17, 5, 3, "provider", BrowserUsage{SessionMinutes: 7, Actions: 11, Bytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	calls := usage.Calls["FinalizeBusinessAccountReservationWithCost"]
	if len(calls) != 1 {
		t.Fatalf("account receipt adapter calls=%d, want 1", len(calls))
	}
	got := calls[0].([]interface{})
	if got[0] != "acct-1" || got[1] != "actor@example.com" || got[2] != "reservation-1" || got[10] != int64(7) || got[11] != int64(11) || got[12] != int64(2048) {
		t.Fatalf("account receipt adapter args=%#v", got)
	}
}

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

	svc := &MeteredInferenceService{
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
	if len(mockUsage.Calls["ReserveCredits"]) == 0 {
		t.Error("expected ReserveCredits to be called")
	}
}

// TestExecuteChat_InsufficientCredits tests credit check failure.
func TestExecuteChat_InsufficientCredits(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.ReserveAndChargeFn = func(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReport) error {
		return ErrInsufficientCredits
	}

	svc := &MeteredInferenceService{
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
	svc := &MeteredInferenceService{
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

	svc := &MeteredInferenceService{
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

	if len(mockUsage.Calls["FinalizeReservation"]) == 0 {
		t.Error("expected FinalizeReservation to be called for settlement")
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

	svc := &MeteredInferenceService{
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

	// Settlement records the measured overage against the held reservation.
	if len(mockUsage.Calls["FinalizeReservation"]) == 0 {
		t.Error("expected FinalizeReservation to be called for settlement")
	}
}

// TestExecuteChat_TierLookupFails_DefaultsFree tests fallback to free tier.
func TestExecuteChat_TierLookupFails_DefaultsFree(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockAccount := &MockAccountService{
		GetSubscriptionFn: func(userIdentity string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
			return nil, errors.New("subscription service unavailable")
		},
	}
	mockOpenRouter := &MockOpenRouterClient{}

	svc := &MeteredInferenceService{
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

	// Verify "free" tier was used in the reservation request.
	if len(mockUsage.Calls["ReserveCredits"]) == 0 {
		t.Fatal("expected ReserveCredits to be called")
	}
	args := mockUsage.Calls["ReserveCredits"][0].([]interface{})
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

	svc := &MeteredInferenceService{
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
	accountService := &MockAccountService{GetBusinessSubscriptionFn: func(_ string, accountID string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
		if accountID != "acct-123" {
			t.Fatalf("subscription lookup account=%q, want acct-123", accountID)
		}
		tier := "business-pro"
		return &landing_page_business_suite_v1.SubscriptionStatus{PlanTier: &tier}, nil
	}}

	svc := &MeteredInferenceService{
		usageService:     mockUsage,
		accountService:   accountService,
		openRouterClient: mockOpenRouter,
		modelPricing:     defaultModelPricing(),
		log:              func(event string, fields map[string]interface{}) {},
	}

	w := NewMockFlusherResponseWriter()

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model:             "openai/gpt-4o-mini",
		BusinessAccountID: "acct-123",
		Messages:          []AIMessage{{Role: "user", Content: "Hello"}},
	}, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Error("expected text/event-stream content type")
	}

	if len(mockUsage.Calls["ReserveBusinessAccountCredits"]) == 0 {
		t.Error("expected account-scoped ReserveBusinessAccountCredits to be called")
	} else {
		reservation := mockUsage.Calls["ReserveBusinessAccountCredits"][0].([]interface{})
		if reservation[0] != "acct-123" || reservation[1] != "user@test.com" || reservation[2] != "business-pro" {
			t.Fatalf("reservation did not preserve account, actor, and tier: %+v", reservation)
		}
	}

	if len(mockUsage.Calls["FinalizeBusinessAccountReservationWithCost"]) == 0 {
		t.Error("expected account-scoped measured settlement")
	}
}

// TestExecuteChatStream_InsufficientCredits tests reservation failure.
func TestExecuteChatStream_InsufficientCredits(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.ReserveCreditsFn = func(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
		return "", ErrInsufficientCredits
	}

	svc := &MeteredInferenceService{
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
			return &OpenRouterUsage{CompletionTokens: 3, TotalTokens: 3}, errors.New("stream failed")
		},
	}

	svc := &MeteredInferenceService{
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
	if len(mockUsage.Calls["FinalizeReservation"]) != 0 {
		t.Error("must not charge when provider reports no measured usage")
	}
}

func TestExecuteChatStream_StreamFailsWithReportedUsage_SettlesPartialUsage(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{
		ChatStreamFn: func(ctx context.Context, req OpenRouterChatRequest, onChunk func(string)) (*OpenRouterUsage, error) {
			onChunk("partial")
			return &OpenRouterUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, UsageReported: true}, errors.New("stream interrupted")
		},
	}
	svc := &MeteredInferenceService{
		usageService: mockUsage, accountService: &MockAccountService{}, openRouterClient: mockOpenRouter,
		modelPricing: defaultModelPricing(), log: func(string, map[string]interface{}) {},
	}

	err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model: "openai/gpt-4o-mini", Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, NewMockFlusherResponseWriter())
	if err == nil {
		t.Fatal("expected the interrupted stream error")
	}
	if got := len(mockUsage.Calls["FinalizeReservation"]); got != 1 {
		t.Fatalf("FinalizeReservation calls = %d, want 1", got)
	}
	wantCost := CalculateCost(defaultModelPricing(), "openai/gpt-4o-mini", 100, 20)
	finalizeCall := mockUsage.Calls["FinalizeReservation"][0].([]interface{})
	if got := finalizeCall[1].(int64); got != wantCost {
		t.Fatalf("settled credits = %d, want measured token charge %d", got, wantCost)
	}
	if got := len(mockUsage.Calls["ReleaseReservation"]); got != 0 {
		t.Fatalf("ReleaseReservation calls = %d, want 0 after measured usage settlement", got)
	}
}

func TestExecuteChatStream_BusinessAccountChargesMeasuredPartialUsage(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockOpenRouter := &MockOpenRouterClient{
		ChatStreamFn: func(_ context.Context, _ OpenRouterChatRequest, onChunk func(string)) (*OpenRouterUsage, error) {
			onChunk("partial")
			return &OpenRouterUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, CostMicros: 1234, UsageReported: true}, errors.New("stream interrupted")
		},
	}
	svc := &MeteredInferenceService{
		usageService: mockUsage, accountService: &MockAccountService{}, openRouterClient: mockOpenRouter,
		modelPricing: defaultModelPricing(), log: func(string, map[string]interface{}) {},
	}

	err := svc.ExecuteChatStream(context.Background(), "buyer@example.com", AIRequest{
		Model: "openai/gpt-4o-mini", BusinessAccountID: "acct-fixture",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, NewMockFlusherResponseWriter())
	if err == nil {
		t.Fatal("expected the interrupted stream error")
	}
	reserve := mockUsage.Calls["ReserveBusinessAccountCreditsWithID"]
	if len(reserve) != 1 || reserve[0].([]interface{})[0] != "acct-fixture" || reserve[0].([]interface{})[1] != "buyer@example.com" {
		t.Fatalf("business-account reservation attribution = %#v", reserve)
	}
	settlements := mockUsage.Calls["FinalizeBusinessAccountReservationWithCost"]
	if len(settlements) != 1 {
		t.Fatalf("account measured settlements = %d, want 1", len(settlements))
	}
	settlement := settlements[0].([]interface{})
	wantCost := CalculateCost(defaultModelPricing(), "openai/gpt-4o-mini", 100, 20)
	if settlement[0] != "acct-fixture" || settlement[1] != "buyer@example.com" || settlement[3] != wantCost || settlement[6] != int64(1234) || settlement[7] != 100 || settlement[8] != 20 || settlement[10] != int64(0) || settlement[11] != int64(0) || settlement[12] != int64(0) {
		t.Fatalf("account partial settlement = %#v, want token usage %d credits and no unattested browser usage", settlement, wantCost)
	}
	if len(mockUsage.Calls["ReleaseBusinessAccountReservation"]) != 0 {
		t.Fatal("measured partial output must not release the account reservation")
	}
}

func TestExecuteInferenceStream_CodeRoleForwardsToolsAndSettlesAccountUsage(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	mockUsage := NewMockUsageService()
	toolSpec := json.RawMessage(`{"type":"function","function":{"name":"read_file","parameters":{"type":"object","properties":{"path":{"type":"string"}}}}}`)
	provider := &MockOpenRouterClient{
		ChatStreamEventsFn: func(_ context.Context, request OpenRouterChatRequest, emit func(OpenRouterStreamEvent)) (*OpenRouterUsage, error) {
			if request.Model != "openai/gpt-4o-mini" || len(request.Tools) != 1 || string(request.Tools[0]) != string(toolSpec) {
				t.Fatalf("provider request = %#v, want role-resolved model and tool schema", request)
			}
			if len(request.Messages) != 1 || request.Messages[0].Role != "user" || request.Messages[0].Content != "inspect src/main.go" {
				t.Fatalf("provider messages = %#v", request.Messages)
			}
			emit(OpenRouterStreamEvent{ToolCalls: []OpenRouterToolCallDelta{{Index: 0, ID: "call_1", Type: "function", Function: OpenRouterToolFunctionDelta{Name: "read_file", Arguments: `{"path":"src/main.go"}`}}}})
			return &OpenRouterUsage{PromptTokens: 25, CompletionTokens: 8, TotalTokens: 33, UsageReported: true}, nil
		},
	}
	svc := &MeteredInferenceService{
		usageService: mockUsage, accountService: &MockAccountService{}, openRouterClient: provider,
		modelPricing: defaultModelPricing(), rolePolicies: DefaultRolePolicies(), creditFeatureOwners: owners, log: func(string, map[string]interface{}) {},
	}
	writer := NewMockFlusherResponseWriter()
	err = svc.ExecuteInferenceStreamForCaller(context.Background(), "actor@example.com", "web-console", InferenceRequest{
		Role: "author.generator", BusinessAccountID: "acct-fixture", Messages: []AIMessage{{Role: "user", Content: "inspect src/main.go"}},
		Tools: []json.RawMessage{toolSpec}, ToolChoice: json.RawMessage(`"auto"`), Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("a", 32)},
	}, writer)
	if err != nil {
		t.Fatalf("ExecuteInferenceStream() error: %v", err)
	}
	if !strings.Contains(writer.Body.String(), `"tool_calls":[{"index":0,"id":"call_1"`) {
		t.Fatalf("SSE did not preserve tool-call delta: %s", writer.Body.String())
	}
	reserve := mockUsage.Calls["ReserveBusinessAccountInferenceCreditsWithID"]
	if len(reserve) != 1 || reserve[0].([]interface{})[0] != "acct-fixture" || reserve[0].([]interface{})[1] != "actor@example.com" || reserve[0].([]interface{})[3] != "web-console" {
		t.Fatalf("feature-scoped reservation attribution = %#v", reserve)
	}
	if got := len(mockUsage.Calls["FinalizeBusinessAccountReservationWithCost"]); got != 1 {
		t.Fatalf("account settlements = %d, want 1 after measured tool-call stream", got)
	}
}

func TestExecuteInference_CodeRoleForwardsToolHistoryAndReturnsToolCall(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	mockUsage := NewMockUsageService()
	toolSpec := json.RawMessage(`{"type":"function","function":{"name":"read_file","parameters":{"type":"object"}}}`)
	provider := &MockOpenRouterClient{
		ChatFn: func(_ context.Context, request OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
			if len(request.Messages) != 2 || request.Messages[1].Role != "tool" || request.Messages[1].ToolCallID != "call_1" || request.Messages[1].Name != "read_file" {
				t.Fatalf("tool conversation = %#v", request.Messages)
			}
			if len(request.Tools) != 1 || string(request.Tools[0]) != string(toolSpec) {
				t.Fatalf("tools = %s", request.Tools)
			}
			return &OpenRouterChatResponse{
				ID: "chatcmpl-fixture", Model: request.Model,
				ToolCalls: []OpenRouterToolCall{{ID: "call_2", Type: "function", Function: OpenRouterToolFunction{Name: "read_file", Arguments: `{"path":"go.mod"}`}}},
				Usage:     OpenRouterUsage{PromptTokens: 32, CompletionTokens: 6, TotalTokens: 38},
			}, nil
		},
	}
	svc := &MeteredInferenceService{
		usageService: mockUsage, accountService: &MockAccountService{}, openRouterClient: provider,
		modelPricing: defaultModelPricing(), rolePolicies: DefaultRolePolicies(), creditFeatureOwners: owners, log: func(string, map[string]interface{}) {},
	}
	response, err := svc.ExecuteInferenceForCaller(context.Background(), "actor@example.com", "web-console", InferenceRequest{
		Role: "author.generator", BusinessAccountID: "acct-fixture", Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("b", 32)},
		Messages: []AIMessage{{Role: "assistant", ToolCalls: []OpenRouterToolCall{{ID: "call_1", Type: "function", Function: OpenRouterToolFunction{Name: "read_file", Arguments: `{"path":"src/main.go"}`}}}}, {Role: "tool", Content: "package main", Name: "read_file", ToolCallID: "call_1"}},
		Tools:    []json.RawMessage{toolSpec}, ToolChoice: json.RawMessage(`"auto"`),
	})
	if err != nil {
		t.Fatalf("ExecuteInference() error: %v", err)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call_2" {
		t.Fatalf("response tool calls = %#v", response.ToolCalls)
	}
	reservations := mockUsage.Calls["ReserveBusinessAccountInferenceCreditsWithID"]
	if len(reservations) != 1 || reservations[0].([]interface{})[3] != "web-console" {
		t.Fatalf("feature-scoped reservation = %#v, want app web-console", reservations)
	}
	if len(mockUsage.Calls["FinalizeBusinessAccountReservationWithCost"]) != 1 {
		t.Fatalf("account settlements = %d, want 1", len(mockUsage.Calls["FinalizeBusinessAccountReservationWithCost"]))
	}
	settlement := mockUsage.Calls["FinalizeBusinessAccountReservationWithCost"][0].([]interface{})
	if settlement[4] != "web-console" {
		t.Fatalf("settlement app = %#v, want generated owner web-console", settlement[4])
	}
}

func TestExecuteInferenceForCallerRejectsOwnerMismatchesBeforeEffects(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	service := NewMeteredInferenceService(MeteredInferenceServiceOptions{CreditFeatureOwners: owners})
	cases := []struct {
		name, caller, role string
	}{
		{name: "wrong scenario", caller: "browser-automation-studio", role: "author.generator"},
		{name: "wrong role for caller", caller: "web-console", role: "write.default"},
		{name: "unowned role", caller: "web-console", role: "code.default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.ExecuteInferenceForCaller(context.Background(), "actor@example.test", tc.caller, InferenceRequest{
				Role: tc.role, BusinessAccountID: "acct", Messages: []AIMessage{{Role: "user", Content: "safe synthetic request"}},
				Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("c", 32), AppBundleKey: "caller-controlled"},
			})
			if !errors.Is(err, ErrBillingScopeUnavailable) {
				t.Fatalf("ExecuteInferenceForCaller error = %v, want closed owner mismatch", err)
			}
		})
	}
}

func TestExecuteInferenceForCallerRefusesRepeatedOperationBeforeProvider(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	usage := NewMockUsageService()
	created := false
	usage.ReserveBusinessAccountWithIDFn = func(_ context.Context, accountID, actor, tier, limitKey string, amount int64, operationID string) (string, bool, error) {
		if created {
			return "account-test-reservation-id", false, nil
		}
		created = true
		return "account-test-reservation-id", true, nil
	}
	providerCalls := 0
	provider := &MockOpenRouterClient{ChatFn: func(_ context.Context, request OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
		providerCalls++
		return &OpenRouterChatResponse{ID: "synthetic", Model: request.Model, Content: "ok", Usage: OpenRouterUsage{PromptTokens: 2, CompletionTokens: 1}}, nil
	}}
	service := NewMeteredInferenceService(MeteredInferenceServiceOptions{UsageService: usage, AccountService: &MockAccountService{}, OpenRouterClient: provider, CreditFeatureOwners: owners})
	req := InferenceRequest{Role: "author.generator", BusinessAccountID: "acct", Messages: []AIMessage{{Role: "user", Content: "synthetic"}}, Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("d", 32)}}
	if _, err := service.ExecuteInferenceForCaller(context.Background(), "actor@example.test", "web-console", req); err != nil {
		t.Fatalf("first operation: %v", err)
	}
	if _, err := service.ExecuteInferenceForCaller(context.Background(), "actor@example.test", "web-console", req); !errors.Is(err, ErrInferenceOperationReplay) {
		t.Fatalf("replay error = %v, want %v", err, ErrInferenceOperationReplay)
	}
	if providerCalls != 1 {
		t.Fatalf("provider calls = %d, want exactly one", providerCalls)
	}
}

func TestExecuteInferenceForCallerRetainsAmbiguousProviderReservation(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	for _, tc := range []struct {
		name               string
		retainErr          error
		providerErr        error
		releaseErr         error
		wantRetentionError bool
		wantUncertain      bool
		wantCleanupError   bool
	}{
		{name: "retained after timeout", providerErr: context.DeadlineExceeded, wantUncertain: true},
		{name: "retention failure surfaced", providerErr: context.DeadlineExceeded, retainErr: errors.New("synthetic database unavailable"), wantRetentionError: true, wantUncertain: true},
		{name: "explicit provider rejection releases", providerErr: errors.Join(ErrProvider, ErrProviderRequestRejected)},
		{name: "release failure surfaced", providerErr: errors.Join(ErrProvider, ErrProviderRequestRejected), releaseErr: errors.New("synthetic database unavailable"), wantCleanupError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := NewMockUsageService()
			cleanupContextIndependent := false
			releaseContextIndependent := false
			usage.RetainBusinessAccountFn = func(ctx context.Context, account, actor, reservation string) error {
				cleanupContextIndependent = ctx.Err() == nil && account == "acct" && actor == "actor@example.test" && reservation == "account-test-reservation-id"
				return tc.retainErr
			}
			usage.ReleaseBusinessAccountFn = func(ctx context.Context, _, _, _ string) error {
				releaseContextIndependent = ctx.Err() == nil
				return tc.releaseErr
			}
			providerCalls := 0
			provider := &MockOpenRouterClient{ChatFn: func(context.Context, OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
				providerCalls++
				return nil, tc.providerErr
			}}
			service := NewMeteredInferenceService(MeteredInferenceServiceOptions{UsageService: usage, AccountService: &MockAccountService{}, OpenRouterClient: provider, CreditFeatureOwners: owners})
			_, err := service.ExecuteInferenceForCaller(context.Background(), "actor@example.test", "web-console", InferenceRequest{
				Role: "author.generator", BusinessAccountID: "acct", Messages: []AIMessage{{Role: "user", Content: "synthetic ambiguous request"}},
				Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("e", 32)},
			})
			if errors.Is(err, ErrInferenceOutcomeUncertain) != tc.wantUncertain || errors.Is(err, ErrProviderRequestRejected) != (tc.providerErr != nil && errors.Is(tc.providerErr, ErrProviderRequestRejected)) {
				t.Fatalf("error=%v wantUncertain=%t providerRejected=%t", err, tc.wantUncertain, errors.Is(tc.providerErr, ErrProviderRequestRejected))
			}
			if errors.Is(err, ErrInferenceReservationRetentionFailed) != tc.wantRetentionError {
				t.Fatalf("retention error classification=%v, want %v", errors.Is(err, ErrInferenceReservationRetentionFailed), tc.wantRetentionError)
			}
			if errors.Is(err, ErrInferenceReservationCleanupFailed) != tc.wantCleanupError {
				t.Fatalf("cleanup error classification=%v, want %v", errors.Is(err, ErrInferenceReservationCleanupFailed), tc.wantCleanupError)
			}
			if providerCalls != 1 || (tc.wantUncertain && !cleanupContextIndependent) || (tc.providerErr != nil && errors.Is(tc.providerErr, ErrProviderRequestRejected) && !releaseContextIndependent) {
				t.Fatalf("providerCalls=%d retainIndependent=%t releaseIndependent=%t", providerCalls, cleanupContextIndependent, releaseContextIndependent)
			}
			wantRetains, wantReleases := 0, 0
			if tc.wantUncertain {
				wantRetains = 1
			} else {
				wantReleases = 1
			}
			if len(usage.Calls["RetainUncertainBusinessAccountReservation"]) != wantRetains || len(usage.Calls["ReleaseBusinessAccountReservation"]) != wantReleases {
				t.Fatalf("retain=%#v release=%#v", usage.Calls["RetainUncertainBusinessAccountReservation"], usage.Calls["ReleaseBusinessAccountReservation"])
			}
		})
	}
}

func TestExecuteInferenceStreamRetainsAmbiguousReservationWithoutUsage(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	usage := NewMockUsageService()
	cleanupIndependent := false
	usage.RetainBusinessAccountFn = func(ctx context.Context, _, _, _ string) error {
		cleanupIndependent = ctx.Err() == nil
		return nil
	}
	providerCalls := 0
	provider := &MockOpenRouterClient{ChatStreamEventsFn: func(context.Context, OpenRouterChatRequest, func(OpenRouterStreamEvent)) (*OpenRouterUsage, error) {
		providerCalls++
		return nil, context.Canceled
	}}
	service := NewMeteredInferenceService(MeteredInferenceServiceOptions{UsageService: usage, AccountService: &MockAccountService{}, OpenRouterClient: provider, CreditFeatureOwners: owners})
	err = service.ExecuteInferenceStreamForCaller(context.Background(), "actor@example.test", "web-console", InferenceRequest{
		Role: "author.generator", BusinessAccountID: "acct", Messages: []AIMessage{{Role: "user", Content: "synthetic stream"}},
		Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("f", 32)},
	}, NewMockFlusherResponseWriter())
	if !errors.Is(err, ErrInferenceOutcomeUncertain) || providerCalls != 1 || !cleanupIndependent {
		t.Fatalf("err=%v providerCalls=%d cleanupIndependent=%t", err, providerCalls, cleanupIndependent)
	}
	if len(usage.Calls["RetainUncertainBusinessAccountReservation"]) != 1 || len(usage.Calls["ReleaseBusinessAccountReservation"]) != 0 {
		t.Fatalf("retain=%#v release=%#v", usage.Calls["RetainUncertainBusinessAccountReservation"], usage.Calls["ReleaseBusinessAccountReservation"])
	}
}

func TestExecuteInferenceStreamSurfacesPartialSettlementAndRetentionFailure(t *testing.T) {
	owners, err := monetization.EmbeddedCreditFeatureRegistry()
	if err != nil {
		t.Fatalf("EmbeddedCreditFeatureRegistry: %v", err)
	}
	usage := NewMockUsageService()
	usage.FinalizeBusinessAccountFn = func(context.Context, string, string, string, int64) error {
		return errors.New("synthetic settlement failure")
	}
	retentionContextIndependent := false
	usage.RetainBusinessAccountFn = func(ctx context.Context, _, _, _ string) error {
		retentionContextIndependent = ctx.Err() == nil
		return errors.New("synthetic retention failure")
	}
	provider := &MockOpenRouterClient{ChatStreamEventsFn: func(context.Context, OpenRouterChatRequest, func(OpenRouterStreamEvent)) (*OpenRouterUsage, error) {
		return &OpenRouterUsage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12, UsageReported: true}, context.Canceled
	}}
	service := NewMeteredInferenceService(MeteredInferenceServiceOptions{UsageService: usage, AccountService: &MockAccountService{}, OpenRouterClient: provider, CreditFeatureOwners: owners})
	err = service.ExecuteInferenceStreamForCaller(context.Background(), "actor@example.test", "web-console", InferenceRequest{
		Role: "author.generator", BusinessAccountID: "acct", Messages: []AIMessage{{Role: "user", Content: "synthetic partial stream"}},
		Metadata: AIMetadata{GatewayRootOperationID: strings.Repeat("a", 32)},
	}, NewMockFlusherResponseWriter())
	if !errors.Is(err, ErrInferenceOutcomeUncertain) || !errors.Is(err, ErrInferenceReservationRetentionFailed) || !retentionContextIndependent {
		t.Fatalf("error=%v retentionContextIndependent=%t", err, retentionContextIndependent)
	}
	if len(usage.Calls["FinalizeBusinessAccountReservationWithCost"]) != 1 || len(usage.Calls["RetainUncertainBusinessAccountReservation"]) != 1 || len(usage.Calls["ReleaseBusinessAccountReservation"]) != 0 {
		t.Fatalf("finalize=%#v retain=%#v release=%#v", usage.Calls["FinalizeBusinessAccountReservationWithCost"], usage.Calls["RetainUncertainBusinessAccountReservation"], usage.Calls["ReleaseBusinessAccountReservation"])
	}
}

func TestExecuteChatStream_PartialSettlementFailureRetainsReservation(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.FinalizeReservationFn = func(context.Context, string, int64) error { return errors.New("settlement unavailable") }
	mockOpenRouter := &MockOpenRouterClient{
		ChatStreamFn: func(context.Context, OpenRouterChatRequest, func(string)) (*OpenRouterUsage, error) {
			return &OpenRouterUsage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12, UsageReported: true}, errors.New("stream interrupted")
		},
	}
	svc := &MeteredInferenceService{
		usageService: mockUsage, accountService: &MockAccountService{}, openRouterClient: mockOpenRouter,
		modelPricing: defaultModelPricing(), log: func(string, map[string]interface{}) {},
	}

	if err := svc.ExecuteChatStream(context.Background(), "user@test.com", AIRequest{
		Model: "openai/gpt-4o-mini", Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}, NewMockFlusherResponseWriter()); err == nil {
		t.Fatal("expected the interrupted stream error")
	}
	if got := len(mockUsage.Calls["RetainUncertainReservation"]); got != 1 {
		t.Fatalf("RetainUncertainReservation calls = %d, want 1", got)
	}
	if got := len(mockUsage.Calls["RecordUsage"]); got != 0 {
		t.Fatalf("RecordUsage calls = %d, want 0 to preserve reservation idempotency", got)
	}
	if got := len(mockUsage.Calls["ReleaseReservation"]); got != 0 {
		t.Fatalf("ReleaseReservation calls = %d, want 0 while settlement is unresolved", got)
	}
}

// TestExecuteChatStream_FinalizeFails_FallbackRecords tests fallback to RecordUsage.
func TestExecuteChatStream_FinalizeFails_FallbackRecords(t *testing.T) {
	mockUsage := NewMockUsageService()
	mockUsage.FinalizeReservationFn = func(ctx context.Context, reservationID string, actualAmount int64) error {
		return errors.New("finalize failed")
	}
	mockOpenRouter := &MockOpenRouterClient{}

	svc := &MeteredInferenceService{
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

	svc := &MeteredInferenceService{
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
		GetSubscriptionFn: func(userIdentity string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
			return &landing_page_business_suite_v1.SubscriptionStatus{PlanTier: &tier}, nil
		},
	}

	svc := &MeteredInferenceService{
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
		GetSubscriptionFn: func(userIdentity string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
			return nil, nil
		},
	}

	svc := &MeteredInferenceService{
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
		GetSubscriptionFn: func(userIdentity string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
			return nil, errors.New("database error")
		},
	}

	svc := &MeteredInferenceService{
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
	svc := &MeteredInferenceService{
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

	svc := &MeteredInferenceService{
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

	svc := &MeteredInferenceService{
		apiKeyService:    mockAPIKey,
		openRouterClient: nil, // No injected client
		log:              func(event string, fields map[string]interface{}) {},
		clientFactory: func(string, func(string, map[string]interface{})) OpenRouterClient {
			return &MockOpenRouterClient{}
		},
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

	svc := &MeteredInferenceService{
		apiKeyService:    mockAPIKey,
		openRouterClient: nil,
		log:              func(event string, fields map[string]interface{}) {},
	}

	_, err := svc.getOpenRouterClient(context.Background())

	if !errors.Is(err, ErrNoAPIKeyConfigured) {
		t.Errorf("expected ErrNoAPIKeyConfigured, got: %v", err)
	}
}

func TestCreditAdmissionFailsClosedBeforeAnyBillingOrProviderSideEffect(t *testing.T) {
	for _, account := range []string{"", "acct-selected"} {
		for _, route := range []string{"chat", "chat-stream", "inference", "inference-stream"} {
			t.Run(route+"/"+account, func(t *testing.T) {
				// Nil dependencies intentionally panic if admission reaches any owner.
				svc := NewMeteredInferenceService(MeteredInferenceServiceOptions{})
				req := AIRequest{BusinessAccountID: account, Model: "openai/gpt-4o", Metadata: AIMetadata{AppBundleKey: "caller-controlled"}}
				inference := InferenceRequest{BusinessAccountID: account, Role: "code.default", Metadata: req.Metadata}
				w := httptest.NewRecorder()
				var err error
				switch route {
				case "chat":
					_, err = svc.ExecuteChat(context.Background(), "actor@example.test", req)
				case "chat-stream":
					err = svc.ExecuteChatStream(context.Background(), "actor@example.test", req, w)
				case "inference":
					_, err = svc.ExecuteInference(context.Background(), "actor@example.test", inference)
				case "inference-stream":
					err = svc.ExecuteInferenceStream(context.Background(), "actor@example.test", inference, w)
				}
				want := ErrBillingScopeUnavailable
				if account == "" {
					want = ErrBillingAccountRequired
				}
				if !errors.Is(err, want) {
					t.Fatalf("error=%v want=%v", err, want)
				}
				if w.Body.Len() != 0 || len(w.Header()) != 0 {
					t.Fatalf("stream setup before admission: %v %s", w.Header(), w.Body)
				}
			})
		}
	}
}
