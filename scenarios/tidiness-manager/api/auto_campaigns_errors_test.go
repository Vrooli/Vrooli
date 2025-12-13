package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
)

// [REQ:TM-AC-007] Mark campaign as error on repeated failures
func TestAutoCampaign_ErrorThreshold(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignErrorsTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-errors", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Start campaign
	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Record 2 errors - should stay active
	err = aco.RecordCampaignError(campaign.ID, "AI timeout")
	if err != nil {
		t.Fatalf("failed to record error 1: %v", err)
	}

	err = aco.RecordCampaignError(campaign.ID, "Network failure")
	if err != nil {
		t.Fatalf("failed to record error 2: %v", err)
	}

	var status string
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "active" {
		t.Errorf("expected status=active after 2 errors, got %s", status)
	}

	// Record 3rd error - should transition to error
	err = aco.RecordCampaignError(campaign.ID, "Persistent AI failure")
	if err != nil {
		t.Fatalf("failed to record error 3: %v", err)
	}

	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "error" {
		t.Errorf("expected status=error after 3 errors, got %s", status)
	}
}

// [REQ:TM-AC-007] Test edge case: error threshold exactly at limit
func TestAutoCampaign_ErrorThresholdBoundary(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignErrorsTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-error-boundary", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Record exactly 3 errors (threshold)
	for i := 0; i < 3; i++ {
		err = aco.RecordCampaignError(campaign.ID, fmt.Sprintf("Error %d", i+1))
		if err != nil {
			t.Fatalf("failed to record error %d: %v", i+1, err)
		}
	}

	// Verify transitioned to error state
	var status string
	var errorCount int
	err = db.QueryRow("SELECT status, error_count FROM campaigns WHERE id = $1", campaign.ID).
		Scan(&status, &errorCount)
	if err != nil {
		t.Fatalf("failed to get campaign: %v", err)
	}

	if status != "error" {
		t.Errorf("expected status=error at threshold (3 errors), got %s", status)
	}

	if errorCount != 3 {
		t.Errorf("expected error_count=3, got %d", errorCount)
	}
}

// setupAutoCampaignErrorsTest creates the test environment for error handling tests
func setupAutoCampaignErrorsTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping auto-campaign tests")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Create campaigns table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS campaigns (
			id SERIAL PRIMARY KEY,
			scenario VARCHAR(255) NOT NULL,
			status VARCHAR(20) DEFAULT 'created',
			max_sessions INTEGER DEFAULT 10,
			max_files_per_session INTEGER DEFAULT 5,
			current_session INTEGER DEFAULT 0,
			files_visited INTEGER DEFAULT 0,
			files_total INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			error_reason TEXT,
			visited_tracker_campaign_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create campaigns table: %v", err)
	}

	// Create config table with max_concurrent_campaigns
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key VARCHAR(100) PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create config table: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO config (key, value, description)
		VALUES ('max_concurrent_campaigns', '3', 'Max concurrent campaigns')
		ON CONFLICT (key) DO UPDATE SET value = '3'
	`)
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	campaignMgr := NewCampaignManager()
	orchestrator, err := NewAutoCampaignOrchestrator(db, campaignMgr)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM campaigns WHERE scenario LIKE 'test-%'")
		db.Close()
	}

	return db, orchestrator, cleanup
}
