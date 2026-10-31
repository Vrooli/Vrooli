package releasedecision

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"deployment-manager/shared"

	"github.com/vrooli/api-core/operatordecision"
)

// SQLStore is the SQLite-backed operatordecision.PendingStore for release
// decision asks. It is keyed by correlation id (the ask id), which is what
// makes a repeated raise idempotent.
type SQLStore struct {
	conn shared.DBTX
}

// NewSQLStore builds a store over the routed database seam.
func NewSQLStore(db shared.RoutedDBTX) *SQLStore {
	return &SQLStore{conn: db}
}

// Get returns the ask for a correlation id, or nil when none exists.
func (s *SQLStore) Get(ctx context.Context, id string) (*operatordecision.PendingAsk, error) {
	row := s.conn.QueryRowContext(ctx,
		`SELECT id, subject, state, raised_at, expires_at, event_id, published, resolved_at, resolved_by, extra
		 FROM release_decision_asks WHERE id = $1`, id)
	return scanAsk(row)
}

// Put inserts or replaces an ask by id.
func (s *SQLStore) Put(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if ask == nil {
		return nil
	}
	extra := "{}"
	if len(ask.Extra) > 0 {
		encoded, err := json.Marshal(ask.Extra)
		if err != nil {
			return err
		}
		extra = string(encoded)
	}
	_, err := s.conn.ExecContext(ctx,
		`INSERT INTO release_decision_asks
			(id, subject, state, raised_at, expires_at, event_id, published, resolved_at, resolved_by, extra)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT(id) DO UPDATE SET
			subject = excluded.subject,
			state = excluded.state,
			raised_at = excluded.raised_at,
			expires_at = excluded.expires_at,
			event_id = excluded.event_id,
			published = excluded.published,
			resolved_at = excluded.resolved_at,
			resolved_by = excluded.resolved_by,
			extra = excluded.extra`,
		ask.ID, ask.Subject, ask.State, ask.RaisedAt.UTC(), ask.ExpiresAt.UTC(), ask.EventID, ask.Published,
		nullTime(ask.ResolvedAt), nullString(ask.ResolvedBy), extra)
	return err
}

// CountOutstandingForSubject returns how many un-decided, un-expired asks the
// subject currently has, for the anti-flood cap.
func (s *SQLStore) CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error) {
	row := s.conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM release_decision_asks
		 WHERE subject = $1 AND state = $2 AND expires_at > $3`,
		subject, operatordecision.PendingStatePending, now.UTC())
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func scanAsk(row *sql.Row) (*operatordecision.PendingAsk, error) {
	var (
		ask        operatordecision.PendingAsk
		published  bool
		resolvedAt sql.NullTime
		resolvedBy sql.NullString
		extra      sql.NullString
	)
	if err := row.Scan(&ask.ID, &ask.Subject, &ask.State, &ask.RaisedAt, &ask.ExpiresAt, &ask.EventID, &published, &resolvedAt, &resolvedBy, &extra); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ask.Published = published
	if resolvedAt.Valid {
		t := resolvedAt.Time.UTC()
		ask.ResolvedAt = &t
	}
	if resolvedBy.Valid {
		ask.ResolvedBy = resolvedBy.String
	}
	if extra.Valid && extra.String != "" {
		decoded := map[string]string{}
		if err := json.Unmarshal([]byte(extra.String), &decoded); err == nil {
			ask.Extra = decoded
		}
	}
	return &ask, nil
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
