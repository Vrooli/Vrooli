package main

import (
	"database/sql"
	"testing"
)

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
