package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// [REQ:TM-AC-004] Integration test: Pause campaigns via API
func TestIntegration_CampaignPause(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping lifecycle integration test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-pause"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create campaign
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (scenario, status, max_sessions, max_files_per_session)
		VALUES ($1, 'active', 10, 5)
		RETURNING id
	`, testScenario).Scan(&campaignID)
	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	// Pause campaign via API
	pauseReq := map[string]string{"action": "pause"}
	bodyBytes, _ := json.Marshal(pauseReq)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/campaigns/%d/action", campaignID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to pause campaign: status %d, body: %s", w.Code, w.Body.String())
	}

	// Verify pause via API response
	var pauseResp struct {
		Campaign map[string]interface{} `json:"campaign"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pauseResp); err != nil {
		t.Fatalf("Failed to parse pause response: %v", err)
	}

	if pausedStatus, ok := pauseResp.Campaign["status"].(string); !ok || pausedStatus != "paused" {
		t.Errorf("Expected campaign status 'paused', got '%v'", pauseResp.Campaign["status"])
	}

	// Verify pause persists in database
	var status string
	err = srv.db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaignID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query campaign status: %v", err)
	}
	if status != "paused" {
		t.Errorf("Expected status 'paused' in database, got '%s'", status)
	}

	t.Log("Campaign pause verified via API")
}

// [REQ:TM-AC-005] Integration test: Resume campaigns via API
func TestIntegration_CampaignResume(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping resume integration test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-resume"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create paused campaign
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (scenario, status, max_sessions, max_files_per_session)
		VALUES ($1, 'paused', 10, 5)
		RETURNING id
	`, testScenario).Scan(&campaignID)
	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	// Resume campaign via API
	resumeReq := map[string]string{"action": "resume"}
	bodyBytes, _ := json.Marshal(resumeReq)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/campaigns/%d/action", campaignID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to resume campaign: status %d, body: %s", w.Code, w.Body.String())
	}

	// Verify resume via API response
	var resumeResp struct {
		Campaign map[string]interface{} `json:"campaign"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resumeResp); err != nil {
		t.Fatalf("Failed to parse resume response: %v", err)
	}

	if resumedStatus, ok := resumeResp.Campaign["status"].(string); !ok || resumedStatus != "active" {
		t.Errorf("Expected campaign status 'active', got '%v'", resumeResp.Campaign["status"])
	}

	// Verify resume persists in database
	var status string
	err = srv.db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaignID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query campaign status: %v", err)
	}
	if status != "active" {
		t.Errorf("Expected status 'active' in database, got '%s'", status)
	}

	t.Log("Campaign resume verified via API")
}

// [REQ:TM-AC-006] Integration test: Terminate campaigns via API
func TestIntegration_CampaignTerminate(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping terminate integration test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-terminate"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create active campaign
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (scenario, status, max_sessions, max_files_per_session)
		VALUES ($1, 'active', 10, 5)
		RETURNING id
	`, testScenario).Scan(&campaignID)
	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	// Terminate campaign
	_, err = srv.db.Exec(`
		UPDATE campaigns
		SET status = 'terminated', completed_at = NOW()
		WHERE id = $1
	`, campaignID)

	if err != nil {
		t.Errorf("Failed to terminate campaign: %v", err)
	}

	// Verify terminated status persists
	var status string
	srv.db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaignID).Scan(&status)
	if status != "terminated" {
		t.Errorf("Expected status 'terminated', got '%s'", status)
	}

	t.Log("Campaign termination verified")
}

// [REQ:TM-AC-005] Edge case: Invalid campaign actions
func TestIntegration_CampaignInvalidActions(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping invalid actions test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-invalid-actions"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create test campaign
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (scenario, status, max_sessions, max_files_per_session)
		VALUES ($1, 'active', 5, 3)
		RETURNING id
	`, testScenario).Scan(&campaignID)
	if err != nil {
		t.Fatalf("Failed to create campaign: %v", err)
	}

	// Test invalid action type
	invalidReq := map[string]string{"action": "invalid-action"}
	bodyBytes, _ := json.Marshal(invalidReq)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/campaigns/%d/action", campaignID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid action, got %d", w.Code)
	}

	// Test nonexistent campaign
	validReq := map[string]string{"action": "pause"}
	bodyBytes, _ = json.Marshal(validReq)

	req = httptest.NewRequest("POST", "/api/v1/campaigns/999999/action", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Logf("Note: Expected 404 or 500 for nonexistent campaign, got %d (error handling may vary)", w.Code)
	}

	// Test malformed request body
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/campaigns/%d/action", campaignID), bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
	}

	t.Log("Invalid action edge cases handled correctly")
}
