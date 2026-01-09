package main

import (
	"database/sql"
	"os"
	"testing"
)

// [REQ:TM-AC-008] Campaign statistics tracking
func TestAutoCampaign_Statistics(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignStatsTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-stats", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Start campaign
	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Record multiple sessions
	err = aco.RecordCampaignSession(campaign.ID, 5)
	if err != nil {
		t.Fatalf("failed to record session 1: %v", err)
	}

	err = aco.RecordCampaignSession(campaign.ID, 3)
	if err != nil {
		t.Fatalf("failed to record session 2: %v", err)
	}

	// Get campaign and verify stats
	updatedCampaign, err := aco.GetCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to get campaign: %v", err)
	}

	if updatedCampaign.CurrentSession != 2 {
		t.Errorf("expected current_session=2, got %d", updatedCampaign.CurrentSession)
	}

	if updatedCampaign.FilesVisited != 8 {
		t.Errorf("expected files_visited=8 (5+3), got %d", updatedCampaign.FilesVisited)
	}
}

// [REQ:TM-AC-008] Test edge case: statistics with zero sessions
func TestAutoCampaign_ZeroSessionStats(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignStatsTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-zero-stats", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Don't start or record any sessions - verify initial state
	fetchedCampaign, err := aco.GetCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to get campaign: %v", err)
	}

	if fetchedCampaign.CurrentSession != 0 {
		t.Errorf("expected current_session=0 initially, got %d", fetchedCampaign.CurrentSession)
	}

	if fetchedCampaign.FilesVisited != 0 {
		t.Errorf("expected files_visited=0 initially, got %d", fetchedCampaign.FilesVisited)
	}

	if fetchedCampaign.ErrorCount != 0 {
		t.Errorf("expected error_count=0 initially, got %d", fetchedCampaign.ErrorCount)
	}
}

// [REQ:TM-AC-008] Test session recording with zero files
func TestAutoCampaign_SessionWithZeroFiles(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignStatsTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-zero-file-session", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Record session with 0 files visited
	err = aco.RecordCampaignSession(campaign.ID, 0)
	if err != nil {
		t.Fatalf("failed to record zero-file session: %v", err)
	}

	// Verify session counted but files_visited remains 0
	updatedCampaign, err := aco.GetCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to get campaign: %v", err)
	}

	if updatedCampaign.CurrentSession != 1 {
		t.Errorf("expected current_session=1, got %d", updatedCampaign.CurrentSession)
	}

	if updatedCampaign.FilesVisited != 0 {
		t.Errorf("expected files_visited=0 after zero-file session, got %d", updatedCampaign.FilesVisited)
	}
}

// setupAutoCampaignStatsTest creates the test environment for statistics tests
func setupAutoCampaignStatsTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
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
