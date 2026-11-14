package main

import (
	"database/sql"
	"testing"
)

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
