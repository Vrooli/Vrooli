package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// ============================================================================
// handleGetTierLimits Tests
// ============================================================================

func TestHandleGetTierLimits_AllTiers(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleGetTierLimits(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/limits", nil)
	req = mux.SetURLVars(req, map[string]string{}) // No tier param

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	limits, ok := resp["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'limits' to be a map")
	}

	// Should have multiple tiers
	if len(limits) == 0 {
		t.Error("expected limits to contain tier data")
	}
}

func TestHandleGetTierLimits_SpecificTier(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleGetTierLimits(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/limits/solo", nil)
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	tierID, ok := resp["tier_id"].(string)
	if !ok || tierID != "solo" {
		t.Errorf("expected tier_id 'solo', got %v", resp["tier_id"])
	}

	limits, ok := resp["limits"].([]interface{})
	if !ok {
		t.Fatal("expected 'limits' to be an array")
	}

	if len(limits) != 1 {
		t.Errorf("expected 1 limit, got %d", len(limits))
	}
}

func TestHandleGetTierLimits_NonExistentTier(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleGetTierLimits(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/limits/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"tier": "nonexistent"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	limits, ok := resp["limits"].([]interface{})
	if !ok {
		t.Fatal("expected 'limits' to be an array")
	}

	if len(limits) != 0 {
		t.Errorf("expected empty limits for nonexistent tier, got %d", len(limits))
	}
}

// ============================================================================
// handleUpdateTierLimits Tests
// ============================================================================

func TestHandleUpdateTierLimits_Success(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleUpdateTierLimits(svc)

	newValue := int64(999999999)
	body := map[string]interface{}{
		"limit_key": "ai_credits",
		"update": map[string]interface{}{
			"limit_value": newValue,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TierLimit
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.LimitValue != newValue {
		t.Errorf("expected limit_value %d, got %d", newValue, resp.LimitValue)
	}
}

func TestHandleUpdateTierLimits_MissingTierID(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleUpdateTierLimits(svc)

	body := map[string]interface{}{
		"limit_key": "ai_credits",
		"update":    map[string]interface{}{"limit_value": 100},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": ""}) // Empty tier

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleUpdateTierLimits_MissingLimitKey(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleUpdateTierLimits(svc)

	body := map[string]interface{}{
		"update": map[string]interface{}{"limit_value": 100},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateTierLimits_InvalidJSON(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleUpdateTierLimits(svc)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleUpdateTierLimits_LimitNotFound(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	// Don't seed any limits

	handler := handleUpdateTierLimits(svc)

	body := map[string]interface{}{
		"limit_key": "nonexistent_key",
		"update":    map[string]interface{}{"limit_value": 100},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// ============================================================================
// handleGetAppLimits Tests
// ============================================================================

func TestHandleGetAppLimits_Success(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	// Seed app-specific limits
	appKey := "test-app"
	_, err := db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'app_specific', 'feature_count', 10, 1, ?),
		       ('pro', 'app_specific', 'feature_count', 50, 1, ?)
	`, appKey, appKey)
	if err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	handler := handleGetAppLimits(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/apps/test-app/limits", nil)
	req = mux.SetURLVars(req, map[string]string{"app": appKey})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["app_bundle_key"] != appKey {
		t.Errorf("expected app_bundle_key '%s', got %v", appKey, resp["app_bundle_key"])
	}
}

func TestHandleGetAppLimits_MissingAppKey(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleGetAppLimits(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/apps//limits", nil)
	req = mux.SetURLVars(req, map[string]string{"app": ""})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleGetAppLimits_EmptyResult(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleGetAppLimits(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin/apps/nonexistent/limits", nil)
	req = mux.SetURLVars(req, map[string]string{"app": "nonexistent"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	limits, ok := resp["limits"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'limits' to be a map")
	}

	if len(limits) != 0 {
		t.Errorf("expected empty limits, got %d tiers", len(limits))
	}
}

// ============================================================================
// handleCreateTierLimit Tests
// ============================================================================

func TestHandleCreateTierLimit_Success(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleCreateTierLimit(svc)

	body := map[string]interface{}{
		"tier_id":     "new_tier",
		"limit_type":  "cost_based",
		"limit_key":   "ai_credits",
		"limit_value": 1000000,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/limits", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp TierLimit
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.TierID != "new_tier" {
		t.Errorf("expected tier_id 'new_tier', got '%s'", resp.TierID)
	}
	if resp.LimitKey != "ai_credits" {
		t.Errorf("expected limit_key 'ai_credits', got '%s'", resp.LimitKey)
	}
}

func TestHandleCreateTierLimit_InvalidJSON(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleCreateTierLimit(svc)

	req := httptest.NewRequest(http.MethodPost, "/admin/limits", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateTierLimit_ValidationError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleCreateTierLimit(svc)

	body := map[string]interface{}{
		"tier_id":    "", // Empty - validation error
		"limit_type": "cost_based",
		"limit_key":  "ai_credits",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/limits", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateTierLimit_InvalidLimitType(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleCreateTierLimit(svc)

	body := map[string]interface{}{
		"tier_id":    "test",
		"limit_type": "invalid_type",
		"limit_key":  "ai_credits",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/limits", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// handleDeleteTierLimit Tests
// ============================================================================

func TestHandleDeleteTierLimit_Success(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleDeleteTierLimit(svc)

	body := map[string]interface{}{
		"tier_id":   "solo",
		"limit_key": "ai_credits",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodDelete, "/admin/limits", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's deleted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM subscription_tier_limits WHERE tier_id = 'solo'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 0 {
		t.Error("expected limit to be deleted")
	}
}

func TestHandleDeleteTierLimit_MissingFields(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleDeleteTierLimit(svc)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing tier_id",
			body: map[string]interface{}{"limit_key": "ai_credits"},
		},
		{
			name: "missing limit_key",
			body: map[string]interface{}{"tier_id": "solo"},
		},
		{
			name: "both missing",
			body: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodDelete, "/admin/limits", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleDeleteTierLimit_InvalidJSON(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleDeleteTierLimit(svc)

	req := httptest.NewRequest(http.MethodDelete, "/admin/limits", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleDeleteTierLimit_NotFound(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	handler := handleDeleteTierLimit(svc)

	body := map[string]interface{}{
		"tier_id":   "nonexistent",
		"limit_key": "nonexistent",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodDelete, "/admin/limits", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteTierLimit_WithAppBundleKey(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	// Seed app-specific limit
	appKey := "test-app"
	_, err := db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'app_specific', 'feature', 10, 1, ?)
	`, appKey)
	if err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	handler := handleDeleteTierLimit(svc)

	body := map[string]interface{}{
		"tier_id":        "solo",
		"limit_key":      "feature",
		"app_bundle_key": appKey,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodDelete, "/admin/limits", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// handleUpdateTierLimits with AppBundleKey Tests
// ============================================================================

func TestHandleUpdateTierLimits_WithAppBundleKey(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	// Seed app-specific limit
	appKey := "test-app"
	_, err := db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'app_specific', 'feature', 10, 1, ?)
	`, appKey)
	if err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	handler := handleUpdateTierLimits(svc)

	newValue := int64(20)
	body := map[string]interface{}{
		"limit_key":      "feature",
		"app_bundle_key": appKey,
		"update": map[string]interface{}{
			"limit_value": newValue,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TierLimit
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.LimitValue != newValue {
		t.Errorf("expected limit_value %d, got %d", newValue, resp.LimitValue)
	}
}

func TestHandleUpdateTierLimits_SetUnlimited(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleUpdateTierLimits(svc)

	body := map[string]interface{}{
		"limit_key": "ai_credits",
		"update": map[string]interface{}{
			"is_unlimited": true,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TierLimit
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.LimitValue != -1 {
		t.Errorf("expected limit_value -1 (unlimited), got %d", resp.LimitValue)
	}
}

func TestHandleUpdateTierLimits_SetDisplayDollars(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()
	seedTestTierLimits(t, db)

	handler := handleUpdateTierLimits(svc)

	body := map[string]interface{}{
		"limit_key": "ai_credits",
		"update": map[string]interface{}{
			"display_dollars": 25.0, // $25
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/limits/solo", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"tier": "solo"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TierLimit
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// $25 = 25 * 100 * 1,000,000 = 2,500,000,000
	expectedValue := int64(2500000000)
	if resp.LimitValue != expectedValue {
		t.Errorf("expected limit_value %d, got %d", expectedValue, resp.LimitValue)
	}
}
