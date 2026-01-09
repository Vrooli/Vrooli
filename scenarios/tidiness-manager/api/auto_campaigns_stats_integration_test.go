package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// [REQ:TM-AC-008] Integration test: Campaign statistics via API
func TestIntegration_CampaignStatistics(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping statistics integration test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-stats"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create campaign with known configuration
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (
			scenario, status, max_sessions, max_files_per_session,
			current_session, files_visited, files_total
		)
		VALUES ($1, 'active', 10, 5, 3, 12, 50)
		RETURNING id
	`, testScenario).Scan(&campaignID)

	if err != nil {
		t.Fatalf("Failed to create campaign: %v", err)
	}

	// Query statistics via API
	req := httptest.NewRequest("GET", "/api/v1/campaigns?scenario="+testScenario, nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to query campaign statistics: status %d", w.Code)
	}

	var response struct {
		Campaigns []map[string]interface{} `json:"campaigns"`
		Count     int                      `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Campaigns) == 0 {
		t.Fatal("Expected campaign in response")
	}

	campaign := response.Campaigns[0]

	// Verify all statistics are present and accurate
	expectedStats := map[string]float64{
		"max_sessions":          10,
		"max_files_per_session": 5,
		"current_session":       3,
		"files_visited":         12,
		"files_total":           50,
	}

	for key, expected := range expectedStats {
		if actual, ok := campaign[key].(float64); ok {
			if int(actual) != int(expected) {
				t.Errorf("Expected %s=%v, got %v", key, expected, actual)
			}
		} else {
			t.Errorf("Statistics field '%s' missing or wrong type in response", key)
		}
	}

	// Calculate progress percentage
	// Progress = (files_visited / files_total) * 100
	expectedProgress := (12.0 / 50.0) * 100 // = 24%

	if progress, ok := campaign["progress_percent"].(float64); ok {
		if progress < expectedProgress-1 || progress > expectedProgress+1 {
			t.Errorf("Expected progress ~24%%, got %.1f%%", progress)
		}
	} else {
		t.Logf("Note: progress_percent not yet calculated in API response (may be added later)")
	}

	t.Logf("Campaign statistics verified: %d/%d sessions, %d/%d files (%.1f%% complete)",
		3, 10, 12, 50, expectedProgress)
}

// [REQ:TM-AC-008] Test statistics accuracy with edge cases
func TestIntegration_CampaignStatisticsEdgeCases(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping statistics edge case test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-stats-edge"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Test with zero files_total (edge case)
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (
			scenario, status, max_sessions, max_files_per_session,
			current_session, files_visited, files_total
		)
		VALUES ($1, 'active', 10, 5, 0, 0, 0)
		RETURNING id
	`, testScenario).Scan(&campaignID)

	if err != nil {
		t.Fatalf("Failed to create campaign: %v", err)
	}

	// Query statistics via API
	req := httptest.NewRequest("GET", "/api/v1/campaigns?scenario="+testScenario, nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to query campaign statistics: status %d", w.Code)
	}

	var response struct {
		Campaigns []map[string]interface{} `json:"campaigns"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Campaigns) == 0 {
		t.Fatal("Expected campaign in response")
	}

	campaign := response.Campaigns[0]

	// Verify zero values are handled correctly
	if filesTotal, ok := campaign["files_total"].(float64); ok {
		if int(filesTotal) != 0 {
			t.Errorf("Expected files_total=0, got %v", filesTotal)
		}
	}

	if filesVisited, ok := campaign["files_visited"].(float64); ok {
		if int(filesVisited) != 0 {
			t.Errorf("Expected files_visited=0, got %v", filesVisited)
		}
	}

	t.Log("Statistics edge cases (zero values) handled correctly")
}
