package mentions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git-control-tower/internal/dbschema"
)

type Store struct{ db dbschema.DB }

type ReplyState string

const (
	ReplyPending    ReplyState = "pending"
	ReplyInFlight   ReplyState = "in_flight"
	ReplyDelivered  ReplyState = "delivered"
	ReplyUnverified ReplyState = "unverified"
)

func NewStore(db dbschema.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("mention store requires database")
	}
	s := &Store{db: db}
	_, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS gct_advisory_mentions (
        event_key TEXT PRIMARY KEY, provider TEXT NOT NULL, actor_id TEXT NOT NULL,
        repository_id TEXT NOT NULL, head_revision TEXT NOT NULL, command TEXT NOT NULL,
        state TEXT NOT NULL, reply_state TEXT NOT NULL DEFAULT 'pending',
        reply_receipt TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	if err != nil {
		return nil, fmt.Errorf("create mention schema: %w", err)
	}
	for _, statement := range []string{
		`ALTER TABLE gct_advisory_mentions ADD COLUMN reply_state TEXT NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE gct_advisory_mentions ADD COLUMN reply_receipt TEXT NOT NULL DEFAULT ''`,
	} {
		if _, alterErr := db.ExecContext(context.Background(), statement); alterErr != nil && !strings.Contains(strings.ToLower(alterErr.Error()), "duplicate column") {
			return nil, fmt.Errorf("migrate mention schema: %w", alterErr)
		}
	}
	return s, nil
}

// ClaimReply gives the external reply owner one idempotent outbox claim. GCT
// never sends the reply itself.
func (s *Store) ClaimReply(ctx context.Context, key string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE gct_advisory_mentions SET reply_state = ? WHERE event_key = ? AND reply_state = ?`, ReplyInFlight, key, ReplyPending)
	if err != nil {
		return false, fmt.Errorf("claim mention reply: %w", err)
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// ClaimReplyForRevision refuses to enqueue a public reply when the reviewed
// head is no longer current. The external reply owner can use this stronger
// claim after re-reading the host revision; the original request remains
// durable for audit and can be re-reviewed under a new event version.
func (s *Store) ClaimReplyForRevision(ctx context.Context, key, currentRevision string) (bool, error) {
	if strings.TrimSpace(currentRevision) == "" {
		return false, fmt.Errorf("current head revision is required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE gct_advisory_mentions SET reply_state = ? WHERE event_key = ? AND head_revision = ? AND reply_state = ?`, ReplyInFlight, key, currentRevision, ReplyPending)
	if err != nil {
		return false, fmt.Errorf("claim mention reply for revision: %w", err)
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) MarkReplyDelivered(ctx context.Context, key, receipt string) error {
	if receipt == "" {
		return fmt.Errorf("reply receipt is required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE gct_advisory_mentions SET reply_state = ?, reply_receipt = ? WHERE event_key = ? AND reply_state = ?`, ReplyDelivered, receipt, key, ReplyInFlight)
	if err != nil {
		return fmt.Errorf("mark mention reply: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return fmt.Errorf("mention reply is not in-flight")
	}
	return nil
}

func (s *Store) Record(ctx context.Context, request Request) (created bool, err error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("mention store is unavailable")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO gct_advisory_mentions (event_key,provider,actor_id,repository_id,head_revision,command,state,created_at) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(event_key) DO NOTHING`, request.Key, request.Delivery.Provider, request.Delivery.ActorID, request.Delivery.RepositoryID, request.Delivery.HeadRevision, request.Command, "queued", time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, fmt.Errorf("record mention: %w", err)
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
