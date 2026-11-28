package campaign

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
)

// Repository SQL access is owned by API Core.

const campaignTimeFormat = time.RFC3339Nano

type sqliteRepository struct {
	db    database.SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db database.SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const (
	insertCampaignSQL = `
INSERT INTO campaigns (id, scenario, name, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`

	selectCampaignSQL = `
SELECT id, scenario, name, status, created_at, updated_at
FROM campaigns WHERE id = ?`

	listCampaignsSQL = `
SELECT id, scenario, name, status, created_at, updated_at
FROM campaigns
ORDER BY created_at DESC, id DESC`

	listCampaignsByScenarioSQL = `
SELECT id, scenario, name, status, created_at, updated_at
FROM campaigns WHERE scenario = ?
ORDER BY created_at DESC, id DESC`

	updateCampaignStatusSQL = `
UPDATE campaigns SET status = ?, updated_at = ? WHERE id = ?`

	insertItemSQL = `
INSERT INTO campaign_items
  (campaign_id, stable_id, scenario, source, code, severity, locations, domains,
   message, suggestion, status, resolution_note, regressed, effort, first_seen_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	transitionItemSQL = `
UPDATE campaign_items SET
  scenario = ?, source = ?, code = ?, severity = ?, locations = ?, domains = ?,
  message = ?, suggestion = ?, status = ?, resolution_note = ?, regressed = ?,
  effort = ?, updated_at = ?
WHERE campaign_id = ? AND stable_id = ? AND status = ?`

	selectItemSQL = `
SELECT campaign_id, stable_id, scenario, source, code, severity, locations, domains,
       message, suggestion, status, resolution_note, regressed, effort, first_seen_at, updated_at
FROM campaign_items WHERE campaign_id = ? AND stable_id = ?`

	listItemsSQL = `
SELECT campaign_id, stable_id, scenario, source, code, severity, locations, domains,
       message, suggestion, status, resolution_note, regressed, effort, first_seen_at, updated_at
FROM campaign_items WHERE campaign_id = ?
ORDER BY severity, code, stable_id`
)

func (r *sqliteRepository) now() time.Time { return r.clock.Now().UTC() }

func (r *sqliteRepository) CreateCampaign(ctx context.Context, c Campaign) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = r.now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	_, err := r.db.ExecContext(ctx, insertCampaignSQL,
		c.ID, c.Scenario, c.Name, string(c.Status),
		c.CreatedAt.Format(campaignTimeFormat), c.UpdatedAt.Format(campaignTimeFormat))
	return err
}

func (r *sqliteRepository) GetCampaign(ctx context.Context, id string) (Campaign, error) {
	row := r.db.QueryRowContext(ctx, selectCampaignSQL, id)
	var (
		c                    Campaign
		status               string
		createdAt, updatedAt string
	)
	if err := row.Scan(&c.ID, &c.Scenario, &c.Name, &status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Campaign{}, ErrCampaignNotFound{ID: id}
		}
		return Campaign{}, err
	}
	c.Status = CampaignLifecycle(status)
	c.CreatedAt, _ = time.Parse(campaignTimeFormat, createdAt)
	c.UpdatedAt, _ = time.Parse(campaignTimeFormat, updatedAt)
	return c, nil
}

func (r *sqliteRepository) ListCampaigns(ctx context.Context, scenario string) ([]Campaign, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if scenario == "" {
		rows, err = r.db.QueryContext(ctx, listCampaignsSQL)
	} else {
		rows, err = r.db.QueryContext(ctx, listCampaignsByScenarioSQL, scenario)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Campaign
	for rows.Next() {
		var (
			c                    Campaign
			status               string
			createdAt, updatedAt string
		)
		if err := rows.Scan(&c.ID, &c.Scenario, &c.Name, &status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		c.Status = CampaignLifecycle(status)
		c.CreatedAt, _ = time.Parse(campaignTimeFormat, createdAt)
		c.UpdatedAt, _ = time.Parse(campaignTimeFormat, updatedAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) UpdateCampaignStatus(ctx context.Context, id string, status CampaignLifecycle) error {
	res, err := r.db.ExecContext(ctx, updateCampaignStatusSQL, string(status), r.now().Format(campaignTimeFormat), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCampaignNotFound{ID: id}
	}
	return nil
}

func (r *sqliteRepository) CreateFinding(ctx context.Context, campaignID string, f Finding) error {
	if f.Status != StatusDetected {
		return ErrInvalidFindingTransition{From: "", To: f.Status}
	}
	args, err := findingWriteArgs(r, f)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, insertItemSQL, append([]any{campaignID}, args...)...)
	return err
}

func (r *sqliteRepository) TransitionFinding(ctx context.Context, campaignID string, expected FindingStatus, f Finding) error {
	if f.Status == expected || !CanTransitionFindingStatus(expected, f.Status) {
		return ErrInvalidFindingTransition{From: expected, To: f.Status}
	}
	args, err := findingWriteArgs(r, f)
	if err != nil {
		return err
	}
	transitionArgs := append(append([]any{}, args[1:13]...), args[14])
	transitionArgs = append(transitionArgs, campaignID, f.StableID, string(expected))
	res, err := r.db.ExecContext(ctx, transitionItemSQL, transitionArgs...)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrFindingWriteConflict{CampaignID: campaignID, StableID: f.StableID, Expected: expected}
	}
	return nil
}

func findingWriteArgs(r *sqliteRepository, f Finding) ([]any, error) {
	if !ValidFindingStatus(f.Status) {
		return nil, ErrInvalidFindingTransition{From: "", To: f.Status}
	}
	if f.FirstSeenAt.IsZero() {
		f.FirstSeenAt = r.now()
	}
	f.UpdatedAt = r.now()
	if f.Effort == "" {
		f.Effort = EffortUnspecified
	}
	locs, err := json.Marshal(nonNil(f.Locations))
	if err != nil {
		return nil, err
	}
	doms, err := json.Marshal(nonNil(f.Domains))
	if err != nil {
		return nil, err
	}
	return []any{
		f.StableID, f.Scenario, f.Source, f.Code, f.Severity, string(locs), string(doms),
		f.Message, f.Suggestion, string(f.Status), f.ResolutionNote, boolToInt(f.Regressed),
		f.Effort, f.FirstSeenAt.Format(campaignTimeFormat), f.UpdatedAt.Format(campaignTimeFormat),
	}, nil
}

func (r *sqliteRepository) GetFinding(ctx context.Context, campaignID, stableID string) (Finding, error) {
	row := r.db.QueryRowContext(ctx, selectItemSQL, campaignID, stableID)
	f, err := scanFinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Finding{}, ErrFindingNotFound{CampaignID: campaignID, StableID: stableID}
		}
		return Finding{}, err
	}
	return f, nil
}

func (r *sqliteRepository) ListFindings(ctx context.Context, campaignID string) ([]Finding, error) {
	rows, err := r.db.QueryContext(ctx, listItemsSQL, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Finding
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// rowScanner unifies *sql.Row and *sql.Rows for scanFinding.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanFinding(row rowScanner) (Finding, error) {
	var (
		f                      Finding
		campaignID             string
		locs, doms             string
		status                 string
		regressed              int
		firstSeenAt, updatedAt string
	)
	if err := row.Scan(&campaignID, &f.StableID, &f.Scenario, &f.Source, &f.Code, &f.Severity,
		&locs, &doms, &f.Message, &f.Suggestion, &status, &f.ResolutionNote, &regressed, &f.Effort,
		&firstSeenAt, &updatedAt); err != nil {
		return Finding{}, err
	}
	_ = json.Unmarshal([]byte(locs), &f.Locations)
	_ = json.Unmarshal([]byte(doms), &f.Domains)
	f.Status = FindingStatus(status)
	if !ValidFindingStatus(f.Status) {
		return Finding{}, ErrInvalidFindingStatus{Status: f.Status}
	}
	f.Regressed = regressed != 0
	if f.Effort == "" {
		f.Effort = EffortUnspecified
	}
	f.FirstSeenAt, _ = time.Parse(campaignTimeFormat, firstSeenAt)
	f.UpdatedAt, _ = time.Parse(campaignTimeFormat, updatedAt)
	return f, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
