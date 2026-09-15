package desktoplink

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SQLRepository is the production repository. SQL remains isolated here so a
// future SQLite/PostgreSQL adapter swap does not change the link protocol.
type SQLRepository struct {
	db interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
}

func NewSQLRepository(db interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) CreateAuthorization(ctx context.Context, authorization Authorization) error {
	scopes, err := encodeScopes(authorization.Scopes)
	if err != nil {
		return fmt.Errorf("encode desktop link scopes: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO desktop_link_authorizations
		(id, code_hash, lpbs_user_id, business_account_id, installation_id, resource, audience, scopes_json, code_challenge, redirect_uri, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		authorization.ID, authorization.CodeHash, authorization.LPBSUserID, authorization.BusinessAccountID,
		authorization.InstallationID, authorization.Resource, authorization.Audience, scopes, authorization.CodeChallenge,
		authorization.RedirectURI, authorization.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store desktop authorization: %w", err)
	}
	return nil
}

func (r *SQLRepository) RedeemAuthorization(ctx context.Context, codeHash, verifier, localPrincipal, installationID, resource string) (Link, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Link{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var authorization Authorization
	var scopesJSON string
	err = tx.QueryRowContext(ctx, `
		SELECT id, lpbs_user_id, business_account_id, installation_id, resource, audience, scopes_json, code_challenge, redirect_uri, expires_at
		FROM desktop_link_authorizations
		WHERE code_hash = $1 AND used_at IS NULL
		FOR UPDATE`, codeHash).Scan(&authorization.ID, &authorization.LPBSUserID, &authorization.BusinessAccountID, &authorization.InstallationID, &authorization.Resource, &authorization.Audience, &scopesJSON, &authorization.CodeChallenge, &authorization.RedirectURI, &authorization.ExpiresAt)
	if err != nil || !time.Now().Before(authorization.ExpiresAt) || authorization.InstallationID != installationID || authorization.Resource != resource || !pkceMatches(verifier, authorization.CodeChallenge) {
		return Link{}, ErrCodeRejected
	}
	if err := json.Unmarshal([]byte(scopesJSON), &authorization.Scopes); err != nil || len(authorization.Scopes) == 0 {
		return Link{}, ErrCodeRejected
	}
	linkID, err := randomSecret()
	if err != nil {
		return Link{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE desktop_link_authorizations SET used_at = CURRENT_TIMESTAMP WHERE id = $1 AND used_at IS NULL`, authorization.ID); err != nil {
		return Link{}, err
	}
	scopes, _ := json.Marshal(authorization.Scopes)
	if _, err := tx.ExecContext(ctx, `INSERT INTO desktop_account_links (id, lpbs_user_id, business_account_id, local_provider, local_principal, installation_id, resource, audience, scopes_json) VALUES ($1,$2,$3,'scenario-authenticator',$4,$5,$6,$7,$8)`, linkID, authorization.LPBSUserID, authorization.BusinessAccountID, localPrincipal, installationID, resource, authorization.Audience, string(scopes)); err != nil {
		return Link{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO desktop_link_audit (id, link_id, authorization_id, event, actor_type, actor_id, installation_id, resource) VALUES ($1,$2,$3,'linked','local_principal',$4,$5,$6)`, randomID(), linkID, authorization.ID, localPrincipal, installationID, resource); err != nil {
		return Link{}, err
	}
	if err := tx.Commit(); err != nil {
		return Link{}, err
	}
	return Link{ID: linkID, LPBSUserID: authorization.LPBSUserID, BusinessAccountID: authorization.BusinessAccountID, LocalProvider: "scenario-authenticator", LocalPrincipal: localPrincipal, InstallationID: installationID, Resource: resource, Audience: authorization.Audience, Scopes: authorization.Scopes, CreatedAt: time.Now().UTC()}, nil
}

func (r *SQLRepository) StatusByLocal(ctx context.Context, principal, installationID, resource string) (Link, error) {
	var link Link
	var scopesJSON string
	var revokedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT id, lpbs_user_id, business_account_id, local_provider, local_principal, installation_id, resource, audience, scopes_json, created_at, revoked_at, COALESCE(revoked_by, '') FROM desktop_account_links WHERE local_principal = $1 AND installation_id = $2 AND resource = $3 ORDER BY created_at DESC LIMIT 1`, principal, installationID, resource).Scan(&link.ID, &link.LPBSUserID, &link.BusinessAccountID, &link.LocalProvider, &link.LocalPrincipal, &link.InstallationID, &link.Resource, &link.Audience, &scopesJSON, &link.CreatedAt, &revokedAt, &link.RevokedBy)
	if err != nil {
		return Link{}, ErrCodeRejected
	}
	if err := json.Unmarshal([]byte(scopesJSON), &link.Scopes); err != nil {
		return Link{}, ErrCodeRejected
	}
	if revokedAt.Valid {
		link.RevokedAt = &revokedAt.Time
	}
	return link, nil
}

func (r *SQLRepository) RevokeByLPBS(ctx context.Context, userID, installationID, resource, actor string) (int64, error) {
	return r.revoke(ctx, `lpbs_user_id = $1`, userID, installationID, resource, actor, "lpbs_user")
}

func (r *SQLRepository) RevokeByLocal(ctx context.Context, principal, installationID, resource, actor string) (int64, error) {
	return r.revoke(ctx, `local_principal = $1`, principal, installationID, resource, actor, "local_principal")
}

func (r *SQLRepository) revoke(ctx context.Context, predicate, identity, installationID, resource, actor, actorType string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT id FROM desktop_account_links WHERE `+predicate+` AND installation_id = $2 AND resource = $3 AND revoked_at IS NULL FOR UPDATE`, identity, installationID, resource)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE desktop_account_links SET revoked_at = CURRENT_TIMESTAMP, revoked_by = $2 WHERE id = $1 AND revoked_at IS NULL`, id, actor); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO desktop_link_audit (id, link_id, event, actor_type, actor_id, installation_id, resource) VALUES ($1,$2,'revoked',$3,$4,$5,$6)`, randomID(), id, actorType, actor, installationID, resource); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int64(len(ids)), nil
}

func randomID() string {
	value, err := randomSecret()
	if err != nil {
		return strings.Repeat("0", 43)
	}
	return value
}

var _ Repository = (*SQLRepository)(nil)
