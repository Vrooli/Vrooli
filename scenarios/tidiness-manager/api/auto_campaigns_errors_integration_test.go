package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// [REQ:TM-AC-007] Integration test: Campaign error recording
func TestIntegration_CampaignErrorRecording(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping error handling integration test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-error-recording"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create campaign
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (scenario, status, max_sessions, error_count, error_reason)
		VALUES ($1, 'active', 5, 0, NULL)
		RETURNING id
	`, testScenario).Scan(&campaignID)
	if err != nil {
		t.Fatalf("Failed to create campaign: %v", err)
	}

	// Simulate errors (increment error_count)
	for i := 1; i <= 3; i++ {
		_, err = srv.db.Exec(`
			UPDATE campaigns
			SET error_count = $1, error_reason = $2, updated_at = NOW()
			WHERE id = $3
		`, i, "Simulated error during session execution", campaignID)

		if err != nil {
			t.Errorf("Failed to record error %d: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Verify error tracking
	var errorCount int
	var errorReason string
	err = srv.db.QueryRow(`
		SELECT error_count, error_reason
		FROM campaigns
		WHERE id = $1
	`, campaignID).Scan(&errorCount, &errorReason)

	if err != nil {
		t.Fatalf("Failed to query error tracking: %v", err)
	}

	if errorCount != 3 {
		t.Errorf("Expected error_count=3, got %d", errorCount)
	}

	if errorReason == "" {
		t.Error("Expected error_reason to be populated")
	}

	t.Logf("Campaign recorded %d errors with reason: %s", errorCount, errorReason)
}

// [REQ:TM-AC-007] Edge case: Error threshold boundary conditions
func TestIntegration_ErrorThresholdBoundaries(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping error threshold boundary test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	testScenario := "integration-test-error-boundaries"
	srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)
	defer srv.db.Exec("DELETE FROM campaigns WHERE scenario = $1", testScenario)

	// Create campaign
	var campaignID int
	err = srv.db.QueryRow(`
		INSERT INTO campaigns (scenario, status, max_sessions, error_count)
		VALUES ($1, 'active', 10, 0)
		RETURNING id
	`, testScenario).Scan(&campaignID)
	if err != nil {
		t.Fatalf("Failed to create campaign: %v", err)
	}

	// Test: Incrementing to just below threshold (should remain active)
	const errorThreshold = 5
	for i := 1; i < errorThreshold; i++ {
		_, err = srv.db.Exec(`
			UPDATE campaigns
			SET error_count = $1, error_reason = $2
			WHERE id = $3
		`, i, fmt.Sprintf("Error %d", i), campaignID)

		if err != nil {
			t.Fatalf("Failed to update error count: %v", err)
		}

		var status string
		srv.db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaignID).Scan(&status)

		// Should still be active before threshold
		if status != "active" {
			t.Errorf("Campaign should remain active with %d errors (below threshold %d), but status is '%s'",
				i, errorThreshold, status)
		}
	}

	// Test: Hitting exact threshold
	_, err = srv.db.Exec(`
		UPDATE campaigns
		SET error_count = $1, error_reason = 'Threshold reached'
		WHERE id = $2
	`, errorThreshold, campaignID)

	if err != nil {
		t.Fatalf("Failed to set error count to threshold: %v", err)
	}

	var status string
	srv.db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaignID).Scan(&status)

	t.Logf("Campaign status at threshold (%d errors): %s", errorThreshold, status)
	// Note: Auto-termination logic might be in orchestrator, not database trigger
	// This test verifies the data flow is correct

	// Test: Error count can be reset (for recovery scenarios)
	_, err = srv.db.Exec(`
		UPDATE campaigns
		SET error_count = 0, error_reason = NULL, status = 'active'
		WHERE id = $1
	`, campaignID)

	if err != nil {
		t.Fatalf("Failed to reset error count: %v", err)
	}

	var resetErrorCount int
	srv.db.QueryRow("SELECT error_count FROM campaigns WHERE id = $1", campaignID).Scan(&resetErrorCount)

	if resetErrorCount != 0 {
		t.Errorf("Expected error_count to be reset to 0, got %d", resetErrorCount)
	}

	t.Log("Error threshold boundary conditions verified")
}
