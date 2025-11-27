package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
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

	if !strings.Contains(err.Error(), "max concurrent campaigns") {
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

// [REQ:TM-AC-001] Test edge case: zero or negative session limits
func TestAutoCampaign_InvalidSessionLimits(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
	defer cleanup()

	// Zero max sessions
	campaign1, err := aco.CreateAutoCampaign("test-zero-sessions", 0, 5)
	if err != nil {
		t.Fatalf("failed to create campaign with zero sessions: %v", err)
	}
	if campaign1.MaxSessions != 0 {
		t.Errorf("expected max_sessions=0 to be accepted, got %d", campaign1.MaxSessions)
	}

	// Negative max sessions
	campaign2, err := aco.CreateAutoCampaign("test-negative-sessions", -5, 5)
	if err != nil {
		t.Fatalf("failed to create campaign with negative sessions: %v", err)
	}
	if campaign2.MaxSessions != -5 {
		t.Errorf("expected max_sessions=-5 to be stored, got %d", campaign2.MaxSessions)
	}

	// Zero files per session
	campaign3, err := aco.CreateAutoCampaign("test-zero-files", 10, 0)
	if err != nil {
		t.Fatalf("failed to create campaign with zero files: %v", err)
	}
	if campaign3.MaxFilesPerSession != 0 {
		t.Errorf("expected max_files_per_session=0 to be accepted, got %d", campaign3.MaxFilesPerSession)
	}
}

// [REQ:TM-AC-002] Test edge case: release concurrency slot when campaign completes
func TestAutoCampaign_ConcurrencySlotRelease(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
	defer cleanup()

	// Fill all 3 slots
	var campaigns []*AutoCampaign
	for i := 0; i < 3; i++ {
		c, err := aco.CreateAutoCampaign(fmt.Sprintf("test-slot-%d", i), 1, 5)
		if err != nil {
			t.Fatalf("failed to create campaign %d: %v", i, err)
		}
		err = aco.StartCampaign(c.ID)
		if err != nil {
			t.Fatalf("failed to start campaign %d: %v", i, err)
		}
		campaigns = append(campaigns, c)
	}

	// Verify we're at capacity
	_, err := aco.CreateAutoCampaign("test-overflow", 10, 5)
	if err == nil {
		t.Error("expected error when at capacity, got nil")
	}

	// Complete one campaign
	err = aco.TerminateCampaign(campaigns[0].ID)
	if err != nil {
		t.Fatalf("failed to terminate campaign: %v", err)
	}

	// Should now be able to create a new one
	newCampaign, err := aco.CreateAutoCampaign("test-released-slot", 10, 5)
	if err != nil {
		t.Errorf("expected to create campaign after slot release, got error: %v", err)
	}
	if newCampaign == nil {
		t.Error("expected valid campaign after slot release")
	}
}

// [REQ:TM-AC-003] [REQ:TM-AC-004] Test edge case: check auto-complete doesn't trigger prematurely
func TestAutoCampaign_NoPreemptiveCompletion(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-005] Test edge case: pause already paused campaign
func TestAutoCampaign_PauseAlreadyPaused(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
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
	db, aco, cleanup := setupAutoCampaignTest(t)
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
	db, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-007] Test edge case: error threshold exactly at limit
func TestAutoCampaign_ErrorThresholdBoundary(t *testing.T) {
	db, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-008] Test edge case: statistics with zero sessions
func TestAutoCampaign_ZeroSessionStats(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-001] [REQ:TM-AC-008] Test session recording with zero files
func TestAutoCampaign_SessionWithZeroFiles(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
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

// [REQ:TM-AC-002] Test concurrent campaign count calculation accuracy
func TestAutoCampaign_ActiveCountAccuracy(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
	defer cleanup()

	// Create campaigns in different states
	c1, _ := aco.CreateAutoCampaign("test-created", 10, 5)
	c2, _ := aco.CreateAutoCampaign("test-active", 10, 5)
	c3, _ := aco.CreateAutoCampaign("test-paused", 10, 5)

	// Transition to different states
	aco.StartCampaign(c2.ID)
	aco.StartCampaign(c3.ID)
	aco.PauseCampaign(c3.ID)

	// Count should include created, active, and paused
	count, err := aco.GetActiveCampaignCount()
	if err != nil {
		t.Fatalf("failed to get active count: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 active campaigns (created+active+paused), got %d", count)
	}

	// Complete one
	aco.TerminateCampaign(c1.ID)

	// Count should decrease
	count, err = aco.GetActiveCampaignCount()
	if err != nil {
		t.Fatalf("failed to get active count after termination: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 active campaigns after termination, got %d", count)
	}
}
