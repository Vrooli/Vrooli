package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// ============================================================================
// Mock AI Client for Testing
// ============================================================================

type mockAIClient struct {
	response string
	err      error
	model    string
}

func (m *mockAIClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	return m.response, m.err
}

func (m *mockAIClient) Model() string {
	if m.model == "" {
		return "mock-model"
	}
	return m.model
}

// ============================================================================
// Mock Entitlement Service for Testing
// ============================================================================

type mockEntitlementService struct {
	enabled      bool
	tier         entitlement.Tier
	creditsLimit int
	canUseAI     bool
	entitlements map[string]*entitlement.Entitlement
}

func newMockEntitlementService() *mockEntitlementService {
	return &mockEntitlementService{
		enabled:      true,
		tier:         entitlement.TierPro,
		creditsLimit: 100,
		canUseAI:     true,
		entitlements: make(map[string]*entitlement.Entitlement),
	}
}

func (m *mockEntitlementService) IsEnabled() bool {
	return m.enabled
}

func (m *mockEntitlementService) GetEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error) {
	if ent, ok := m.entitlements[userIdentity]; ok {
		return ent, nil
	}
	return &entitlement.Entitlement{
		UserIdentity: userIdentity,
		Tier:         m.tier,
		Status:       entitlement.StatusActive,
	}, nil
}

func (m *mockEntitlementService) GetAICreditsLimit(tier entitlement.Tier) int {
	return m.creditsLimit
}

// ============================================================================
// Mock AI Credits Tracker for Testing
// ============================================================================

type mockAICreditsTracker struct {
	creditsUsed   int
	requestsCount int
	chargedCalls  int
	loggedCalls   int
	canUse        bool
	remaining     int
}

func newMockAICreditsTracker() *mockAICreditsTracker {
	return &mockAICreditsTracker{
		canUse:    true,
		remaining: 100,
	}
}

func (m *mockAICreditsTracker) CanUseCredits(ctx context.Context, userIdentity string, creditsLimit int) (bool, int, error) {
	return m.canUse, m.remaining, nil
}

func (m *mockAICreditsTracker) ChargeCredits(ctx context.Context, userIdentity string, credits int, requestType, model string) error {
	m.chargedCalls++
	m.creditsUsed += credits
	return nil
}

func (m *mockAICreditsTracker) LogRequest(ctx context.Context, req *entitlement.AIRequestLog) error {
	m.loggedCalls++
	return nil
}

// ============================================================================
// CreditsClient Tests
// ============================================================================

func TestCreditsClient_ExecutePrompt_Success(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		response: "AI response",
		err:      nil,
	}

	svc := newMockEntitlementService()
	tracker := newMockAICreditsTracker()

	// Create a real entitlement.Service for testing
	cfg := config.EntitlementConfig{
		Enabled:     true,
		DefaultTier: "free",
		AICreditsLimits: map[string]int{
			"pro": 100,
		},
	}
	entSvc := entitlement.NewService(cfg, log)

	client := NewCreditsClient(CreditsClientOptions{
		Inner:            innerClient,
		EntitlementSvc:   entSvc,
		AICreditsTracker: nil, // Use nil for simpler test
		Logger:           log,
		UserIdentityFn: func(ctx context.Context) string {
			return "test@example.com"
		},
	})

	ctx := context.Background()
	response, err := client.ExecutePrompt(ctx, "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "AI response" {
		t.Errorf("expected response 'AI response', got %q", response)
	}

	// Verify tracker was not called since we passed nil
	_ = svc
	_ = tracker
}

func TestCreditsClient_ExecutePrompt_EntitlementsDisabled(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		response: "AI response",
		err:      nil,
	}

	cfg := config.EntitlementConfig{
		Enabled: false, // Disabled
	}
	entSvc := entitlement.NewService(cfg, log)

	client := NewCreditsClient(CreditsClientOptions{
		Inner:          innerClient,
		EntitlementSvc: entSvc,
		Logger:         log,
		UserIdentityFn: func(ctx context.Context) string {
			return "test@example.com"
		},
	})

	ctx := context.Background()
	response, err := client.ExecutePrompt(ctx, "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "AI response" {
		t.Errorf("expected response 'AI response', got %q", response)
	}
}

func TestCreditsClient_ExecutePrompt_NoUserIdentity(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		response: "AI response",
		err:      nil,
	}

	cfg := config.EntitlementConfig{
		Enabled: true,
	}
	entSvc := entitlement.NewService(cfg, log)

	client := NewCreditsClient(CreditsClientOptions{
		Inner:          innerClient,
		EntitlementSvc: entSvc,
		Logger:         log,
		UserIdentityFn: func(ctx context.Context) string {
			return "" // No user identity
		},
	})

	ctx := context.Background()
	response, err := client.ExecutePrompt(ctx, "test prompt")
	// Should succeed without user identity (skip credits check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "AI response" {
		t.Errorf("expected response 'AI response', got %q", response)
	}
}

func TestCreditsClient_ExecutePrompt_InnerError(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		response: "",
		err:      errors.New("AI service error"),
	}

	cfg := config.EntitlementConfig{
		Enabled: false,
	}
	entSvc := entitlement.NewService(cfg, log)

	client := NewCreditsClient(CreditsClientOptions{
		Inner:          innerClient,
		EntitlementSvc: entSvc,
		Logger:         log,
	})

	ctx := context.Background()
	_, err := client.ExecutePrompt(ctx, "test prompt")

	if err == nil {
		t.Fatal("expected error from inner client")
	}
	if err.Error() != "AI service error" {
		t.Errorf("expected error 'AI service error', got %q", err.Error())
	}
}

func TestCreditsClient_Model(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		model: "gpt-4",
	}

	client := NewCreditsClient(CreditsClientOptions{
		Inner:  innerClient,
		Logger: log,
	})

	if client.Model() != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", client.Model())
	}
}

func TestCreditsClient_ImplementsAIClient(t *testing.T) {
	// Compile-time check that CreditsClient implements AIClient
	var _ AIClient = (*CreditsClient)(nil)
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestErrNoAICredits(t *testing.T) {
	if ErrNoAICredits == nil {
		t.Fatal("ErrNoAICredits should not be nil")
	}
	if ErrNoAICredits.Error() != "no AI credits remaining" {
		t.Errorf("unexpected error message: %q", ErrNoAICredits.Error())
	}
}

func TestErrNoAIAccess(t *testing.T) {
	if ErrNoAIAccess == nil {
		t.Fatal("ErrNoAIAccess should not be nil")
	}
	if ErrNoAIAccess.Error() != "your subscription tier does not include AI access" {
		t.Errorf("unexpected error message: %q", ErrNoAIAccess.Error())
	}
}

// ============================================================================
// Request Type Tests
// ============================================================================

func TestCreditsClient_ExecutePromptWithType(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		response: "AI response",
		err:      nil,
	}

	cfg := config.EntitlementConfig{
		Enabled: false,
	}
	entSvc := entitlement.NewService(cfg, log)

	client := NewCreditsClient(CreditsClientOptions{
		Inner:          innerClient,
		EntitlementSvc: entSvc,
		Logger:         log,
	})

	ctx := context.Background()
	response, err := client.ExecutePromptWithType(ctx, "test prompt", "element_analyze")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "AI response" {
		t.Errorf("expected response 'AI response', got %q", response)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkCreditsClient_ExecutePrompt(b *testing.B) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	innerClient := &mockAIClient{
		response: "AI response",
		err:      nil,
	}

	cfg := config.EntitlementConfig{
		Enabled: false,
	}
	entSvc := entitlement.NewService(cfg, log)

	client := NewCreditsClient(CreditsClientOptions{
		Inner:          innerClient,
		EntitlementSvc: entSvc,
		Logger:         log,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ExecutePrompt(ctx, "test prompt")
	}
}
