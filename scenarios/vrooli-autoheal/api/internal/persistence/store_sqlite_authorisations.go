package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

func (s *Store) recordRemediationAuthorisationSQLite(ctx context.Context, askID, incidentID, fingerprint, remediationID string, generation int, eventID, digest, candidateSnapshot, artifactSnapshot, approvedBy string, approvedAt time.Time) error {
	query := `INSERT INTO remediation_authorisations (ask_id, incident_id, incident_fingerprint, remediation_id, approved_by, approved_at)
		SELECT ?, ?, ?, ?, ?, ? WHERE EXISTS (SELECT 1 FROM incidents WHERE ` + currentRemediationDecisionPredicate + `)
		ON CONFLICT(ask_id) DO NOTHING`
	result, err := s.db.ExecContext(ctx, query, askID, incidentID, fingerprint, remediationID, approvedBy, approvedAt.UTC().Format(time.RFC3339Nano), incidentID, fingerprint, generation, generation, eventID, remediationID, digest, candidateSnapshot, artifactSnapshot, approvedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}

func (s *Store) claimRemediationAuthorisationSQLite(ctx context.Context, askID, incidentID, fingerprint, remediationID string, generation int, eventID, digest, candidateSnapshot, artifactSnapshot string, consumedAt time.Time) (bool, error) {
	query := `UPDATE remediation_authorisations SET consumed_at=?
		WHERE ask_id=? AND incident_id=? AND incident_fingerprint=? AND remediation_id=? AND consumed_at IS NULL
		AND EXISTS (SELECT 1 FROM incidents WHERE ` + currentRemediationDecisionPredicate + `)`
	result, err := s.db.ExecContext(ctx, query, consumedAt.UTC().Format(time.RFC3339Nano), askID, incidentID, fingerprint, remediationID, incidentID, fingerprint, generation, generation, eventID, remediationID, digest, candidateSnapshot, artifactSnapshot, consumedAt.UTC().Format(time.RFC3339Nano))
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

func (s *Store) recordRemediationDeclineIfCurrentSQLite(ctx context.Context, incidentID, fingerprint, candidateID, eventID, digest string, generation int, candidateSnapshot, artifactSnapshot string, outcome incidents.Outcome) (bool, error) {
	if outcome.ReportedAt.IsZero() {
		outcome.ReportedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(outcome)
	if err != nil {
		return false, fmt.Errorf("marshal declined remediation outcome: %w", err)
	}
	query := `UPDATE incidents SET outcome_json=?, updated_at=? WHERE ` + currentRemediationDecisionPredicate
	result, err := s.db.ExecContext(ctx, query, string(payload), outcome.ReportedAt.UTC().Format(time.RFC3339Nano), incidentID, fingerprint, generation, generation, eventID, candidateID, digest, candidateSnapshot, artifactSnapshot, outcome.ReportedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	return changed == 1, err
}
