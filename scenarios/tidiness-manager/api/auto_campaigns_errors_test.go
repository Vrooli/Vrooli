package main

import (
	"fmt"
	"testing"
)

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
