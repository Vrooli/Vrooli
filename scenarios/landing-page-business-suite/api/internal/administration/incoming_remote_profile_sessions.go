package administration

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// IncomingRemoteProfileSessionStore is the persistence seam for sessions
// created by remote-profile connectors. It is deliberately domain-owned so
// HTTP handlers do not couple transport behavior to the admin_sessions schema.
type IncomingRemoteProfileSessionStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// IncomingRemoteProfileSessionRepository owns the session queries used by the
// administration transport.
type IncomingRemoteProfileSessionRepository struct {
	store IncomingRemoteProfileSessionStore
}

func NewIncomingRemoteProfileSessionRepository(store IncomingRemoteProfileSessionStore) *IncomingRemoteProfileSessionRepository {
	return &IncomingRemoteProfileSessionRepository{store: store}
}

func (r *IncomingRemoteProfileSessionRepository) List(ctx context.Context, connectorID string) ([]IncomingRemoteProfileSession, error) {
	rows, err := r.store.QueryContext(ctx, `SELECT id, admin_email, created_at, last_activity, expires_at, ip_address, user_agent FROM admin_sessions ORDER BY last_activity DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filter := strings.TrimSpace(connectorID)
	sessions := []IncomingRemoteProfileSession{}
	for rows.Next() {
		var id, email string
		var created, active, expiresAt time.Time
		var ip, agent sql.NullString
		if err := rows.Scan(&id, &email, &created, &active, &expiresAt, &ip, &agent); err != nil {
			return nil, err
		}
		meta, ok := ParseRemoteProfileSessionUserAgent(agent.String)
		if !ok || (filter != "" && filter != meta.ConnectorID) {
			continue
		}
		sessions = append(sessions, IncomingRemoteProfileSession{
			SessionID:    id,
			AdminEmail:   email,
			ConnectorID:  meta.ConnectorID,
			ProfileTag:   meta.ProfileTag,
			Origin:       meta.Origin,
			CreatedAt:    created,
			LastActivity: active,
			ExpiresAt:    expiresAt,
			IPAddress:    nullStringValue(ip),
			UserAgent:    nullStringValue(agent),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *IncomingRemoteProfileSessionRepository) Revoke(ctx context.Context, sessionID string) (bool, error) {
	result, err := r.store.ExecContext(ctx, `DELETE FROM admin_sessions WHERE id = $1 AND user_agent LIKE $2`, strings.TrimSpace(sessionID), RemoteProfileSessionAgentPrefix+"%")
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
