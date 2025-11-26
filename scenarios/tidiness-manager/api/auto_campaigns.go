package main

import (
	"database/sql"
	"fmt"
	"time"
)

// AutoCampaignOrchestrator manages automatic tidiness campaigns across scenarios
// Implements OT-P1-001 (Auto-tidiness campaigns) and OT-P1-002 (Campaign lifecycle)
type AutoCampaignOrchestrator struct {
	db            *sql.DB
	campaignMgr   *CampaignManager
	maxConcurrent int
}

// NewAutoCampaignOrchestrator creates a new orchestrator instance
func NewAutoCampaignOrchestrator(db *sql.DB, campaignMgr *CampaignManager) (*AutoCampaignOrchestrator, error) {
	// Get max_concurrent from config table
	var maxConcurrent int
	err := db.QueryRow("SELECT value FROM config WHERE key = 'max_concurrent_campaigns'").Scan(&maxConcurrent)
	if err != nil {
		maxConcurrent = 3 // Default fallback
	}

	return &AutoCampaignOrchestrator{
		db:            db,
		campaignMgr:   campaignMgr,
		maxConcurrent: maxConcurrent,
	}, nil
}

// AutoCampaign represents an auto-tidiness campaign
type AutoCampaign struct {
	ID                       int            `json:"id"`
	Scenario                 string         `json:"scenario"`
	Status                   string         `json:"status"`
	MaxSessions              int            `json:"max_sessions"`
	MaxFilesPerSession       int            `json:"max_files_per_session"`
	CurrentSession           int            `json:"current_session"`
	FilesVisited             int            `json:"files_visited"`
	FilesTotal               int            `json:"files_total"`
	ErrorCount               int            `json:"error_count"`
	ErrorReason              sql.NullString `json:"error_reason,omitempty"`
	VisitedTrackerCampaignID sql.NullInt64  `json:"visited_tracker_campaign_id,omitempty"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	CompletedAt              *time.Time     `json:"completed_at,omitempty"`
}

// CreateAutoCampaign creates a new auto-tidiness campaign [REQ:TM-AC-001]
func (aco *AutoCampaignOrchestrator) CreateAutoCampaign(scenario string, maxSessions, maxFilesPerSession int) (*AutoCampaign, error) {
	// Check concurrency limit [REQ:TM-AC-002]
	activeCampaigns, err := aco.GetActiveCampaignCount()
	if err != nil {
		return nil, fmt.Errorf("failed to check active campaigns: %w", err)
	}

	if activeCampaigns >= aco.maxConcurrent {
		return nil, fmt.Errorf("max concurrent campaigns (%d) reached", aco.maxConcurrent)
	}

	// Create visited-tracker campaign
	vtCampaign, err := aco.campaignMgr.GetOrCreateCampaign(scenario)
	var vtCampaignID int
	if err == nil && vtCampaign != nil {
		// Parse campaign ID (assuming it's numeric)
		fmt.Sscanf(vtCampaign.ID, "%d", &vtCampaignID)
	}

	// Insert campaign record
	query := `
		INSERT INTO campaigns (
			scenario, status, max_sessions, max_files_per_session,
			visited_tracker_campaign_id, files_total
		) VALUES ($1, $2, $3, $4, $5, 0)
		RETURNING id, created_at, updated_at
	`

	campaign := &AutoCampaign{
		Scenario:           scenario,
		Status:             "created",
		MaxSessions:        maxSessions,
		MaxFilesPerSession: maxFilesPerSession,
	}

	if vtCampaignID > 0 {
		err = aco.db.QueryRow(query, scenario, "created", maxSessions, maxFilesPerSession, vtCampaignID).
			Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
	} else {
		err = aco.db.QueryRow(query, scenario, "created", maxSessions, maxFilesPerSession, nil).
			Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	if vtCampaignID > 0 {
		campaign.VisitedTrackerCampaignID = sql.NullInt64{Int64: int64(vtCampaignID), Valid: true}
	}
	return campaign, nil
}

// GetActiveCampaignCount returns the number of active/paused campaigns [REQ:TM-AC-002]
func (aco *AutoCampaignOrchestrator) GetActiveCampaignCount() (int, error) {
	var count int
	err := aco.db.QueryRow(`
		SELECT COUNT(*) FROM campaigns
		WHERE status IN ('created', 'active', 'paused')
	`).Scan(&count)
	return count, err
}

// StartCampaign transitions a campaign from 'created' to 'active' [REQ:TM-AC-001]
func (aco *AutoCampaignOrchestrator) StartCampaign(campaignID int) error {
	_, err := aco.db.Exec(`
		UPDATE campaigns
		SET status = 'active', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'created'
	`, campaignID)
	return err
}

// PauseCampaign pauses an active campaign [REQ:TM-AC-005]
func (aco *AutoCampaignOrchestrator) PauseCampaign(campaignID int) error {
	_, err := aco.db.Exec(`
		UPDATE campaigns
		SET status = 'paused', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'active'
	`, campaignID)
	return err
}

// ResumeCampaign resumes a paused campaign [REQ:TM-AC-005]
func (aco *AutoCampaignOrchestrator) ResumeCampaign(campaignID int) error {
	_, err := aco.db.Exec(`
		UPDATE campaigns
		SET status = 'active', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'paused'
	`, campaignID)
	return err
}

// TerminateCampaign manually terminates a campaign [REQ:TM-AC-006]
func (aco *AutoCampaignOrchestrator) TerminateCampaign(campaignID int) error {
	_, err := aco.db.Exec(`
		UPDATE campaigns
		SET status = 'terminated', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status IN ('created', 'active', 'paused')
	`, campaignID)
	return err
}

// CheckAndAutoComplete checks if a campaign should auto-complete [REQ:TM-AC-003, TM-AC-004]
func (aco *AutoCampaignOrchestrator) CheckAndAutoComplete(campaignID int) error {
	// Get campaign details
	var campaign AutoCampaign
	err := aco.db.QueryRow(`
		SELECT id, scenario, status, max_sessions, current_session,
			files_visited, files_total, visited_tracker_campaign_id
		FROM campaigns WHERE id = $1
	`, campaignID).Scan(
		&campaign.ID, &campaign.Scenario, &campaign.Status,
		&campaign.MaxSessions, &campaign.CurrentSession,
		&campaign.FilesVisited, &campaign.FilesTotal,
		&campaign.VisitedTrackerCampaignID,
	)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}

	// Only check active campaigns
	if campaign.Status != "active" {
		return nil
	}

	shouldComplete := false
	reason := ""

	// Check max sessions [REQ:TM-AC-004]
	if campaign.CurrentSession >= campaign.MaxSessions {
		shouldComplete = true
		reason = "max_sessions_reached"
	}

	// Check if all files visited [REQ:TM-AC-003]
	if campaign.VisitedTrackerCampaignID.Valid && campaign.VisitedTrackerCampaignID.Int64 > 0 && campaign.FilesTotal > 0 {
		if campaign.FilesVisited >= campaign.FilesTotal {
			shouldComplete = true
			reason = "all_files_visited"
		}
	}

	if shouldComplete {
		_, err := aco.db.Exec(`
			UPDATE campaigns
			SET status = 'completed', completed_at = CURRENT_TIMESTAMP,
				updated_at = CURRENT_TIMESTAMP, error_reason = $2
			WHERE id = $1
		`, campaignID, reason)
		return err
	}

	return nil
}

// RecordCampaignSession increments session count and checks for auto-completion
func (aco *AutoCampaignOrchestrator) RecordCampaignSession(campaignID int, filesProcessed int) error {
	// Increment session count
	_, err := aco.db.Exec(`
		UPDATE campaigns
		SET current_session = current_session + 1,
			files_visited = files_visited + $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, campaignID, filesProcessed)
	if err != nil {
		return fmt.Errorf("failed to update campaign session: %w", err)
	}

	// Check for auto-completion
	return aco.CheckAndAutoComplete(campaignID)
}

// RecordCampaignError increments error count and marks as 'error' if threshold exceeded [REQ:TM-AC-007]
func (aco *AutoCampaignOrchestrator) RecordCampaignError(campaignID int, errorReason string) error {
	// Increment error count
	result, err := aco.db.Exec(`
		UPDATE campaigns
		SET error_count = error_count + 1,
			error_reason = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, campaignID, errorReason)
	if err != nil {
		return fmt.Errorf("failed to record error: %w", err)
	}

	// Get updated error count
	var errorCount int
	err = aco.db.QueryRow(`
		SELECT error_count FROM campaigns WHERE id = $1
	`, campaignID).Scan(&errorCount)
	if err != nil {
		return fmt.Errorf("failed to get error count: %w", err)
	}

	// Mark as error if threshold exceeded (3 consecutive errors)
	if errorCount >= 3 {
		_, err = aco.db.Exec(`
			UPDATE campaigns
			SET status = 'error', completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`, campaignID)
		return err
	}

	_, _ = result.RowsAffected() // Ignore unused result
	return nil
}

// GetCampaign retrieves a campaign by ID
func (aco *AutoCampaignOrchestrator) GetCampaign(campaignID int) (*AutoCampaign, error) {
	campaign := &AutoCampaign{}
	var completedAt sql.NullTime

	err := aco.db.QueryRow(`
		SELECT id, scenario, status, max_sessions, max_files_per_session,
			current_session, files_visited, files_total, error_count, error_reason,
			visited_tracker_campaign_id, created_at, updated_at, completed_at
		FROM campaigns WHERE id = $1
	`, campaignID).Scan(
		&campaign.ID, &campaign.Scenario, &campaign.Status,
		&campaign.MaxSessions, &campaign.MaxFilesPerSession,
		&campaign.CurrentSession, &campaign.FilesVisited, &campaign.FilesTotal,
		&campaign.ErrorCount, &campaign.ErrorReason,
		&campaign.VisitedTrackerCampaignID,
		&campaign.CreatedAt, &campaign.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		campaign.CompletedAt = &completedAt.Time
	}

	return campaign, nil
}

// ListCampaigns retrieves all campaigns with optional status filter
func (aco *AutoCampaignOrchestrator) ListCampaigns(statusFilter string) ([]*AutoCampaign, error) {
	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		rows, err = aco.db.Query(`
			SELECT id, scenario, status, max_sessions, max_files_per_session,
				current_session, files_visited, files_total, error_count, error_reason,
				visited_tracker_campaign_id, created_at, updated_at, completed_at
			FROM campaigns WHERE status = $1
			ORDER BY created_at DESC
		`, statusFilter)
	} else {
		rows, err = aco.db.Query(`
			SELECT id, scenario, status, max_sessions, max_files_per_session,
				current_session, files_visited, files_total, error_count, error_reason,
				visited_tracker_campaign_id, created_at, updated_at, completed_at
			FROM campaigns
			ORDER BY created_at DESC
		`)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := []*AutoCampaign{}
	for rows.Next() {
		campaign := &AutoCampaign{}
		var completedAt sql.NullTime

		err := rows.Scan(
			&campaign.ID, &campaign.Scenario, &campaign.Status,
			&campaign.MaxSessions, &campaign.MaxFilesPerSession,
			&campaign.CurrentSession, &campaign.FilesVisited, &campaign.FilesTotal,
			&campaign.ErrorCount, &campaign.ErrorReason,
			&campaign.VisitedTrackerCampaignID,
			&campaign.CreatedAt, &campaign.UpdatedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			campaign.CompletedAt = &completedAt.Time
		}

		campaigns = append(campaigns, campaign)
	}

	return campaigns, rows.Err()
}
