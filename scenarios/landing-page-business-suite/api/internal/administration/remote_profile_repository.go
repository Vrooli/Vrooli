package administration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *RemoteProfileService) GetByID(ctx context.Context, id int64) (*RemoteProfile, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	profile := rec.toProfile(s.nowTime())
	return &profile, nil
}

func (s *RemoteProfileService) getRecordByID(ctx context.Context, id int64) (*remoteProfileRecord, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, tag, label, api_base, connector_id, remote_session_id, status, encrypted_session,
		       session_expires_at, remote_session_last_synced_at, last_login_at, last_used_at,
		       created_by, created_at, updated_at
		FROM remote_profiles
		WHERE id = $1
	`, id)
	rec, err := scanRemoteProfileRow(row)
	if err != nil {
		return nil, err
	}
	connectorID, err := s.ensureConnectorID(ctx, id, remoteProfileConnectorID(rec))
	if err != nil {
		return nil, err
	}
	rec.ConnectorID = sql.NullString{String: connectorID, Valid: true}
	return rec, nil
}

func (s *RemoteProfileService) tagExists(ctx context.Context, tag string, excludeID int64) (bool, error) {
	var count int
	if excludeID > 0 {
		if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM remote_profiles WHERE tag = $1 AND id <> $2`, tag, excludeID).Scan(&count); err != nil {
			return false, err
		}
	} else {
		if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM remote_profiles WHERE tag = $1`, tag).Scan(&count); err != nil {
			return false, err
		}
	}
	return count > 0, nil
}

func (s *RemoteProfileService) ensureConnectorID(ctx context.Context, id int64, current string) (string, error) {
	trimmed := strings.TrimSpace(current)
	if trimmed != "" {
		return trimmed, nil
	}

	connectorID, err := generateRemoteConnectorID()
	if err != nil {
		return "", err
	}

	if _, err := s.DB.ExecContext(ctx, `
		UPDATE remote_profiles
		SET connector_id = $1,
		    updated_at = NOW()
		WHERE id = $2 AND (connector_id IS NULL OR connector_id = '')
	`, connectorID, id); err != nil {
		return "", err
	}

	var resolved sql.NullString
	if err := s.DB.QueryRowContext(ctx, `SELECT connector_id FROM remote_profiles WHERE id = $1`, id).Scan(&resolved); err != nil {
		return "", err
	}
	if !resolved.Valid || strings.TrimSpace(resolved.String) == "" {
		return "", fmt.Errorf("connector id missing for remote profile %d", id)
	}
	return strings.TrimSpace(resolved.String), nil
}

func (s *RemoteProfileService) EnsureConnectorID(ctx context.Context, id int64, current string) (string, error) {
	return s.ensureConnectorID(ctx, id, current)
}

func (s *RemoteProfileService) lookupAdminID(ctx context.Context, email string) (*int64, error) {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return nil, nil
	}
	var id int64
	err := s.DB.QueryRowContext(ctx, `SELECT id FROM admin_users WHERE LOWER(email) = LOWER($1)`, trimmed).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *RemoteProfileService) setSession(ctx context.Context, id int64, sessionValue string, remoteSessionID string, expiresAt *time.Time) error {
	encrypted, err := s.encrypt(sessionValue)
	if err != nil {
		return err
	}
	normalizedRemoteSessionID := normalizeRemoteProfileSessionID(remoteSessionID)
	var remoteSessionIDPtr *string
	if normalizedRemoteSessionID != "" {
		remoteSessionIDPtr = &normalizedRemoteSessionID
	}
	_, err = s.DB.ExecContext(ctx, `
		UPDATE remote_profiles
		SET encrypted_session = $1,
		    encryption_state = $6,
		    remote_session_id = $2,
		    remote_session_last_synced_at = NOW(),
		    session_expires_at = $3,
		    status = $4,
		    last_login_at = NOW(),
		    last_used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $5
	`, encrypted, stringToNullString(remoteSessionIDPtr), timeToNullTime(expiresAt), remoteProfileStatusActive, id, encryptionState(s.EncryptionKey))
	return err
}

func (s *RemoteProfileService) SetSession(ctx context.Context, id int64, sessionValue string, remoteSessionID string, expiresAt *time.Time) error {
	return s.setSession(ctx, id, sessionValue, remoteSessionID, expiresAt)
}

func (s *RemoteProfileService) clearSession(ctx context.Context, id int64, status string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE remote_profiles
		SET encrypted_session = NULL,
		    encryption_state = 'unknown',
		    remote_session_id = NULL,
		    remote_session_last_synced_at = NOW(),
		    session_expires_at = NULL,
		    status = $1,
		    last_used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}

func encryptionState(key []byte) string {
	if key == nil {
		return "unsealed"
	}
	return "sealed"
}

func (s *RemoteProfileService) updateStatus(ctx context.Context, id int64, status string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE remote_profiles
		SET status = $1,
		    last_used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}
