package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/sqlutil"
	"time"
)

// SelectedCanonical returns one exact canonical row with bounded allocation.
// It is a storage primitive, not authorization or a public HTTP route. Trusted
// finite readers must hold their authority boundary through validation/release.
// Metadata is deliberately excluded; identity must also match the envelope.
func (s *SQLiteStore) SelectedCanonical(ctx context.Context, eventID string) (*Event, error) {
	if s == nil || s.db == nil || ctx == nil || ctx.Err() != nil || eventID == "" || len(eventID) > 128 {
		return nil, errors.New("invalid selected event read")
	}
	var e Event
	var size int64
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT event_id,event_type,source_scenario,
 CASE WHEN length(payload) <= 1048576 THEN payload ELSE NULL END,
 COALESCE(length(payload),0),created_at FROM events WHERE event_id=?`, eventID).
		Scan(&e.EventID, &e.EventType, &e.SourceScenario, &e.Payload, &size, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if size <= 0 || size > 1<<20 || len(e.Payload) != int(size) {
		return nil, errors.New("invalid canonical event size")
	}
	e.CreatedAt, err = time.Parse(sqlutil.TimestampFormat, created)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
