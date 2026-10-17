package database

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"agent-manager/internal/domain"
)

type standingGrantRepository struct{ db *DB }

// NewStandingGrantRepository returns the durable, subject-keyed store behind the
// standing-grant authority model.
func NewStandingGrantRepository(db *DB) domain.StandingGrantRepository {
	return &standingGrantRepository{db}
}

const standingGrantTimeLayout = "2006-01-02T15:04:05.000000000Z"

func (r *standingGrantRepository) Put(ctx context.Context, g *domain.StandingGrant) error {
	if g == nil || g.ID == "" || g.Subject == "" || g.Grant == nil {
		return errors.New("standing grant requires id, subject and grant")
	}
	body, err := json.Marshal(g)
	if err != nil {
		return err
	}
	var revokedAt any
	if g.RevokedAt != nil {
		revokedAt = g.RevokedAt.UTC().Format(standingGrantTimeLayout)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO standing_grants (id, subject, session_id, state, body, approved_by, approved_at, expires_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET subject=excluded.subject, session_id=excluded.session_id, state=excluded.state, body=excluded.body, approved_by=excluded.approved_by, approved_at=excluded.approved_at, expires_at=excluded.expires_at, revoked_at=excluded.revoked_at`,
		g.ID, g.Subject, g.SessionID, g.State, string(body), g.ApprovedBy,
		g.ApprovedAt.UTC().Format(standingGrantTimeLayout), g.ExpiresAt.UTC().Format(standingGrantTimeLayout), revokedAt)
	return err
}

func (r *standingGrantRepository) ActiveForSubject(ctx context.Context, subject string, now time.Time) ([]*domain.StandingGrant, error) {
	if subject == "" {
		return nil, nil
	}
	var bodies []string
	// The store filters state and expiry in SQL so a caller never re-applies the
	// active predicate; Active() is still re-checked below as a defensive cross
	// of the decoded record against the same clock.
	if err := r.db.SelectContext(ctx, &bodies, `SELECT body FROM standing_grants
		WHERE subject = ? AND state = ? AND revoked_at IS NULL AND expires_at > ?
		ORDER BY approved_at DESC LIMIT 64`,
		subject, domain.StandingGrantApproved, now.UTC().Format(standingGrantTimeLayout)); err != nil {
		return nil, err
	}
	grants := make([]*domain.StandingGrant, 0, len(bodies))
	for _, body := range bodies {
		var g domain.StandingGrant
		if err := json.Unmarshal([]byte(body), &g); err != nil {
			return nil, err
		}
		if !g.Active(now) {
			continue
		}
		grants = append(grants, &g)
	}
	return grants, nil
}

func (r *standingGrantRepository) List(ctx context.Context, limit int) ([]*domain.StandingGrant, error) {
	if limit <= 0 || limit > 256 {
		limit = 64
	}
	var bodies []string
	// The admin surface shows the full history, including revoked and expired
	// records, so the operator can audit what was granted and when; each record
	// carries its own state and expiry.
	if err := r.db.SelectContext(ctx, &bodies, `SELECT body FROM standing_grants ORDER BY approved_at DESC LIMIT ?`, limit); err != nil {
		return nil, err
	}
	grants := make([]*domain.StandingGrant, 0, len(bodies))
	for _, body := range bodies {
		var g domain.StandingGrant
		if err := json.Unmarshal([]byte(body), &g); err != nil {
			return nil, err
		}
		grants = append(grants, &g)
	}
	return grants, nil
}

func (r *standingGrantRepository) Revoke(ctx context.Context, id string, now time.Time) (bool, error) {
	if id == "" {
		return false, nil
	}
	revokedAt := now.UTC().Format(standingGrantTimeLayout)
	// Only an approved row transitions; re-revoking is a no-op. The body is kept
	// as the durable record of what was granted, with its state updated so a
	// decode still reads revoked.
	result, err := r.db.ExecContext(ctx, `UPDATE standing_grants
		SET state = ?, revoked_at = ?, body = json_set(body, '$.state', ?, '$.revoked_at', ?)
		WHERE id = ? AND state = ? AND revoked_at IS NULL`,
		domain.StandingGrantRevoked, revokedAt, domain.StandingGrantRevoked, revokedAt, id, domain.StandingGrantApproved)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
