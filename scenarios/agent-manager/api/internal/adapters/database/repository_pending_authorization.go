package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"agent-manager/internal/domain"
)

type pendingAuthorizationRepository struct{ db *DB }

// NewPendingAuthorizationRepository returns the durable store behind the
// "ask, don't block" authorization path.
func NewPendingAuthorizationRepository(db *DB) domain.PendingAuthorizationRepository {
	return &pendingAuthorizationRepository{db}
}

func (r *pendingAuthorizationRepository) Get(ctx context.Context, id string) (*domain.PendingAuthorizationRequest, error) {
	if id == "" {
		return nil, nil
	}
	var body string
	err := r.db.GetContext(ctx, &body, "SELECT body FROM pending_authorization_requests WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var req domain.PendingAuthorizationRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *pendingAuthorizationRepository) Put(ctx context.Context, req *domain.PendingAuthorizationRequest) error {
	if req == nil || req.ID == "" || req.Subject == "" {
		return errors.New("pending authorization request requires id and subject")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO pending_authorization_requests (id, subject, state, body, raised_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET subject=excluded.subject, state=excluded.state, body=excluded.body, raised_at=excluded.raised_at, expires_at=excluded.expires_at`,
		req.ID, req.Subject, req.State, string(body),
		req.RaisedAt.UTC().Format(standingGrantTimeLayout), req.ExpiresAt.UTC().Format(standingGrantTimeLayout))
	return err
}

func (r *pendingAuthorizationRepository) CountOutstandingForSubject(ctx context.Context, subject string, now time.Time) (int, error) {
	if subject == "" {
		return 0, nil
	}
	var n int
	// Outstanding = pending state AND not yet past expiry. Expired or decided
	// asks age out of the anti-flood count so a later legitimate raise is allowed.
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM pending_authorization_requests
		WHERE subject = ? AND state = ? AND expires_at > ?`,
		subject, domain.PendingAuthorizationPending, now.UTC().Format(standingGrantTimeLayout))
	if err != nil {
		return 0, err
	}
	return n, nil
}
