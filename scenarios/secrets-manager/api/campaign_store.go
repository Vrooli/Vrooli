package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DBCampaignStore persists campaigns to Postgres so the UI can resume work reliably.
type DBCampaignStore struct {
	db *sql.DB
}

func NewDBCampaignStore(db *sql.DB) *DBCampaignStore {
	return &DBCampaignStore{db: db}
}

func (s *DBCampaignStore) List(ctx context.Context, scenarioFilter string) ([]CampaignSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, scenario, tier, status, progress, blockers, next_action, last_step, summary, updated_at
		FROM deployment_campaigns
		WHERE ($1 = '' OR LOWER(scenario) = LOWER($1))
		ORDER BY scenario, tier
	`, scenarioFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []CampaignSummary
	for rows.Next() {
		var (
			id         string
			scenario   string
			tier       string
			status     string
			progress   int
			blockers   int
			nextAction sql.NullString
			lastStep   sql.NullString
			summaryRaw []byte
			updatedAt  time.Time
		)
		if err := rows.Scan(&id, &scenario, &tier, &status, &progress, &blockers, &nextAction, &lastStep, &summaryRaw, &updatedAt); err != nil {
			return nil, err
		}
		var summary *DeploymentSummary
		if len(summaryRaw) > 0 && string(summaryRaw) != "null" {
			var parsed DeploymentSummary
			if err := json.Unmarshal(summaryRaw, &parsed); err == nil {
				summary = &parsed
			}
		}
		campaigns = append(campaigns, CampaignSummary{
			ID:         id,
			Scenario:   scenario,
			Tier:       tier,
			Status:     status,
			Progress:   progress,
			Blockers:   blockers,
			NextAction: strings.TrimSpace(nextAction.String),
			LastStep:   strings.TrimSpace(lastStep.String),
			Summary:    summary,
			UpdatedAt:  updatedAt,
		})
	}
	return campaigns, rows.Err()
}

func (s *DBCampaignStore) Upsert(ctx context.Context, campaign CampaignSummary) error {
	var summaryJSON any
	if campaign.Summary != nil {
		raw, err := json.Marshal(campaign.Summary)
		if err != nil {
			return fmt.Errorf("marshal summary: %w", err)
		}
		summaryJSON = raw
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO deployment_campaigns (id, scenario, tier, status, progress, blockers, next_action, last_step, summary, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9, COALESCE($10, NOW()))
		ON CONFLICT (scenario, tier) DO UPDATE SET
			status = EXCLUDED.status,
			progress = EXCLUDED.progress,
			blockers = EXCLUDED.blockers,
			next_action = EXCLUDED.next_action,
			last_step = EXCLUDED.last_step,
			summary = EXCLUDED.summary,
			updated_at = NOW()
	`, campaign.ID, campaign.Scenario, campaign.Tier, campaign.Status, campaign.Progress, campaign.Blockers, campaign.NextAction, campaign.LastStep, summaryJSON, campaign.UpdatedAt)
	return err
}
