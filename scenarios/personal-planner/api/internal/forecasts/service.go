package forecasts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type service struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewService(db SQLExecutor, clock schedule.Clock) Service { return &service{db: db, clock: clock} }

func (s *service) Get(ctx context.Context, localDate, timezone string, horizonDays int) (Forecast, error) {
	if _, err := time.Parse("2006-01-02", localDate); err != nil {
		return Forecast{}, fmt.Errorf("local_date must be YYYY-MM-DD")
	}
	if horizonDays == 0 {
		horizonDays = 28
	}
	if horizonDays < 1 || horizonDays > 90 {
		return Forecast{}, fmt.Errorf("horizon_days must be between 1 and 90")
	}
	var known, unresolved, occupied int64
	// The current work schema represents unknown effort outside this table; a
	// zero remaining value is completed work, not an unknown estimate. Keep the
	// unresolved count honest until the effort-range contract is available.
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN remaining_minutes > 0 THEN remaining_minutes ELSE 0 END),0), 0 FROM work_items`).Scan(&known, &unresolved); err != nil {
		return Forecast{}, fmt.Errorf("read work demand: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(duration_minutes),0) FROM calendar_allocations WHERE state='accepted' AND local_date>=? AND local_date<=date(?, '+' || (? - 1) || ' days')`, localDate, localDate, horizonDays).Scan(&occupied); err != nil {
		return Forecast{}, fmt.Errorf("read accepted capacity: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,result,promised_boundary FROM commitments WHERE state IN ('proposed','active') ORDER BY promised_boundary`)
	if err != nil {
		return Forecast{}, fmt.Errorf("read commitments: %w", err)
	}
	defer rows.Close()
	commitments := []CommitmentInput{}
	for rows.Next() {
		var x CommitmentInput
		if err := rows.Scan(&x.ID, &x.Result, &x.PromisedBoundary); err != nil {
			return Forecast{}, fmt.Errorf("scan commitment: %w", err)
		}
		commitments = append(commitments, x)
	}
	if err := rows.Err(); err != nil {
		return Forecast{}, fmt.Errorf("read commitments: %w", err)
	}
	x, err := Build(Snapshot{StartDate: localDate, Timezone: timezone, HorizonDays: horizonDays, KnownWork: known, UnresolvedWork: unresolved, OccupiedMinutes: occupied, Commitments: commitments}, s.clock.Now())
	if err != nil {
		return Forecast{}, err
	}
	return s.persist(ctx, x)
}

func (s *service) List(ctx context.Context, limit int) ([]SnapshotSummary, error) {
	if limit == 0 {
		limit = 8
	}
	if limit < 1 || limit > 20 {
		return nil, fmt.Errorf("limit must be between 1 and 20")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id,s.generated_at,s.input_fingerprint,s.horizon_start,s.horizon_end,
		       s.central_finish,s.cautious_finish,s.result_state,s.risk_state,s.explanation,
		       COALESCE(c.previous_snapshot_id,''),COALESCE(c.explanation,'')
		FROM forecast_snapshots s
		LEFT JOIN forecast_change_records c ON c.snapshot_id=s.id
		WHERE s.subject=?
		ORDER BY s.created_at DESC
		LIMIT ?`, "workspace", limit)
	if err != nil {
		return nil, fmt.Errorf("list forecast snapshots: %w", err)
	}
	defer rows.Close()
	items := make([]SnapshotSummary, 0, limit)
	for rows.Next() {
		var x SnapshotSummary
		if err := rows.Scan(&x.ID, &x.GeneratedAt, &x.InputFingerprint, &x.HorizonStart, &x.HorizonEnd, &x.CentralFinish, &x.CautiousFinish, &x.ResultState, &x.RiskState, &x.Explanation, &x.PreviousSnapshotID, &x.ChangeExplanation); err != nil {
			return nil, fmt.Errorf("scan forecast snapshot: %w", err)
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read forecast snapshots: %w", err)
	}
	return items, nil
}

type storedSnapshot struct {
	ID, Central, Cautious, State, Risk string
}

func (s *service) persist(ctx context.Context, x Forecast) (Forecast, error) {
	const subject = "workspace"
	var existing storedSnapshot
	err := s.db.QueryRowContext(ctx, `SELECT id,central_finish,cautious_finish,result_state,risk_state FROM forecast_snapshots WHERE subject=? AND input_fingerprint=?`, subject, x.InputFingerprint).Scan(&existing.ID, &existing.Central, &existing.Cautious, &existing.State, &existing.Risk)
	if err == nil {
		x.SnapshotID = existing.ID
		return x, nil
	}
	if err != sql.ErrNoRows {
		return Forecast{}, fmt.Errorf("read forecast snapshot: %w", err)
	}
	var previous storedSnapshot
	previousErr := s.db.QueryRowContext(ctx, `SELECT id,central_finish,cautious_finish,result_state,risk_state FROM forecast_snapshots WHERE subject=? ORDER BY created_at DESC LIMIT 1`, subject).Scan(&previous.ID, &previous.Central, &previous.Cautious, &previous.State, &previous.Risk)
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	id := uuid.NewString()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO forecast_snapshots (id,subject,input_fingerprint,generated_at,horizon_start,horizon_end,central_finish,cautious_finish,result_state,risk_state,explanation,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, id, subject, x.InputFingerprint, x.GeneratedAt, x.HorizonStart, x.HorizonEnd, x.CentralFinish, x.CautiousFinish, x.ResultState, x.RiskState, x.Explanation, now); err != nil {
		return Forecast{}, fmt.Errorf("store forecast snapshot: %w", err)
	}
	x.SnapshotID = id
	if previousErr == nil {
		x.PreviousSnapshotID = previous.ID
		if previous.Central != x.CentralFinish || previous.Cautious != x.CautiousFinish || previous.State != x.ResultState || previous.Risk != x.RiskState {
			x.ChangeExplanation = fmt.Sprintf("Outlook changed from central %s / cautious %s (%s) to central %s / cautious %s (%s).", value(previous.Central), value(previous.Cautious), previous.Risk, value(x.CentralFinish), value(x.CautiousFinish), x.RiskState)
			if _, err := s.db.ExecContext(ctx, `INSERT INTO forecast_change_records (id,subject,previous_snapshot_id,snapshot_id,explanation,created_at) VALUES (?,?,?,?,?,?)`, uuid.NewString(), subject, previous.ID, id, x.ChangeExplanation, now); err != nil {
				return Forecast{}, fmt.Errorf("store forecast change: %w", err)
			}
		}
	} else if previousErr != sql.ErrNoRows {
		return Forecast{}, fmt.Errorf("read previous forecast snapshot: %w", previousErr)
	}
	return x, nil
}

func value(v string) string {
	if v == "" {
		return "not available"
	}
	return v
}
