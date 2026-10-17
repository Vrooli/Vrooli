package database

import (
	"agent-manager/internal/domain"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
)

type authorizationRepository struct{ db *DB }

func NewAuthorizationRepository(db *DB) domain.AuthorizationRepository {
	return &authorizationRepository{db}
}

// Private persistence carries frozen claims for deterministic reissuance using
// the existing run signer. Public grant receipts deliberately omit those fields.
type authorizationRecord struct {
	Grant  *domain.AuthorizationGrant `json:"grant"`
	Claims string                     `json:"claims"`
	Hash   string                     `json:"hash"`
}

func encodeAuthorization(g *domain.AuthorizationGrant) ([]byte, error) {
	return json.Marshal(authorizationRecord{g, g.ClaimsJSON, g.TokenHash})
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
func (r *authorizationRepository) Replace(ctx context.Context, g *domain.AuthorizationGrant, state string) (bool, error) {
	body, err := encodeAuthorization(g)
	if err != nil {
		return false, err
	}
	result, err := r.db.ExecContext(ctx, "UPDATE authorization_grants SET state = ?, body = ? WHERE id = ? AND state = ?", g.State, string(body), g.ID.String(), state)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
