package database

import (
	"agent-manager/internal/domain"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"time"
)

type authorizationRepository struct{ db *DB }

func NewAuthorizationRepository(db *DB) domain.AuthorizationRepository {
	return &authorizationRepository{db}
}

// Private persistence carries frozen claims for deterministic reissuance using
// the existing run signer. Public grant receipts deliberately omit those fields.
type authorizationRecord struct {
	Grant                   *domain.AuthorizationGrant                `json:"grant"`
	Claims                  string                                    `json:"claims"`
	Hash                    string                                    `json:"hash"`
	NotificationSnapshot    *domain.AuthorizationNotificationSnapshot `json:"notification_snapshot,omitempty"`
	NotificationPublishedAt *time.Time                                `json:"notification_published_at,omitempty"`
	NotificationUnreached   string                                    `json:"notification_unreached,omitempty"`
	RequesterContext        *domain.AuthorizationRequesterContext     `json:"requester_context,omitempty"`
}

func encodeAuthorization(g *domain.AuthorizationGrant) ([]byte, error) {
	return json.Marshal(authorizationRecord{
		Grant:                   g,
		Claims:                  g.ClaimsJSON,
		Hash:                    g.TokenHash,
		NotificationSnapshot:    g.NotificationSnapshot,
		NotificationPublishedAt: g.NotificationPublishedAt,
		NotificationUnreached:   g.NotificationUnreached,
		RequesterContext:        g.RequesterContext,
	})
}
func decodeAuthorization(body string) (*domain.AuthorizationGrant, error) {
	var record authorizationRecord
	if err := json.Unmarshal([]byte(body), &record); err != nil {
		return nil, err
	}
	if record.Grant == nil {
		return nil, errors.New("invalid authorization record")
	}
	record.Grant.ClaimsJSON, record.Grant.TokenHash = record.Claims, record.Hash
	record.Grant.NotificationSnapshot = record.NotificationSnapshot
	record.Grant.NotificationPublishedAt = record.NotificationPublishedAt
	record.Grant.NotificationUnreached = record.NotificationUnreached
	record.Grant.RequesterContext = record.RequesterContext
	return record.Grant, nil
}
func (r *authorizationRepository) Create(ctx context.Context, g *domain.AuthorizationGrant) error {
	body, err := encodeAuthorization(g)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, "INSERT INTO authorization_grants (id, root_pid, state, body, created_at) VALUES (?, ?, ?, ?, ?)", g.ID.String(), g.Binding.PID, g.State, string(body), g.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z"))
	return err
}
func (r *authorizationRepository) Get(ctx context.Context, id uuid.UUID) (*domain.AuthorizationGrant, error) {
	var body string
	err := r.db.GetContext(ctx, &body, "SELECT body FROM authorization_grants WHERE id = ?", id.String())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeAuthorization(body)
}
func (r *authorizationRepository) List(ctx context.Context, pid int) ([]*domain.AuthorizationGrant, error) {
	var bodies []string
	if err := r.db.SelectContext(ctx, &bodies, "SELECT body FROM authorization_grants WHERE root_pid = ? ORDER BY created_at DESC LIMIT 64", pid); err != nil {
		return nil, err
	}
	grants := make([]*domain.AuthorizationGrant, 0, len(bodies))
	for _, body := range bodies {
		grant, err := decodeAuthorization(body)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

// ListNotificationRetryScanPage reads a bounded keyset page from the existing
// grant table. Eligibility is filtered by the caller after this bounded scan,
// so published or terminal rows cannot make the database walk an unbounded
// number of records to fill a page. UUID ordering gives the reconciler a stable
// cursor without materializing the whole authorization table.
func (r *authorizationRepository) ListNotificationRetryScanPage(ctx context.Context, afterID string, limit int) ([]*domain.AuthorizationGrant, error) {
	if limit <= 0 {
		return nil, nil
	}
	var bodies []string
	err := r.db.SelectContext(ctx, &bodies, `SELECT body FROM authorization_grants
WHERE id > ? ORDER BY id LIMIT ?`, afterID, limit)
	if err != nil {
		return nil, err
	}
	grants := make([]*domain.AuthorizationGrant, 0, len(bodies))
	for _, body := range bodies {
		grant, err := decodeAuthorization(body)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

func (r *authorizationRepository) Replace(ctx context.Context, g *domain.AuthorizationGrant, state string) (bool, error) {
	// A notification receipt may be stored while an approval/supersession is
	// being prepared from a previously-read grant. Retry the state transition
	// against the latest body and carry its private delivery metadata forward;
	// only a real state change makes the transition fail.
	for attempt := 0; attempt < 8; attempt++ {
		var currentBody string
		if err := r.db.GetContext(ctx, &currentBody, "SELECT body FROM authorization_grants WHERE id = ?", g.ID.String()); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		current, err := decodeAuthorization(currentBody)
		if err != nil {
			return false, err
		}
		if current.State != state {
			return false, nil
		}
		g.NotificationSnapshot = current.NotificationSnapshot
		g.NotificationPublishedAt = current.NotificationPublishedAt
		g.NotificationUnreached = current.NotificationUnreached
		g.RequesterContext = current.RequesterContext
		body, err := encodeAuthorization(g)
		if err != nil {
			return false, err
		}
		result, err := r.db.ExecContext(ctx, `UPDATE authorization_grants SET state = ?, body = ?
			WHERE id = ? AND state = ? AND body = ?`, g.State, string(body), g.ID.String(), state, currentBody)
		if err != nil {
			return false, err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 1 {
			return n == 1, err
		}
	}
	return false, nil
}

// UpdateNotification stores or acknowledges delivery metadata only while the
// same pending grant body is current. Comparing the full encoded wrapper makes
// this a generation-safe CAS without allowing a receipt to overwrite approval,
// revocation, or supersession.
func sameRequesterContext(a, b *domain.AuthorizationRequesterContext) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.BigPicture == b.BigPicture && a.TechnicalDetails == b.TechnicalDetails
}

func (r *authorizationRepository) UpdateNotification(ctx context.Context, expected, next *domain.AuthorizationGrant) (bool, error) {
	if expected == nil || next == nil || expected.ID == uuid.Nil || next.ID != expected.ID || expected.State != domain.AuthorizationPending || next.State != domain.AuthorizationPending {
		return false, errors.New("notification update requires one pending authorization generation")
	}
	expectedPublic, err := json.Marshal(expected)
	if err != nil {
		return false, err
	}
	nextPublic, err := json.Marshal(next)
	if err != nil {
		return false, err
	}
	if string(expectedPublic) != string(nextPublic) || expected.ClaimsJSON != next.ClaimsJSON || expected.TokenHash != next.TokenHash || !sameRequesterContext(expected.RequesterContext, next.RequesterContext) {
		return false, errors.New("notification update cannot change authorization terms or authority")
	}
	expectedBody, err := encodeAuthorization(expected)
	if err != nil {
		return false, err
	}
	nextBody, err := encodeAuthorization(next)
	if err != nil {
		return false, err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE authorization_grants SET body = ?
		WHERE id = ? AND state = ? AND body = ?`, string(nextBody), expected.ID.String(), domain.AuthorizationPending, string(expectedBody))
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
