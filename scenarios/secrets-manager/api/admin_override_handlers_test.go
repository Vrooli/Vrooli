package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/lib/pq"
)

// TestListOrphans_NoOrphans tests listing orphans when all overrides are valid.
func TestListOrphans_NoOrphans(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewAdminOverrideHandlers(db, &Logger{})

	req := httptest.NewRequest("GET", "/admin/overrides/orphans", nil)
	rec := httptest.NewRecorder()

	handlers.ListOrphans(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Count can be >= 0, just verify it's a valid response
	if _, ok := resp["count"]; !ok {
		t.Error("expected 'count' field in response")
	}
	if _, ok := resp["orphans"]; !ok {
		t.Error("expected 'orphans' field in response")
	}
}

// TestCleanupOrphans_DryRun tests cleanup in dry-run mode.
func TestCleanupOrphans_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewAdminOverrideHandlers(db, &Logger{})

	payload := CleanupOrphansRequest{DryRun: true}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/admin/overrides/cleanup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.CleanupOrphans(rec, req)

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
	if resp["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", resp["dry_run"])
	}
	// In dry run, deleted should be 0 (nothing actually deleted)
	if resp["deleted"].(float64) != 0 {
		t.Errorf("expected deleted=0 in dry run, got %v", resp["deleted"])
	}
}

// TestCleanupOrphans_DefaultToDryRun tests that cleanup defaults to dry-run without body.
func TestCleanupOrphans_DefaultToDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	handlers := NewAdminOverrideHandlers(db, &Logger{})

	// Send request with empty body - should default to dry_run=true
	req := httptest.NewRequest("POST", "/admin/overrides/cleanup", nil)
	rec := httptest.NewRecorder()

	handlers.CleanupOrphans(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["dry_run"] != true {
		t.Errorf("expected dry_run=true by default, got %v", resp["dry_run"])
	}
}

// TestCleanupOrphans_WithOrphan tests cleanup when an orphan override exists.
func TestCleanupOrphans_WithOrphan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	// Setup: Create a resource secret and an override for a non-existent scenario
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)
	defer cleanupTestData(t, db, resourceSecretID, "")

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()
	handling := "strip"
	// Create override for a scenario that doesn't exist (will be an orphan)
	override, err := store.UpsertOverride(ctx, "definitely-nonexistent-scenario-xyz123", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("failed to create test override: %v", err)
	}
	defer func() {
		// Ensure cleanup regardless of test outcome
		_, _ = db.Exec("DELETE FROM scenario_secret_strategy_overrides WHERE id = $1", override.ID)
	}()

	handlers := NewAdminOverrideHandlers(db, &Logger{})

	// First, list orphans to see if our override is detected
	listReq := httptest.NewRequest("GET", "/admin/overrides/orphans", nil)
	listRec := httptest.NewRecorder()
	handlers.ListOrphans(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Errorf("expected status 200 for list, got %d", listRec.Code)
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	// The orphan detection depends on whether scenario list is available
	// If it is, we should find our orphan; if not, it's skipped (fail-open)
	t.Logf("Orphan detection found %v orphans", listResp["count"])

	// Now test actual cleanup (not dry-run)
	cleanupPayload := CleanupOrphansRequest{DryRun: false}
	cleanupBody, _ := json.Marshal(cleanupPayload)

	cleanupReq := httptest.NewRequest("POST", "/admin/overrides/cleanup", bytes.NewReader(cleanupBody))
	cleanupReq.Header.Set("Content-Type", "application/json")
	cleanupRec := httptest.NewRecorder()

	handlers.CleanupOrphans(cleanupRec, cleanupReq)

	if cleanupRec.Code != http.StatusOK {
		t.Errorf("expected status 200 for cleanup, got %d: %s", cleanupRec.Code, cleanupRec.Body.String())
	}

	var cleanupResp map[string]interface{}
	if err := json.NewDecoder(cleanupRec.Body).Decode(&cleanupResp); err != nil {
		t.Fatalf("failed to decode cleanup response: %v", err)
	}

	if cleanupResp["success"] != true {
		t.Errorf("expected success=true, got %v", cleanupResp["success"])
	}
	if cleanupResp["dry_run"] != false {
		t.Errorf("expected dry_run=false, got %v", cleanupResp["dry_run"])
	}
}

// TestListOrphans_DatabaseUnavailable tests handling when database is nil.
func TestListOrphans_DatabaseUnavailable(t *testing.T) {
	handlers := &AdminOverrideHandlers{
		store:  nil,
		logger: &Logger{},
	}

	req := httptest.NewRequest("GET", "/admin/overrides/orphans", nil)
	rec := httptest.NewRecorder()

	handlers.ListOrphans(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

// TestCleanupOrphans_DatabaseUnavailable tests handling when database is nil.
func TestCleanupOrphans_DatabaseUnavailable(t *testing.T) {
	handlers := &AdminOverrideHandlers{
		store:  nil,
		logger: &Logger{},
	}

	req := httptest.NewRequest("POST", "/admin/overrides/cleanup", nil)
	rec := httptest.NewRecorder()

	handlers.CleanupOrphans(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}
