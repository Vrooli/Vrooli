package main

import (
	"testing"
)

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

// [REQ:TM-AC-001] Test edge case: zero or negative session limits
func TestAutoCampaign_InvalidSessionLimits(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignTest(t)
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
