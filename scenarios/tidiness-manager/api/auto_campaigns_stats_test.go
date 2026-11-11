package main

import (
	"testing"
)

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

// [REQ:TM-AC-008] Test session recording with zero files
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
