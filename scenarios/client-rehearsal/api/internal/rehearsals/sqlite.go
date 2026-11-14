package rehearsals

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }
func (r *sqliteRepository) Create(ctx context.Context, item Rehearsal) (Rehearsal, error) {
	trace, err := json.Marshal(item.Trace)
	if err != nil {
		return Rehearsal{}, fmt.Errorf("encode rehearsal trace: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO rehearsals (id,title,source_format,source_sha256,credential_replacements,simulator_profile,simulator_source_sha256,state,trace_json,fault_summary,repair_summary,handoff_markdown,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.ID, item.Title, item.SourceFormat, item.SourceSHA256, item.CredentialReplacements, item.SimulatorProfile, item.SimulatorSourceSHA256, item.State, string(trace), item.FaultSummary, item.RepairSummary, item.HandoffMarkdown, item.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Rehearsal{}, fmt.Errorf("insert rehearsal %q: %w", item.ID, err)
	}
	return item, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Rehearsal, error) {
	item, err := scan(r.db.QueryRowContext(ctx, selectRehearsal+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Rehearsal{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return Rehearsal{}, fmt.Errorf("get rehearsal %q: %w", id, err)
	}
	return item, nil
}

func (r *sqliteRepository) List(ctx context.Context, limit int) ([]Rehearsal, error) {
	rows, err := r.db.QueryContext(ctx, selectRehearsal+` ORDER BY created_at DESC,id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list rehearsals: %w", err)
	}
	defer rows.Close()
	items := make([]Rehearsal, 0)
	for rows.Next() {
		item, e := scan(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rehearsals: %w", err)
	}
	return items, nil
}

func (r *sqliteRepository) RecordReview(ctx context.Context, id string, decision Decision, note string, at time.Time) (Rehearsal, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE rehearsals SET state=?,review_decision=?,review_note=?,reviewed_at=? WHERE id=? AND state=?`, stateForDecision(decision), decision, note, at.Format(time.RFC3339Nano), id, StateNeedsReview)
	if err != nil {
		return Rehearsal{}, fmt.Errorf("record review: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Rehearsal{}, fmt.Errorf("read review result: %w", err)
	}
	if n == 0 {
		if _, getErr := r.Get(ctx, id); getErr != nil {
			return Rehearsal{}, getErr
		}
		return Rehearsal{}, ErrConflict{Reason: "review already recorded; create a new workflow revision to change the decision"}
	}
	return r.Get(ctx, id)
}

func stateForDecision(d Decision) State {
	if d == DecisionApprove {
		return StateApproved
	}
	return StateChangesRequested
}

const selectRehearsal = `SELECT id,title,source_format,source_sha256,credential_replacements,simulator_profile,simulator_source_sha256,state,trace_json,fault_summary,repair_summary,handoff_markdown,review_decision,review_note,created_at,reviewed_at FROM rehearsals`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Rehearsal, error) {
	var item Rehearsal
	var trace, created, reviewed string
	var state, decision string
	err := row.Scan(&item.ID, &item.Title, &item.SourceFormat, &item.SourceSHA256, &item.CredentialReplacements, &item.SimulatorProfile, &item.SimulatorSourceSHA256, &state, &trace, &item.FaultSummary, &item.RepairSummary, &item.HandoffMarkdown, &decision, &item.ReviewNote, &created, &reviewed)
	if err != nil {
		return Rehearsal{}, err
	}
	item.State = State(state)
	item.ReviewDecision = Decision(decision)
	if err = json.Unmarshal([]byte(trace), &item.Trace); err != nil {
		return Rehearsal{}, fmt.Errorf("decode rehearsal trace: %w", err)
	}
	item.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Rehearsal{}, fmt.Errorf("parse created time: %w", err)
	}
	if reviewed != "" {
		value, e := time.Parse(time.RFC3339Nano, reviewed)
		if e != nil {
			return Rehearsal{}, fmt.Errorf("parse reviewed time: %w", e)
		}
		item.ReviewedAt = &value
	}
	return item, nil
}
