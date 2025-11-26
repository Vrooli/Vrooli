package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
)

func setupAutoCampaignTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
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

// [REQ:TM-AC-001] Campaign configuration with max_sessions and max_files_per_session
func TestAutoCampaign_Configuration(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-002] Enforce maximum K concurrent campaigns
func TestAutoCampaign_ConcurrencyLimit(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
	defer cleanup()

	// Create 3 campaigns (at the limit)
	for i := 0; i < 3; i++ {
		campaign, err := aco.CreateAutoCampaign(fmt.Sprintf("test-scenario-%d", i), 10, 5)
		if err != nil {
			t.Fatalf("failed to create campaign %d: %v", i, err)
		}

		// Start the campaign to keep it active
		err = aco.StartCampaign(campaign.ID)
		if err != nil {
			t.Fatalf("failed to start campaign %d: %v", i, err)
		}
	}

	// Try to create a 4th campaign - should fail
	_, err := aco.CreateAutoCampaign("test-scenario-overflow", 10, 5)
	if err == nil {
		t.Error("expected error when exceeding max concurrent campaigns, got nil")
	}

	if !contains(err.Error(), "max concurrent campaigns") {
		t.Errorf("expected concurrency error, got: %v", err)
	}
}

// [REQ:TM-AC-003] Auto-complete when all files visited
func TestAutoCampaign_AutoCompleteAllFilesVisited(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
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
	db, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-005] Campaign pause and resume
func TestAutoCampaign_PauseResume(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-pause", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Start campaign
	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Pause campaign
	err = aco.PauseCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to pause campaign: %v", err)
	}

	// Verify status
	var status string
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "paused" {
		t.Errorf("expected status=paused, got %s", status)
	}

	// Resume campaign
	err = aco.ResumeCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to resume campaign: %v", err)
	}

	// Verify status
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "active" {
		t.Errorf("expected status=active after resume, got %s", status)
	}
}

// [REQ:TM-AC-006] Manual campaign termination
func TestAutoCampaign_ManualTermination(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-scenario-terminate", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	// Start campaign
	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Terminate campaign
	err = aco.TerminateCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to terminate campaign: %v", err)
	}

	// Verify status and completed_at
	var status string
	var completedAt sql.NullTime
	err = db.QueryRow(`
		SELECT status, completed_at FROM campaigns WHERE id = $1
	`, campaign.ID).Scan(&status, &completedAt)
	if err != nil {
		t.Fatalf("failed to get campaign: %v", err)
	}

	if status != "terminated" {
		t.Errorf("expected status=terminated, got %s", status)
	}

	if !completedAt.Valid {
		t.Error("expected completed_at to be set after termination")
	}
}

// [REQ:TM-AC-007] Mark campaign as error on repeated failures
func TestAutoCampaign_ErrorThreshold(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-008] Campaign statistics tracking
func TestAutoCampaign_Statistics(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
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
