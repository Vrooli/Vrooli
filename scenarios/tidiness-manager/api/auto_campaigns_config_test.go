package main

import (
	"database/sql"
	"os"
	"testing"
)

// [REQ:TM-AC-001] Campaign configuration with max_sessions and max_files_per_session
func TestAutoCampaign_Configuration(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignConfigTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-1", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	if campaign.MaxSessions != 10 {
		t.Errorf("expected max_sessions=10, got %d", campaign.MaxSessions)
	}

	if campaign.MaxFilesPerSession != 5 {
		t.Errorf("expected max_files_per_session=5, got %d", campaign.MaxFilesPerSession)
	}

	if campaign.Status != "created" {
		t.Errorf("expected status=created, got %s", campaign.Status)
	}
}

// [REQ:TM-AC-001] Test edge case: zero or negative session limits
func TestAutoCampaign_InvalidSessionLimits(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignConfigTest(t)
	defer cleanup()

	// Zero max sessions
	campaign1, err := aco.CreateAutoCampaign("test-zero-sessions", 0, 5)
	if err != nil {
		t.Fatalf("failed to create campaign with zero sessions: %v", err)
	}
	if campaign1.MaxSessions != defaultMaxSessions {
		t.Errorf("expected max_sessions to default to %d, got %d", defaultMaxSessions, campaign1.MaxSessions)
	}

	// Negative max sessions
	campaign2, err := aco.CreateAutoCampaign("test-negative-sessions", -5, 5)
	if err != nil {
		t.Fatalf("failed to create campaign with negative sessions: %v", err)
	}
	if campaign2.MaxSessions != defaultMaxSessions {
		t.Errorf("expected negative max_sessions to default to %d, got %d", defaultMaxSessions, campaign2.MaxSessions)
	}

	// Zero files per session
	campaign3, err := aco.CreateAutoCampaign("test-zero-files", 10, 0)
	if err != nil {
		t.Fatalf("failed to create campaign with zero files: %v", err)
	}
	if campaign3.MaxFilesPerSession != defaultMaxFilesPerSession {
		t.Errorf("expected max_files_per_session to default to %d, got %d", defaultMaxFilesPerSession, campaign3.MaxFilesPerSession)
	}

	// Above allowed limits should be rejected
	_, err = aco.CreateAutoCampaign("test-over-limit-sessions", maxAllowedSessions+1, 5)
	if err == nil {
		t.Fatalf("expected error when max_sessions exceeds %d", maxAllowedSessions)
	}

	_, err = aco.CreateAutoCampaign("test-over-limit-files", 10, maxAllowedFilesPerSession+1)
	if err == nil {
		t.Fatalf("expected error when max_files_per_session exceeds %d", maxAllowedFilesPerSession)
	}
}

// setupAutoCampaignConfigTest creates the test environment for campaign config tests
func setupAutoCampaignConfigTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
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
