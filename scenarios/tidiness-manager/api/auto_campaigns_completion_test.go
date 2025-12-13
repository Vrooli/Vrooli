package main

import (
	"database/sql"
	"os"
	"testing"
)

// [REQ:TM-AC-003] Auto-complete when all files visited
func TestAutoCampaign_AutoCompleteAllFilesVisited(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignCompletionTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-completion", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Start campaign
	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Set files_total and files_visited to match
	_, err = db.Exec(`
		UPDATE campaigns
		SET files_total = 10, files_visited = 10, visited_tracker_campaign_id = 123
		WHERE id = $1
	`, campaign.ID)
	if err != nil {
		t.Fatalf("failed to update campaign: %v", err)
	}

	// Check for auto-completion
	err = aco.CheckAndAutoComplete(campaign.ID)
	if err != nil {
		t.Fatalf("auto-complete check failed: %v", err)
	}

	// Verify campaign marked as completed
	var status string
	var completedAt sql.NullTime
	err = db.QueryRow(`
		SELECT status, completed_at FROM campaigns WHERE id = $1
	`, campaign.ID).Scan(&status, &completedAt)
	if err != nil {
		t.Fatalf("failed to get campaign status: %v", err)
	}

	if status != "completed" {
		t.Errorf("expected status=completed, got %s", status)
	}

	if !completedAt.Valid {
		t.Error("expected completed_at to be set")
	}
}

// [REQ:TM-AC-004] Auto-complete when max sessions reached
func TestAutoCampaign_AutoCompleteMaxSessions(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignCompletionTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-max-sessions", 5, 3)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Start campaign
	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Record 5 sessions (reaching max_sessions)
	for i := 0; i < 5; i++ {
		err = aco.RecordCampaignSession(campaign.ID, 3)
		if err != nil {
			t.Fatalf("failed to record session %d: %v", i, err)
		}
	}

	// Verify campaign auto-completed
	var status string
	err = db.QueryRow(`
		SELECT status FROM campaigns WHERE id = $1
	`, campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get campaign status: %v", err)
	}

	if status != "completed" {
		t.Errorf("expected status=completed after max sessions, got %s", status)
	}
}

// [REQ:TM-AC-003] [REQ:TM-AC-004] Test edge case: check auto-complete doesn't trigger prematurely
func TestAutoCampaign_NoPreemptiveCompletion(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignCompletionTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-no-preempt", 5, 3)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Set files_visited to less than files_total
	_, err = db.Exec(`
		UPDATE campaigns
		SET files_total = 10, files_visited = 5
		WHERE id = $1
	`, campaign.ID)
	if err != nil {
		t.Fatalf("failed to update campaign: %v", err)
	}

	// Check auto-complete (should not complete)
	err = aco.CheckAndAutoComplete(campaign.ID)
	if err != nil {
		t.Fatalf("auto-complete check failed: %v", err)
	}

	// Verify still active
	var status string
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "active" {
		t.Errorf("expected status=active when files_visited < files_total, got %s", status)
	}
}

// setupAutoCampaignCompletionTest creates the test environment for completion tests
func setupAutoCampaignCompletionTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
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
