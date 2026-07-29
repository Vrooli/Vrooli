package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/intelligence"
)

// testUserContext creates a context with user claims for testing authenticated endpoints.
func testUserContext(userID, email string) context.Context {
	claims := &UserClaims{
		UserID: userID,
		Email:  email,
	}
	return context.WithValue(context.Background(), userClaimsKey, claims)
}

// newTestAIGatewayDeps creates AIGatewayDeps with a mock service for testing.
func newTestAIGatewayDeps(mockSvc *MockAIGateway) *AIGatewayDeps {
	return &AIGatewayDeps{
		Service:         mockSvc,
		UsageService:    nil, // Will be set in tests that need it
		AccountService:  nil, // Will be set in tests that need it
		UserRateLimiter: NewRateLimiter(60, time.Minute),
		IPRateLimiter:   NewRateLimiter(120, time.Minute),
		IPKeyFunc:       IPKeyFunc(),
	}
}

// MockAIGateway is a test-only implementation of the handler's gateway seam.
type MockAIGateway struct {
	ExecuteChatFn        func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error)
	ExecuteChatStreamFn  func(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error
	GetAvailableModelsFn func() []string
	HealthCheckFn        func(ctx context.Context) error
}

func (m *MockAIGateway) ExecuteChat(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
	if m.ExecuteChatFn != nil {
		return m.ExecuteChatFn(ctx, userIdentity, req)
	}
	return &AIResponse{ID: "mock-chat-id", Model: req.Model, Content: "Mock response content", PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, CreditsCharged: 1000, FinishReason: "stop"}, nil
}

func (m *MockAIGateway) ExecuteChatStream(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
	if m.ExecuteChatStreamFn != nil {
		return m.ExecuteChatStreamFn(ctx, userIdentity, req, w)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte("data: {\"type\":\"chunk\",\"content\":\"Mock\"}\n\n"))
	_, _ = w.Write([]byte("data: {\"type\":\"done\",\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":1,\"total_tokens\":11,\"credits_charged\":500}}\n\n"))
	return nil
}

func (m *MockAIGateway) GetAvailableModels() []string {
	if m.GetAvailableModelsFn != nil {
		return m.GetAvailableModelsFn()
	}
	return []string{"mock/model-1", "mock/model-2"}
}

func (m *MockAIGateway) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFn != nil {
		return m.HealthCheckFn(ctx)
	}
	return nil
}

var _ AIGateway = (*MockAIGateway)(nil)

// --- handleAIChat Tests ---

func TestHandleAIChat_Success(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model: "openai/gpt-4o",
		Messages: []AIMessage{
			{Role: "user", Content: "Hello"},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp AIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Content != "Mock response content" {
		t.Errorf("Expected mock content, got %s", resp.Content)
	}
}

func TestHandleAIChat_RequiresAuth(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	// No user context - unauthenticated

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleAIChat_InvalidBody(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", strings.NewReader("{invalid json"))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAIChat_ValidationError(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	// Missing model field
	reqBody := AIRequest{
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify error message mentions model
	if !strings.Contains(w.Body.String(), "model") {
		t.Errorf("Expected error message to mention 'model', got %s", w.Body.String())
	}
}

func TestHandleAIChat_UserRateLimitEnforced(t *testing.T) {
	mockSvc := &MockAIGateway{}
	// Create deps with a very low rate limit
	deps := &AIGatewayDeps{
		Service:         mockSvc,
		UserRateLimiter: NewRateLimiter(2, time.Minute), // Only 2 requests allowed
		IPRateLimiter:   NewRateLimiter(1000, time.Minute),
		IPKeyFunc:       IPKeyFunc(),
	}

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)

	// Make 2 requests (at limit)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
		req = req.WithContext(testUserContext("user-123", "test@example.com"))
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d", w.Code)
	}
}

func TestHandleAIChat_IPRateLimitEnforced(t *testing.T) {
	mockSvc := &MockAIGateway{}
	// Create deps with very low IP rate limit
	deps := &AIGatewayDeps{
		Service:         mockSvc,
		UserRateLimiter: NewRateLimiter(1000, time.Minute),
		IPRateLimiter:   NewRateLimiter(2, time.Minute), // Only 2 requests per IP
		IPKeyFunc:       IPKeyFunc(),
	}

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)

	// Make 2 requests from same IP (different users)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
		req = req.WithContext(testUserContext("user-"+string(rune('A'+i)), "user"+string(rune('A'+i))+"@example.com"))
		req.RemoteAddr = "10.0.0.1:12345" // Same IP
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// Third request from same IP should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-C", "userC@example.com"))
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third request from same IP should be rate limited, got status %d", w.Code)
	}
}

func TestHandleAIChat_ServiceError(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
			return nil, ErrInsufficientCredits
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("Expected status %d for insufficient credits, got %d", http.StatusPaymentRequired, w.Code)
	}
}

// --- handleAIStream Tests ---

func TestHandleAIStream_Success(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIStream(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/stream", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify SSE content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	// Verify response contains SSE data
	body2 := w.Body.String()
	if !strings.Contains(body2, "data:") {
		t.Errorf("Expected SSE data events in response, got %s", body2)
	}
}

func TestHandleAIStream_RequiresAuth(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIStream(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/stream", bytes.NewReader(body))
	// No user context

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// --- handleAIModels Tests ---

func TestHandleAIModels_Success(t *testing.T) {
	mockSvc := &MockAIGateway{
		GetAvailableModelsFn: func() []string {
			return []string{"model-a", "model-b", "model-c"}
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIModels(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	models, ok := resp["models"].([]interface{})
	if !ok {
		t.Fatal("Expected 'models' field in response")
	}

	if len(models) != 3 {
		t.Errorf("Expected 3 models, got %d", len(models))
	}
}

// --- handleAIUsage Tests ---

func TestHandleAIUsage_Success(t *testing.T) {
	db := setupTestDB(t)

	limitsService := NewLimitsService(db, "sqlite")
	usageService := NewUsageService(db, limitsService, "sqlite")

	mockSvc := &MockAIGateway{}
	deps := &AIGatewayDeps{
		Service:         mockSvc,
		UsageService:    usageService,
		AccountService:  nil, // Will default to "free" tier
		UserRateLimiter: NewRateLimiter(60, time.Minute),
		IPRateLimiter:   NewRateLimiter(120, time.Minute),
		IPKeyFunc:       IPKeyFunc(),
	}

	handler := handleAIUsage(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/usage", nil)
	req = req.WithContext(testUserContext("user-123", "test@example.com"))

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify expected fields
	if _, ok := resp["user_identity"]; !ok {
		t.Error("Expected 'user_identity' in response")
	}
	if _, ok := resp["tier"]; !ok {
		t.Error("Expected 'tier' in response")
	}
	if _, ok := resp["display"]; !ok {
		t.Error("Expected 'display' in response")
	}
}

func TestHandleAIUsage_RequiresAuth(t *testing.T) {
	mockSvc := &MockAIGateway{}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIUsage(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/usage", nil)
	// No user context

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// --- handleAIHealth Tests ---

func TestHandleAIHealth_Success(t *testing.T) {
	mockSvc := &MockAIGateway{
		HealthCheckFn: func(ctx context.Context) error {
			return nil
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIHealth(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	healthy, ok := resp["healthy"].(bool)
	if !ok || !healthy {
		t.Error("Expected healthy: true in response")
	}
}

func TestHandleAIHealth_ServiceDown(t *testing.T) {
	mockSvc := &MockAIGateway{
		HealthCheckFn: func(ctx context.Context) error {
			return errors.New("OpenRouter API unreachable")
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIHealth(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	healthy, ok := resp["healthy"].(bool)
	if !ok || healthy {
		t.Error("Expected healthy: false in response")
	}

	if _, hasError := resp["error"]; !hasError {
		t.Error("Expected error message in response")
	}
}

// --- validateAIRequest Tests ---

func TestValidateAIRequest_ValidRequest(t *testing.T) {
	req := &AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}

	if err := validateAIRequest(req); err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}
}

func TestValidateAIRequest_MissingModel(t *testing.T) {
	req := &AIRequest{
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}

	if err := validateAIRequest(req); err == nil {
		t.Error("Expected error for missing model")
	}
}

func TestValidateAIRequest_NoMessages(t *testing.T) {
	req := &AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{},
	}

	if err := validateAIRequest(req); err == nil {
		t.Error("Expected error for empty messages")
	}
}

func TestValidateAIRequest_TooManyMessages(t *testing.T) {
	messages := make([]AIMessage, 101)
	for i := range messages {
		messages[i] = AIMessage{Role: "user", Content: "msg"}
	}

	req := &AIRequest{
		Model:    "openai/gpt-4o",
		Messages: messages,
	}

	if err := validateAIRequest(req); err == nil {
		t.Error("Expected error for too many messages")
	}
}

func TestValidateAIRequest_InvalidRole(t *testing.T) {
	req := &AIRequest{
		Model: "openai/gpt-4o",
		Messages: []AIMessage{
			{Role: "invalid_role", Content: "Hello"},
		},
	}

	if err := validateAIRequest(req); err == nil {
		t.Error("Expected error for invalid role")
	}
}

func TestValidateAIRequest_InvalidMaxTokens(t *testing.T) {
	req := &AIRequest{
		Model:     "openai/gpt-4o",
		Messages:  []AIMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: -1,
	}

	if err := validateAIRequest(req); err == nil {
		t.Error("Expected error for negative max_tokens")
	}
}

func TestValidateAIRequest_MaxTokensExceedsLimit(t *testing.T) {
	req := &AIRequest{
		Model:     "openai/gpt-4o",
		Messages:  []AIMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 20000, // Exceeds maxMaxTokens (16384)
	}

	if err := validateAIRequest(req); err == nil {
		t.Error("Expected error for max_tokens exceeding limit")
	}
}

// ============================================================================
// handleAIChat Error Mapping Tests
// ============================================================================

func TestHandleAIChat_ErrNoAPIKeyConfigured_Returns503(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
			return nil, ErrNoAPIKeyConfigured
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d for ErrNoAPIKeyConfigured, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleAIChat_ErrModelNotAllowed_Returns400(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
			return nil, ErrModelNotAllowed
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "forbidden-model",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for ErrModelNotAllowed, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAIChat_ProviderError_Returns502(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
			return nil, intelligence.ErrProvider
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status %d for provider error, got %d", http.StatusBadGateway, w.Code)
	}
}

func TestHandleAIChat_UnknownError_Returns500(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatFn: func(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
			return nil, errors.New("unexpected internal error")
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIChat(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for unknown error, got %d", http.StatusInternalServerError, w.Code)
	}
}

// ============================================================================
// handleAIStream Error Mapping Tests
// ============================================================================

func TestHandleAIStream_ErrNoAPIKeyConfigured_Returns503(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatStreamFn: func(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
			return ErrNoAPIKeyConfigured
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIStream(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/stream", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d for ErrNoAPIKeyConfigured, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleAIStream_ErrModelNotAllowed_Returns400(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatStreamFn: func(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
			return ErrModelNotAllowed
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIStream(deps)

	reqBody := AIRequest{
		Model:    "forbidden-model",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/stream", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for ErrModelNotAllowed, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAIStream_ErrStreamingNotSupported_Returns501(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatStreamFn: func(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
			return ErrStreamingNotSupported
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIStream(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/stream", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected status %d for ErrStreamingNotSupported, got %d", http.StatusNotImplemented, w.Code)
	}
}

func TestHandleAIStream_ProviderError_Returns502(t *testing.T) {
	mockSvc := &MockAIGateway{
		ExecuteChatStreamFn: func(ctx context.Context, userIdentity string, req AIRequest, w http.ResponseWriter) error {
			return intelligence.ErrProvider
		},
	}
	deps := newTestAIGatewayDeps(mockSvc)

	handler := handleAIStream(deps)

	reqBody := AIRequest{
		Model:    "openai/gpt-4o",
		Messages: []AIMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/stream", bytes.NewReader(body))
	req = req.WithContext(testUserContext("user-123", "test@example.com"))
	req.RemoteAddr = "192.168.1.1:12345"

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status %d for provider error, got %d", http.StatusBadGateway, w.Code)
	}
}

// ============================================================================
// handleAIError Tests
// ============================================================================

func TestHandleAIError_ErrNoAPIKeyConfigured_Returns503(t *testing.T) {
	w := httptest.NewRecorder()
	handleAIError(w, ErrNoAPIKeyConfigured)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHandleAIError_ErrModelNotAllowed_Returns400(t *testing.T) {
	w := httptest.NewRecorder()
	handleAIError(w, ErrModelNotAllowed)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAIError_ProviderError_Returns502(t *testing.T) {
	w := httptest.NewRecorder()
	handleAIError(w, intelligence.ErrProvider)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
}

func TestHandleAIError_ErrStreamingNotSupported_Returns501(t *testing.T) {
	w := httptest.NewRecorder()
	handleAIError(w, ErrStreamingNotSupported)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected status %d, got %d", http.StatusNotImplemented, w.Code)
	}
}
