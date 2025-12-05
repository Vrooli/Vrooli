package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestListAllOverrides_Empty tests listing overrides when none exist.
func TestListAllOverrides_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("GET", "/nonexistent-scenario/overrides", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "nonexistent-scenario"})
	rec := httptest.NewRecorder()

	handlers.ListAllOverrides(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	count, ok := resp["count"].(float64)
	if !ok || count != 0 {
		t.Errorf("expected count=0, got %v", resp["count"])
	}
}

// TestListOverrides_Empty tests listing overrides for a tier when none exist.
func TestListOverrides_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("GET", "/test-scenario/overrides/tier-2-desktop", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
	})
	rec := httptest.NewRecorder()

	handlers.ListOverrides(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["count"].(float64) != 0 {
		t.Errorf("expected count=0, got %v", resp["count"])
	}
}

// TestSetOverride_ValidPayload tests creating an override with valid data.
func TestSetOverride_ValidPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	// Setup: Create a resource secret
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)
	defer cleanupTestData(t, db, resourceSecretID, "")

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	payload := SetOverrideRequest{
		HandlingStrategy: strPtr("strip"),
		OverrideReason:   strPtr("Test override for handler test"),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/tier-2-desktop/"+resourceName+"/"+secretKey, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
		"resource": resourceName,
		"secret":   secretKey,
	})
	rec := httptest.NewRecorder()

	handlers.SetOverride(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var override ScenarioSecretOverride
	if err := json.NewDecoder(rec.Body).Decode(&override); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if override.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario_name 'test-scenario', got '%s'", override.ScenarioName)
	}
	if override.HandlingStrategy == nil || *override.HandlingStrategy != "strip" {
		t.Error("expected handling_strategy 'strip'")
	}

	// Cleanup the override
	_, _ = db.Exec("DELETE FROM scenario_secret_strategy_overrides WHERE id = $1", override.ID)
}

// TestSetOverride_InvalidStrategy tests rejection of invalid handling strategy.
func TestSetOverride_InvalidStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	// Setup: Create a resource secret
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)
	defer cleanupTestData(t, db, resourceSecretID, "")

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	payload := SetOverrideRequest{
		HandlingStrategy: strPtr("invalid_strategy"),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/tier-2-desktop/"+resourceName+"/"+secretKey, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
		"resource": resourceName,
		"secret":   secretKey,
	})
	rec := httptest.NewRecorder()

	handlers.SetOverride(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid strategy, got %d", rec.Code)
	}
}

// TestSetOverride_NonexistentSecret tests handling of nonexistent secret.
func TestSetOverride_NonexistentSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	payload := SetOverrideRequest{
		HandlingStrategy: strPtr("strip"),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/tier-2-desktop/nonexistent-resource/NONEXISTENT_SECRET", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
		"resource": "nonexistent-resource",
		"secret":   "NONEXISTENT_SECRET",
	})
	rec := httptest.NewRecorder()

	handlers.SetOverride(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for nonexistent secret, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestGetOverride_NotFound tests fetching a nonexistent override.
func TestGetOverride_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("GET", "/test-scenario/overrides/tier-2-desktop/some-resource/SOME_SECRET", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
		"resource": "some-resource",
		"secret":   "SOME_SECRET",
	})
	rec := httptest.NewRecorder()

	handlers.GetOverride(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

// TestDeleteOverride_Success tests successfully deleting an override.
func TestDeleteOverride_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	// Setup: Create a resource secret and an override
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)
	defer cleanupTestData(t, db, resourceSecretID, "")

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()
	handling := "strip"
	override, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("failed to create test override: %v", err)
	}

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("DELETE", "/test-scenario/overrides/tier-2-desktop/"+resourceName+"/"+secretKey, nil)
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
		"resource": resourceName,
		"secret":   secretKey,
	})
	rec := httptest.NewRecorder()

	handlers.DeleteOverride(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}

	// Verify deletion
	fetched, _ := store.FetchOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey)
	if fetched != nil {
		t.Error("expected override to be deleted")
	}
	_ = override // silence unused warning
}

// TestGetEffectiveStrategies_NoOverrides tests effective strategies without overrides.
func TestGetEffectiveStrategies_NoOverrides(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("GET", "/test-scenario/effective/tier-2-desktop", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario",
		"tier":     "tier-2-desktop",
	})
	rec := httptest.NewRecorder()

	handlers.GetEffectiveStrategies(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["scenario"] != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got %v", resp["scenario"])
	}
	if resp["tier"] != "tier-2-desktop" {
		t.Errorf("expected tier 'tier-2-desktop', got %v", resp["tier"])
	}
}

// TestCopyFromTier_EmptySource tests copying from a tier with no overrides.
func TestCopyFromTier_EmptySource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	payload := CopyFromTierRequest{
		SourceTier: "tier-2-desktop",
		TargetTier: "tier-3-mobile",
		Overwrite:  false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/copy-from-tier", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rec := httptest.NewRecorder()

	handlers.CopyFromTier(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["copied"].(float64) != 0 {
		t.Errorf("expected copied=0 for empty source, got %v", resp["copied"])
	}
}

// TestCopyFromTier_SameTier tests rejection of same source and target tier.
func TestCopyFromTier_SameTier(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	payload := CopyFromTierRequest{
		SourceTier: "tier-2-desktop",
		TargetTier: "tier-2-desktop",
		Overwrite:  false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/copy-from-tier", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rec := httptest.NewRecorder()

	handlers.CopyFromTier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for same tier, got %d", rec.Code)
	}
}

// TestCopyFromScenario_SameScenario tests rejection of same source and target scenario.
func TestCopyFromScenario_SameScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	payload := CopyFromScenarioRequest{
		SourceScenario: "test-scenario",
		Tier:           "tier-2-desktop",
		Overwrite:      false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/copy-from-scenario", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rec := httptest.NewRecorder()

	handlers.CopyFromScenario(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for same scenario, got %d", rec.Code)
	}
}

// TestCopyFromTier_MissingFields tests rejection of missing required fields.
func TestCopyFromTier_MissingFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	// Test missing target_tier
	payload := CopyFromTierRequest{
		SourceTier: "tier-2-desktop",
		TargetTier: "",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/test-scenario/overrides/copy-from-tier", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rec := httptest.NewRecorder()

	handlers.CopyFromTier(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing fields, got %d", rec.Code)
	}
}

// TestListOverrides_WithData tests listing overrides when data exists.
func TestListOverrides_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	// Setup: Create a resource secret and an override
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)
	defer cleanupTestData(t, db, resourceSecretID, "")

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()
	handling := "generate"
	override, err := store.UpsertOverride(ctx, "test-scenario-list", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("failed to create test override: %v", err)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM scenario_secret_strategy_overrides WHERE id = $1", override.ID)
	}()

	handlers := NewScenarioOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("GET", "/test-scenario-list/overrides/tier-2-desktop", nil)
	req = mux.SetURLVars(req, map[string]string{
		"scenario": "test-scenario-list",
		"tier":     "tier-2-desktop",
	})
	rec := httptest.NewRecorder()

	handlers.ListOverrides(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	count := resp["count"].(float64)
	if count < 1 {
		t.Errorf("expected count >= 1, got %v", count)
	}
}

// Helper function for pointer conversion
func strPtr(s string) *string {
	return &s
}
