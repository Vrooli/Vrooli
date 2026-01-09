package main

import (
	"database/sql"
	"os"
	"testing"
)

// [REQ:TM-AC-005] Campaign pause and resume
func TestAutoCampaign_PauseResume(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignLifecycleTest(t)
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
	db, aco, cleanup := setupAutoCampaignLifecycleTest(t)
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

// [REQ:TM-AC-005] Test edge case: pause already paused campaign
func TestAutoCampaign_PauseAlreadyPaused(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignLifecycleTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-double-pause", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Pause once
	err = aco.PauseCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to pause campaign: %v", err)
	}

	// Pause again - should be idempotent (no error, but no effect)
	err = aco.PauseCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("unexpected error on double pause: %v", err)
	}

	// Verify still paused
	var status string
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "paused" {
		t.Errorf("expected status=paused after double pause, got %s", status)
	}
}

// [REQ:TM-AC-005] Test edge case: resume campaign that was never paused
func TestAutoCampaign_ResumeNonPaused(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignLifecycleTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-bad-resume", 10, 5)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Try to resume an active campaign (should have no effect)
	err = aco.ResumeCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("unexpected error on resume of active campaign: %v", err)
	}

	// Verify still active (not changed)
	var status string
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "active" {
		t.Errorf("expected status=active after resume of active campaign, got %s", status)
	}
}

// [REQ:TM-AC-006] Test edge case: terminate already completed campaign
func TestAutoCampaign_TerminateCompleted(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignLifecycleTest(t)
	defer cleanup()

	campaign, err := aco.CreateAutoCampaign("test-terminate-completed", 1, 1)
	if err != nil {
		t.Fatalf("failed to create campaign: %v", err)
	}

	err = aco.StartCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("failed to start campaign: %v", err)
	}

	// Auto-complete by reaching max sessions
	err = aco.RecordCampaignSession(campaign.ID, 1)
	if err != nil {
		t.Fatalf("failed to record session: %v", err)
	}

	// Try to terminate completed campaign (should have no effect)
	err = aco.TerminateCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("unexpected error terminating completed campaign: %v", err)
	}

	// Verify status remains completed (not changed to terminated)
	var status string
	err = db.QueryRow("SELECT status FROM campaigns WHERE id = $1", campaign.ID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status != "completed" {
		t.Errorf("expected status to remain 'completed', got %s", status)
	}
}

// setupAutoCampaignLifecycleTest creates the test environment for lifecycle tests
func setupAutoCampaignLifecycleTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
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
