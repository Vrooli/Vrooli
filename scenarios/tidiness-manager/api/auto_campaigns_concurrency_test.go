package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
)

// [REQ:TM-AC-002] Enforce maximum K concurrent campaigns
func TestAutoCampaign_ConcurrencyLimit(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignConcurrencyTest(t)
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

// [REQ:TM-AC-002] Test edge case: release concurrency slot when campaign completes
func TestAutoCampaign_ConcurrencySlotRelease(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignConcurrencyTest(t)
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

// [REQ:TM-AC-002] Test concurrent campaign count calculation accuracy
func TestAutoCampaign_ActiveCountAccuracy(t *testing.T) {
	_, aco, cleanup := setupAutoCampaignConcurrencyTest(t)
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

// setupAutoCampaignConcurrencyTest creates the test environment for concurrency tests
func setupAutoCampaignConcurrencyTest(t *testing.T) (*sql.DB, *AutoCampaignOrchestrator, func()) {
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
