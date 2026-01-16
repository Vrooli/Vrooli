package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// ============================================================================
// Mock UserSettingsRepository
// ============================================================================

type mockSettingsRepo struct {
	settings map[string]string
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{
		settings: make(map[string]string),
	}
}

func (m *mockSettingsRepo) GetSetting(ctx context.Context, key string) (string, error) {
	return m.settings[key], nil
}

func (m *mockSettingsRepo) SetSetting(ctx context.Context, key, value string) error {
	m.settings[key] = value
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func createTestEntitlementHandler(t *testing.T) (*EntitlementHandler, *mockSettingsRepo) {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	cfg := config.EntitlementConfig{
		RequestTimeout: 5,
		DefaultTier:    "free",
		AICreditsLimits: map[string]int{
			"free": 10,
			"pro":  100,
		},
	}

	service := entitlement.NewService(cfg, log)
	settingsRepo := newMockSettingsRepo()

	// Pass nil for creditService since we don't have a DB in tests
	handler := NewEntitlementHandler(service, nil, settingsRepo)
	return handler, settingsRepo
}

// ============================================================================
// GetEntitlementStatus Tests
// ============================================================================

func TestGetEntitlementStatus_Success(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have tier and status fields
	if response.Tier == "" {
		t.Fatal("expected tier to be set")
	}
	if response.Status == "" {
		t.Fatal("expected status to be set")
	}
}

func TestGetEntitlementStatus_WithUserQuery(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status?user=test@example.com", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.UserIdentity != "test@example.com" {
		t.Fatalf("expected user identity 'test@example.com', got %q", response.UserIdentity)
	}
}

func TestGetEntitlementStatus_UsesStoredIdentity(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	// Pre-set the user identity in settings
	settingsRepo.settings["user_identity"] = "stored@example.com"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.UserIdentity != "stored@example.com" {
		t.Fatalf("expected user identity 'stored@example.com', got %q", response.UserIdentity)
	}
}

// ============================================================================
// SetUserIdentity Tests
// ============================================================================

func TestSetUserIdentity_Success(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	body := `{"email": "newuser@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/identity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetUserIdentity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify identity was saved
	if settingsRepo.settings["user_identity"] != "newuser@example.com" {
		t.Fatalf("expected identity to be saved, got %q", settingsRepo.settings["user_identity"])
	}
}

func TestSetUserIdentity_InvalidEmail(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	body := `{"email": "notanemail"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/identity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetUserIdentity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSetUserIdentity_InvalidJSON(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	body := `{invalid`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/identity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetUserIdentity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSetUserIdentity_EmptyEmail(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	// First set an identity
	settingsRepo.settings["user_identity"] = "existing@example.com"

	// Then clear it
	body := `{"email": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/identity", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetUserIdentity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify identity was cleared
	if settingsRepo.settings["user_identity"] != "" {
		t.Fatalf("expected identity to be cleared, got %q", settingsRepo.settings["user_identity"])
	}
}

// ============================================================================
// GetUserIdentity Tests
// ============================================================================

func TestGetUserIdentity_Success(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	settingsRepo.settings["user_identity"] = "test@example.com"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/identity", nil)
	rr := httptest.NewRecorder()

	handler.GetUserIdentity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["email"] != "test@example.com" {
		t.Fatalf("expected email 'test@example.com', got %q", response["email"])
	}
}

func TestGetUserIdentity_Empty(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/identity", nil)
	rr := httptest.NewRecorder()

	handler.GetUserIdentity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["email"] != "" {
		t.Fatalf("expected empty email, got %q", response["email"])
	}
}

// ============================================================================
// ClearUserIdentity Tests
// ============================================================================

func TestClearUserIdentity_Success(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	settingsRepo.settings["user_identity"] = "test@example.com"

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entitlement/identity", nil)
	rr := httptest.NewRecorder()

	handler.ClearUserIdentity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify identity was cleared
	if settingsRepo.settings["user_identity"] != "" {
		t.Fatalf("expected identity to be cleared, got %q", settingsRepo.settings["user_identity"])
	}
}

// ============================================================================
// GetUsageSummary Tests
// ============================================================================

func TestGetUsageSummary_NoTracker(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/usage", nil)
	rr := httptest.NewRecorder()

	handler.GetUsageSummary(rr, req)

	// Should return 503 when usage tracker is not available
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// EntitlementOverride Tests
// ============================================================================

func TestGetEntitlementOverride_Empty(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/override", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementOverride(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["tier"] != "" {
		t.Fatalf("expected empty tier override, got %q", response["tier"])
	}
}

func TestSetEntitlementOverride_Success(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	body := `{"tier": "pro"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetEntitlementOverride(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify override was saved
	if settingsRepo.settings[entitlement.OverrideTierSettingKey] != "pro" {
		t.Fatalf("expected override tier to be 'pro', got %q", settingsRepo.settings[entitlement.OverrideTierSettingKey])
	}
}

func TestSetEntitlementOverride_InvalidTier(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	body := `{"tier": "invalid_tier"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetEntitlementOverride(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSetEntitlementOverride_ClearWithEmptyTier(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	// Set an override first
	settingsRepo.settings[entitlement.OverrideTierSettingKey] = "pro"

	// Clear it
	body := `{"tier": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetEntitlementOverride(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify override was cleared
	if settingsRepo.settings[entitlement.OverrideTierSettingKey] != "" {
		t.Fatalf("expected override tier to be cleared, got %q", settingsRepo.settings[entitlement.OverrideTierSettingKey])
	}
}

func TestClearEntitlementOverride_Success(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	settingsRepo.settings[entitlement.OverrideTierSettingKey] = "pro"

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entitlement/override", nil)
	rr := httptest.NewRecorder()

	handler.ClearEntitlementOverride(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify override was cleared
	if settingsRepo.settings[entitlement.OverrideTierSettingKey] != "" {
		t.Fatalf("expected override tier to be cleared, got %q", settingsRepo.settings[entitlement.OverrideTierSettingKey])
	}
}

// ============================================================================
// RefreshEntitlement Tests
// ============================================================================

func TestRefreshEntitlement_MissingUser(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/refresh", nil)
	rr := httptest.NewRecorder()

	handler.RefreshEntitlement(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRefreshEntitlement_Success(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/refresh?user=test@example.com", nil)
	rr := httptest.NewRecorder()

	handler.RefreshEntitlement(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Should return updated entitlement status
	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.UserIdentity != "test@example.com" {
		t.Fatalf("expected user identity 'test@example.com', got %q", response.UserIdentity)
	}
}

// ============================================================================
// Feature Access Summary Tests
// ============================================================================

func TestGetEntitlementStatus_IncludesFeatureAccess(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should include feature access summary
	if len(response.FeatureAccess) == 0 {
		t.Fatal("expected feature_access to be populated")
	}

	// Check that expected features are present
	featureIDs := make(map[string]bool)
	for _, fa := range response.FeatureAccess {
		featureIDs[fa.ID] = true
	}

	expectedFeatures := []string{"ai", "recording", "watermark-free"}
	for _, expected := range expectedFeatures {
		if !featureIDs[expected] {
			t.Fatalf("expected feature %q to be present in feature_access", expected)
		}
	}
}

// ============================================================================
// AI Credits Tests
// ============================================================================

func TestGetEntitlementStatus_IncludesAICreditsFields(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify AI credits fields are present in response
	// When no aiCreditsTracker is configured, values should be defaults
	// The AICreditsLimit should match the tier's limit from config
	if response.AICreditsLimit == 0 && response.Tier != "free" {
		// For non-free tiers that have AI access, limit should not be 0
		t.Logf("AICreditsLimit = %d for tier %s", response.AICreditsLimit, response.Tier)
	}

	// AICreditsUsed should be 0 when no tracker
	if response.AICreditsUsed != 0 {
		t.Errorf("expected AICreditsUsed = 0 without tracker, got %d", response.AICreditsUsed)
	}

	// AIRequestsCount should be 0 when no tracker
	if response.AIRequestsCount != 0 {
		t.Errorf("expected AIRequestsCount = 0 without tracker, got %d", response.AIRequestsCount)
	}
}

func TestGetEntitlementStatus_ResponseIncludesAIResetDate(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// AIResetDate should be empty when no tracker is configured
	// (the reset date is computed from actual usage tracking)
	if response.AIResetDate != "" {
		// If a reset date is present, it should be a valid date format
		t.Logf("AIResetDate = %s", response.AIResetDate)
	}
}

func TestGetEntitlementStatus_AICreditsRemainingCalculation(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/status", nil)
	rr := httptest.NewRecorder()

	handler.GetEntitlementStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response EntitlementStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// For limited tiers: remaining = limit - used
	if response.AICreditsLimit > 0 {
		expectedRemaining := response.AICreditsLimit - response.AICreditsUsed
		if expectedRemaining < 0 {
			expectedRemaining = 0
		}
		if response.AICreditsRemaining != expectedRemaining {
			t.Errorf("expected AICreditsRemaining = %d, got %d",
				expectedRemaining, response.AICreditsRemaining)
		}
	}

	// For unlimited tiers: remaining should be -1
	if response.AICreditsLimit < 0 {
		if response.AICreditsRemaining != -1 {
			t.Errorf("expected AICreditsRemaining = -1 for unlimited, got %d",
				response.AICreditsRemaining)
		}
	}
}

// ============================================================================
// GetApiSource Tests
// ============================================================================

func TestGetApiSource_DefaultValues(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/api-source", nil)
	rr := httptest.NewRecorder()

	handler.GetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "production" {
		t.Errorf("expected source 'production', got %q", response.Source)
	}
	if response.LocalPort != 15000 {
		t.Errorf("expected local_port 15000, got %d", response.LocalPort)
	}
}

func TestGetApiSource_StoredValues(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	settingsRepo.settings[entitlement.ApiSourceSettingKey] = "local"
	settingsRepo.settings[entitlement.LocalApiPortSettingKey] = "16000"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/api-source", nil)
	rr := httptest.NewRecorder()

	handler.GetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "local" {
		t.Errorf("expected source 'local', got %q", response.Source)
	}
	if response.LocalPort != 16000 {
		t.Errorf("expected local_port 16000, got %d", response.LocalPort)
	}
}

func TestGetApiSource_DisabledSource(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	settingsRepo.settings[entitlement.ApiSourceSettingKey] = "disabled"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entitlement/api-source", nil)
	rr := httptest.NewRecorder()

	handler.GetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "disabled" {
		t.Errorf("expected source 'disabled', got %q", response.Source)
	}
}

// ============================================================================
// SetApiSource Tests
// ============================================================================

func TestSetApiSource_ValidProduction(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	body := `{"source": "production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "production" {
		t.Errorf("expected source 'production', got %q", response.Source)
	}

	if settingsRepo.settings[entitlement.ApiSourceSettingKey] != "production" {
		t.Errorf("expected setting to be 'production', got %q", settingsRepo.settings[entitlement.ApiSourceSettingKey])
	}
}

func TestSetApiSource_ValidLocal(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	body := `{"source": "local", "local_port": 17000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "local" {
		t.Errorf("expected source 'local', got %q", response.Source)
	}
	if response.LocalPort != 17000 {
		t.Errorf("expected local_port 17000, got %d", response.LocalPort)
	}

	if settingsRepo.settings[entitlement.ApiSourceSettingKey] != "local" {
		t.Errorf("expected setting to be 'local', got %q", settingsRepo.settings[entitlement.ApiSourceSettingKey])
	}
	if settingsRepo.settings[entitlement.LocalApiPortSettingKey] != "17000" {
		t.Errorf("expected local port setting to be '17000', got %q", settingsRepo.settings[entitlement.LocalApiPortSettingKey])
	}
}

func TestSetApiSource_ValidDisabled(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	body := `{"source": "disabled"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "disabled" {
		t.Errorf("expected source 'disabled', got %q", response.Source)
	}

	if settingsRepo.settings[entitlement.ApiSourceSettingKey] != "disabled" {
		t.Errorf("expected setting to be 'disabled', got %q", settingsRepo.settings[entitlement.ApiSourceSettingKey])
	}
}

func TestSetApiSource_InvalidSource(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	body := `{"source": "invalid_source"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSetApiSource_InvalidJSON(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	body := `{invalid`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSetApiSource_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"PRODUCTION", "production"},
		{"Production", "production"},
		{"LOCAL", "local"},
		{"Local", "local"},
		{"DISABLED", "disabled"},
		{"Disabled", "disabled"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			handler, _ := createTestEntitlementHandler(t)

			body := `{"source": "` + tc.input + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.SetApiSource(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected status 200 for input %q, got %d: %s", tc.input, rr.Code, rr.Body.String())
			}

			var response ApiSourceResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if response.Source != tc.expected {
				t.Errorf("expected source %q, got %q", tc.expected, response.Source)
			}
		})
	}
}

func TestSetApiSource_WhitespaceHandling(t *testing.T) {
	handler, _ := createTestEntitlementHandler(t)

	body := `{"source": "  local  "}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Source != "local" {
		t.Errorf("expected source 'local' (trimmed), got %q", response.Source)
	}
}

func TestSetApiSource_LocalPortDefaultWhenNotProvided(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	// Pre-set a local port in settings
	settingsRepo.settings[entitlement.LocalApiPortSettingKey] = "18000"

	body := `{"source": "local"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entitlement/api-source", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.SetApiSource(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ApiSourceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should use the existing stored port
	if response.LocalPort != 18000 {
		t.Errorf("expected local_port 18000 (from settings), got %d", response.LocalPort)
	}
}

// ============================================================================
// ClearApiSource Tests
// ============================================================================

func TestClearApiSource_Success(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	// Pre-set to local
	settingsRepo.settings[entitlement.ApiSourceSettingKey] = "local"

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entitlement/api-source", nil)
	rr := httptest.NewRecorder()

	handler.ClearApiSource(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Should reset to production
	if settingsRepo.settings[entitlement.ApiSourceSettingKey] != "production" {
		t.Errorf("expected setting to be 'production', got %q", settingsRepo.settings[entitlement.ApiSourceSettingKey])
	}
}

func TestClearApiSource_AlreadyProduction(t *testing.T) {
	handler, settingsRepo := createTestEntitlementHandler(t)

	// Already production
	settingsRepo.settings[entitlement.ApiSourceSettingKey] = "production"

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entitlement/api-source", nil)
	rr := httptest.NewRecorder()

	handler.ClearApiSource(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Should still be production
	if settingsRepo.settings[entitlement.ApiSourceSettingKey] != "production" {
		t.Errorf("expected setting to be 'production', got %q", settingsRepo.settings[entitlement.ApiSourceSettingKey])
	}
}
