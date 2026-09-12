package triage

import (
	"context"
	"database/sql"
	"time"
)

const timeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database capability this repository needs. Keeping
// it local prevents the triage domain from importing the signals domain merely
// to share a test seam.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepo struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepo{db} }
func (r *sqliteRepo) GetDisposition(c context.Context, id string) (Disposition, bool, error) {
	var d Disposition
	var revisit, updated string
	e := r.db.QueryRowContext(c, "SELECT signal_id,state,revisit_at,updated_at FROM disposition WHERE signal_id=?", id).Scan(&d.SignalID, &d.State, &revisit, &updated)
	if e == sql.ErrNoRows {
		return d, false, nil
	}
	if e != nil {
		return d, false, e
	}
	d.UpdatedAt, _ = time.Parse(timeFormat, updated)
	if revisit != "" {
		v, _ := time.Parse(timeFormat, revisit)
		d.RevisitAt = &v
	}
	return d, true, nil
}

func (r *sqliteRepo) UpsertDisposition(c context.Context, d Disposition) (Disposition, error) {
	revisit := ""
	if d.RevisitAt != nil {
		revisit = d.RevisitAt.UTC().Format(timeFormat)
	}
	_, e := r.db.ExecContext(c, "INSERT INTO disposition(signal_id,state,revisit_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(signal_id) DO UPDATE SET state=excluded.state,revisit_at=excluded.revisit_at,updated_at=excluded.updated_at", d.SignalID, d.State, revisit, d.UpdatedAt.UTC().Format(timeFormat))
	return d, e
}

func (r *sqliteRepo) AppendAnnotation(c context.Context, a Annotation) (Annotation, error) {
	kind, target := "", ""
	if a.Outcome != nil {
		kind, target = string(a.Outcome.Kind), a.Outcome.TargetID
	}
	_, e := r.db.ExecContext(c, "INSERT INTO annotation(id,signal_id,author,body,outcome_kind,outcome_target_id,created_at) VALUES(?,?,?,?,?,?,?)", a.ID, a.SignalID, a.Author, a.Body, kind, target, a.CreatedAt.UTC().Format(timeFormat))
	return a, e
}

func (r *sqliteRepo) ListAnnotations(c context.Context, id string) ([]Annotation, error) {
	// rowid is the stable tie-breaker when a deterministic clock gives two
	// append-only annotations the same timestamp. UUID order would rewrite the
	// visible append sequence at read time.
	rows, e := r.db.QueryContext(c, "SELECT id,signal_id,author,body,outcome_kind,outcome_target_id,created_at FROM annotation WHERE signal_id=? ORDER BY created_at,rowid", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Annotation{}
	for rows.Next() {
		var a Annotation
		var k, t, at string
		if e = rows.Scan(&a.ID, &a.SignalID, &a.Author, &a.Body, &k, &t, &at); e != nil {
			return nil, e
		}
		a.CreatedAt, _ = time.Parse(timeFormat, at)
		if k != "" {
			a.Outcome = &Outcome{Kind: OutcomeKind(k), TargetID: t}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
