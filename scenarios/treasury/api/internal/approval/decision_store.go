package approval

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/vrooli/api-core/operatordecision"
)

// DecisionAskStore is Treasury-owned durable correlation state for Notification
// Hub decisions. Hub state never replaces the local approval queue.
type DecisionAskStore interface {
	operatordecision.PendingStore
}

type SQLiteDecisionAskStore struct{ db DB }

func NewSQLiteDecisionAskStore(db DB) *SQLiteDecisionAskStore {
	return &SQLiteDecisionAskStore{db: db}
}

func (s *SQLiteDecisionAskStore) Get(ctx context.Context, id string) (*operatordecision.PendingAsk, error) {
	var value operatordecision.PendingAsk
	var raisedAt, expiresAt, resolvedAt, extra string
	var published int
	err := s.db.QueryRowContext(ctx, `SELECT id,subject,state,raised_at,expires_at,event_id,published,resolved_at,resolved_by,extra_json FROM approval_decision_asks WHERE id=?`, id).Scan(
		&value.ID, &value.Subject, &value.State, &raisedAt, &expiresAt, &value.EventID, &published, &resolvedAt, &value.ResolvedBy, &extra,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if value.RaisedAt, err = time.Parse(time.RFC3339Nano, raisedAt); err != nil {
		return nil, err
	}
	if value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt); err != nil {
		return nil, err
	}
	if resolvedAt != "" {
		resolved, parseErr := time.Parse(time.RFC3339Nano, resolvedAt)
		if parseErr != nil {
			return nil, parseErr
		}
		value.ResolvedAt = &resolved
	}
	value.Published = published != 0
	if err := json.Unmarshal([]byte(extra), &value.Extra); err != nil {
		return nil, err
	}
	return &value, nil
}

func (s *SQLiteDecisionAskStore) Put(ctx context.Context, value *operatordecision.PendingAsk) error {
	if value == nil || value.ID == "" || value.Subject == "" || value.RaisedAt.IsZero() || value.ExpiresAt.IsZero() {
		return errors.New("complete Treasury decision ask is required")
	}
	extra, err := json.Marshal(value.Extra)
	if err != nil {
		return err
	}
	resolvedAt := ""
	if value.ResolvedAt != nil {
		resolvedAt = value.ResolvedAt.UTC().Format(time.RFC3339Nano)
	}
	published := 0
	if value.Published {
		published = 1
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO approval_decision_asks(id,subject,state,raised_at,expires_at,event_id,published,resolved_at,resolved_by,extra_json) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET subject=excluded.subject,state=excluded.state,raised_at=excluded.raised_at,expires_at=excluded.expires_at,event_id=excluded.event_id,published=excluded.published,resolved_at=excluded.resolved_at,resolved_by=excluded.resolved_by,extra_json=excluded.extra_json`,
		value.ID, value.Subject, value.State, value.RaisedAt.UTC().Format(time.RFC3339Nano), value.ExpiresAt.UTC().Format(time.RFC3339Nano), value.EventID, published, resolvedAt, value.ResolvedBy, string(extra),
	)
	return err
}

func (s *SQLiteDecisionAskStore) CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM approval_decision_asks WHERE subject=? AND state=? AND expires_at>?`, subject, operatordecision.PendingStatePending, now.UTC().Format(time.RFC3339Nano)).Scan(&count)
	return count, err
}

var _ DecisionAskStore = (*SQLiteDecisionAskStore)(nil)
