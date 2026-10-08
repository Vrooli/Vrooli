package persistence

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) recordRemediationAuthorisationSQLite(ctx context.Context, askID, incidentID, fingerprint, remediationID, approvedBy string, approvedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO remediation_authorisations
			(ask_id, incident_id, incident_fingerprint, remediation_id, approved_by, approved_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, askID, incidentID, fingerprint, remediationID, approvedBy, approvedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) claimRemediationAuthorisationSQLite(ctx context.Context, askID, incidentID, fingerprint, remediationID string, consumedAt time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE remediation_authorisations
		SET consumed_at = ?
		WHERE ask_id = ? AND incident_id = ? AND incident_fingerprint = ?
		  AND remediation_id = ? AND consumed_at IS NULL
	`, consumedAt.UTC().Format(time.RFC3339Nano), askID, incidentID, fingerprint, remediationID)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if count == 0 {
		// Distinguish a missing table in an old test database from a normal
		// rejected replay while keeping the caller's error typed.
		var exists int
		if scanErr := s.db.QueryRowContext(ctx, `SELECT 1 FROM remediation_authorisations WHERE ask_id = ?`, askID).Scan(&exists); scanErr != nil && scanErr != sql.ErrNoRows {
			return false, scanErr
		}
	}
	return count == 1, nil
}
