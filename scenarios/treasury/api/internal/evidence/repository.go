package evidence

import (
	"context"
	"database/sql"
	"fmt"
)

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type SQLiteRecorder struct{ db DB }

func NewSQLiteRecorder(db DB) *SQLiteRecorder { return &SQLiteRecorder{db: db} }

func (r *SQLiteRecorder) Append(ctx context.Context, value Record) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO evidence_records(id, authorization_id, mandate_id, agent_subject, verdict, violated_constraint, detail, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`, value.ID, value.AuthorizationID, value.MandateID, value.AgentSubject, value.Verdict, value.ViolatedConstraint, value.Detail, value.CreatedAt)
	if err != nil {
		return fmt.Errorf("append evidence: %w", err)
	}
	return nil
}
