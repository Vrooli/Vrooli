package plans

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/database"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the shared database surface sqliteRepository depends on.
// Both *sql.DB (repository unit tests via testutil/db.NewSQLite) and
// *database.RoutedDB (production) satisfy it.
type sqliteRepository struct {
	db    database.SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db database.SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

const planTimeFormat = time.RFC3339Nano

const (
	insertPlanSQL = `
	INSERT INTO plans (id, name, schedule, keep_latest, keep_hourly, keep_daily, keep_weekly, keep_monthly, keep_annual, enabled, protection_tier, recovery_drill_schedule, target_selector, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	insertPlanTargetSQL = `
INSERT OR IGNORE INTO plan_targets (plan_id, target_id) VALUES (?, ?)
`
	insertPlanDestSQL = `
INSERT OR IGNORE INTO plan_destinations (plan_id, destination_id) VALUES (?, ?)
`
	insertPlanExclusionSQL = `
INSERT OR IGNORE INTO plan_target_exclusions (plan_id, target_id) VALUES (?, ?)
`
	deletePlanExclusionsSQL = `DELETE FROM plan_target_exclusions WHERE plan_id = ?`
	selectPlanExclusionsSQL = `SELECT target_id FROM plan_target_exclusions WHERE plan_id = ? ORDER BY target_id ASC`
	deletePlanTargetsSQL    = `DELETE FROM plan_targets WHERE plan_id = ?`
	deletePlanDestsSQL      = `DELETE FROM plan_destinations WHERE plan_id = ?`

	updatePlanSQL = `
UPDATE plans
	SET name = ?, schedule = ?, keep_latest = ?, keep_hourly = ?, keep_daily = ?, keep_weekly = ?, keep_monthly = ?, keep_annual = ?, enabled = ?, protection_tier = ?, recovery_drill_schedule = ?, target_selector = ?, updated_at = ?
WHERE id = ?
`
	selectPlanByIDSQL = `
	SELECT id, name, schedule, keep_latest, keep_hourly, keep_daily, keep_weekly, keep_monthly, keep_annual, enabled, protection_tier, recovery_drill_schedule, target_selector, created_at, updated_at
FROM plans WHERE id = ?
`
	listPlansSQL = `
	SELECT id, name, schedule, keep_latest, keep_hourly, keep_daily, keep_weekly, keep_monthly, keep_annual, enabled, protection_tier, recovery_drill_schedule, target_selector, created_at, updated_at
FROM plans
ORDER BY name ASC
LIMIT ?
`
	selectPlanTargetsSQL = `
SELECT target_id FROM plan_targets WHERE plan_id = ? ORDER BY target_id ASC
`
	selectPlanDestsSQL = `
SELECT destination_id FROM plan_destinations WHERE plan_id = ? ORDER BY destination_id ASC
`
	deletePlanSQL = `DELETE FROM plans WHERE id = ?`
)

func (s *sqliteRepository) Create(ctx context.Context, p Plan) (Plan, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}

	enabled := 0
	if p.Enabled {
		enabled = 1
	}

	_, err := s.db.ExecContext(ctx, insertPlanSQL,
		p.ID, p.Name, p.Schedule, p.KeepLatest, p.KeepHourly, p.KeepDaily, p.KeepWeekly, p.KeepMonthly, p.KeepAnnual, enabled, string(p.ProtectionTier), p.RecoveryDrillSchedule, string(p.TargetSelector),
		p.CreatedAt.Format(planTimeFormat), p.UpdatedAt.Format(planTimeFormat),
	)
	if err != nil {
		return Plan{}, fmt.Errorf("insert plan %q: %w", p.Name, err)
	}

	if err := s.writeMembership(ctx, p.ID, p.TargetIDs, p.DestinationIDs); err != nil {
		return Plan{}, err
	}
	if err := s.writeExclusions(ctx, p.ID, p.ExcludedTargetIDs); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *sqliteRepository) Update(ctx context.Context, p Plan) (Plan, error) {
	p.UpdatedAt = s.clock.Now().UTC()

	enabled := 0
	if p.Enabled {
		enabled = 1
	}

	_, err := s.db.ExecContext(ctx, updatePlanSQL,
		p.Name, p.Schedule, p.KeepLatest, p.KeepHourly, p.KeepDaily, p.KeepWeekly, p.KeepMonthly, p.KeepAnnual, enabled, string(p.ProtectionTier), p.RecoveryDrillSchedule, string(p.TargetSelector), p.UpdatedAt.Format(planTimeFormat), p.ID,
	)
	if err != nil {
		return Plan{}, fmt.Errorf("update plan %q: %w", p.ID, err)
	}

	// Replace membership: delete then re-insert.
	if _, err := s.db.ExecContext(ctx, deletePlanTargetsSQL, p.ID); err != nil {
		return Plan{}, fmt.Errorf("delete plan_targets for %q: %w", p.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, deletePlanDestsSQL, p.ID); err != nil {
		return Plan{}, fmt.Errorf("delete plan_destinations for %q: %w", p.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, deletePlanExclusionsSQL, p.ID); err != nil {
		return Plan{}, fmt.Errorf("delete plan_target_exclusions for %q: %w", p.ID, err)
	}
	if err := s.writeMembership(ctx, p.ID, p.TargetIDs, p.DestinationIDs); err != nil {
		return Plan{}, err
	}
	if err := s.writeExclusions(ctx, p.ID, p.ExcludedTargetIDs); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *sqliteRepository) writeMembership(ctx context.Context, planID string, targetIDs, destIDs []string) error {
	for _, tid := range targetIDs {
		if _, err := s.db.ExecContext(ctx, insertPlanTargetSQL, planID, tid); err != nil {
			return fmt.Errorf("insert plan_target %q → %q: %w", planID, tid, err)
		}
	}
	for _, did := range destIDs {
		if _, err := s.db.ExecContext(ctx, insertPlanDestSQL, planID, did); err != nil {
			return fmt.Errorf("insert plan_destination %q → %q: %w", planID, did, err)
		}
	}
	return nil
}

func (s *sqliteRepository) writeExclusions(ctx context.Context, planID string, targetIDs []string) error {
	for _, tid := range targetIDs {
		if _, err := s.db.ExecContext(ctx, insertPlanExclusionSQL, planID, tid); err != nil {
			return fmt.Errorf("insert plan_target_exclusion %q → %q: %w", planID, tid, err)
		}
	}
	return nil
}

func (s *sqliteRepository) GetByID(ctx context.Context, id string) (Plan, error) {
	row := s.db.QueryRowContext(ctx, selectPlanByIDSQL, id)
	p, err := scanPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Plan{}, ErrPlanNotFound{ID: id}
	}
	if err != nil {
		return Plan{}, fmt.Errorf("get plan %q: %w", id, err)
	}
	if err := s.loadMembership(ctx, &p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (s *sqliteRepository) List(ctx context.Context, limit int) ([]Plan, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, listPlansSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	var out []Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plans: %w", err)
	}

	for i := range out {
		if err := s.loadMembership(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ListPage provides a stable name-ordered page for the list RPC.
func (s *sqliteRepository) ListPage(ctx context.Context, limit, offset int) ([]Plan, error) {
	if limit <= 0 || offset < 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, schedule, keep_latest, keep_hourly, keep_daily, keep_weekly, keep_monthly, keep_annual, enabled, protection_tier, recovery_drill_schedule, target_selector, created_at, updated_at FROM plans ORDER BY name ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list plan page: %w", err)
	}
	defer rows.Close()
	var out []Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.loadMembership(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) (bool, error) {
	// plan_targets and plan_destinations CASCADE on delete, so we only need to
	// delete the parent row.
	res, err := s.db.ExecContext(ctx, deletePlanSQL, id)
	if err != nil {
		return false, fmt.Errorf("delete plan %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete plan %q rows: %w", id, err)
	}
	return n > 0, nil
}

// loadMembership populates p.TargetIDs and p.DestinationIDs from the join tables.
func (s *sqliteRepository) loadMembership(ctx context.Context, p *Plan) error {
	tids, err := s.loadIDs(ctx, selectPlanTargetsSQL, p.ID)
	if err != nil {
		return fmt.Errorf("load plan_targets for %q: %w", p.ID, err)
	}
	dids, err := s.loadIDs(ctx, selectPlanDestsSQL, p.ID)
	if err != nil {
		return fmt.Errorf("load plan_destinations for %q: %w", p.ID, err)
	}
	excluded, err := s.loadIDs(ctx, selectPlanExclusionsSQL, p.ID)
	if err != nil {
		return fmt.Errorf("load plan_target_exclusions for %q: %w", p.ID, err)
	}
	p.TargetIDs = tids
	p.DestinationIDs = dids
	p.ExcludedTargetIDs = excluded
	return nil
}

func (s *sqliteRepository) loadIDs(ctx context.Context, query, planID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, query, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlan(sc rowScanner) (Plan, error) {
	var (
		p             Plan
		enabled       int
		tier          string
		drillSchedule string
		selector      string
		createdRaw    string
		updatedRaw    string
	)
	if err := sc.Scan(&p.ID, &p.Name, &p.Schedule, &p.KeepLatest, &p.KeepHourly, &p.KeepDaily, &p.KeepWeekly, &p.KeepMonthly, &p.KeepAnnual, &enabled, &tier, &drillSchedule, &selector, &createdRaw, &updatedRaw); err != nil {
		return Plan{}, err
	}
	p.Enabled = enabled != 0
	p.ProtectionTier = ProtectionTier(tier)
	p.RecoveryDrillSchedule = drillSchedule
	p.TargetSelector = TargetSelector(selector)
	if p.ProtectionTier == "" {
		p.ProtectionTier = TierFullPrimary
	}
	created, err := time.Parse(planTimeFormat, createdRaw)
	if err != nil {
		return Plan{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	updated, err := time.Parse(planTimeFormat, updatedRaw)
	if err != nil {
		return Plan{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	p.CreatedAt = created
	p.UpdatedAt = updated
	return p, nil
}
