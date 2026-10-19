package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

const currentRemediationDecisionPredicate = `id=? AND fingerprint=? AND status IN ('open','acknowledged')
	AND event_count=? AND remediation_decision_generation=? AND remediation_decision_event_id=?
	AND remediation_decision_candidate_id=? AND remediation_candidate_digest=? AND CAST(remediation_candidates_json AS TEXT)=? AND CAST(remediation_artifacts_json AS TEXT)=? AND remediation_decision_expires_at>?`

// EnsureRemediationDecisionPublication atomically binds the latest eligible
// candidate snapshot and persists its immutable request before any publication.
func (s *Store) EnsureRemediationDecisionPublication(ctx context.Context, incidentID string, observedAt time.Time) (*incidents.RemediationDecisionPublication, error) {
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	observedAt = observedAt.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin remediation decision snapshot: %w", err)
	}
	defer tx.Rollback()
	incident, err := getIncidentSQLiteTx(ctx, tx, incidentID)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, sql.ErrNoRows
	}

	generation := incident.EventCount
	if generation < 1 {
		return nil, nil
	}
	if incident.RemediationDecisionGeneration != generation {
		incident.RemediationDecisionGeneration = generation
		incident.RemediationDecisionRevision = 0
		incident.RemediationDecisionEventID = ""
		incident.RemediationDecisionCandidateID = ""
		incident.RemediationCandidateDigest = ""
		incident.RemediationDecisionExpiresAt = nil
	}
	candidate, eligible := firstApplicableRemediationCandidate(*incident)
	if !eligible || incident.Severity != incidents.SeverityCritical || (incident.Status != incidents.StatusOpen && incident.Status != incidents.StatusAcknowledged) {
		if _, err := tx.ExecContext(ctx, `UPDATE incident_remediation_publications SET superseded_at = ? WHERE incident_id = ? AND generation <= ? AND superseded_at IS NULL`, observedAt.Format(time.RFC3339Nano), incidentID, generation); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE incidents SET remediation_decision_generation=?, remediation_decision_revision=?, remediation_decision_event_id='', remediation_decision_candidate_id='', remediation_candidate_digest='', remediation_decision_expires_at=NULL WHERE id=? AND event_count=?`, generation, incident.RemediationDecisionRevision, incidentID, generation); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	digest, err := incidents.RemediationDecisionSnapshotDigest(*incident, candidate)
	if err != nil {
		return nil, err
	}
	if digest == incident.RemediationCandidateDigest && incident.RemediationDecisionEventID != "" && incident.RemediationDecisionExpiresAt != nil && observedAt.Before(*incident.RemediationDecisionExpiresAt) {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	revision := 1
	if incident.RemediationDecisionGeneration == generation {
		revision = incident.RemediationDecisionRevision + 1
	}
	if revision < 1 {
		revision = 1
	}
	deadline := observedAt.Add(operatordecision.DefaultWindow)
	incident.RemediationDecisionGeneration = generation
	event := incidents.BuildRemediationDecisionEvent(*incident, candidate, revision, digest, observedAt, deadline)
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal remediation decision request: %w", err)
	}
	eventID := operatordecision.EventID(event)
	publication := &incidents.RemediationDecisionPublication{
		EventID: eventID, IncidentID: incidentID, Generation: generation, Revision: revision,
		CandidateDigest: digest, EventJSON: eventJSON,
	}
	if _, err := tx.ExecContext(ctx, `UPDATE incident_remediation_publications SET superseded_at=? WHERE incident_id=? AND generation<=? AND superseded_at IS NULL`, observedAt.Format(time.RFC3339Nano), incidentID, generation); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO incident_remediation_publications (event_id, incident_id, generation, revision, candidate_digest, event_json, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(event_id) DO NOTHING`, eventID, incidentID, generation, revision, digest, string(eventJSON), observedAt.Format(time.RFC3339Nano), deadline.Format(time.RFC3339Nano)); err != nil {
		return nil, fmt.Errorf("persist immutable remediation request: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE incidents SET remediation_decision_generation=?, remediation_decision_revision=?, remediation_decision_event_id=?, remediation_decision_candidate_id=?, remediation_candidate_digest=?, remediation_decision_expires_at=? WHERE id=? AND event_count=? AND status IN ('open','acknowledged')`, generation, revision, eventID, candidate.ID, digest, deadline.Format(time.RFC3339Nano), incidentID, generation); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return publication, nil
}

func firstApplicableRemediationCandidate(incident incidents.Incident) (incidents.RemediationCandidate, bool) {
	for _, candidate := range incident.RemediationCandidates {
		if strings.TrimSpace(candidate.ID) != "" && candidate.Applicability == "applicable" {
			return candidate, true
		}
	}
	return incidents.RemediationCandidate{}, false
}

func getIncidentSQLiteTx(ctx context.Context, tx *sql.Tx, id string) (*incidents.Incident, error) {
	row := tx.QueryRowContext(ctx, `SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
		resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
		source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
		diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
		rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
		event_count, observation_count, operator_notes, remediation_decision_generation, remediation_decision_revision,
		remediation_decision_event_id, remediation_decision_candidate_id, remediation_candidate_digest, remediation_decision_expires_at FROM incidents WHERE id=?`, id)
	return scanIncident(row)
}

func (s *Store) ListPendingRemediationDecisionPublications(ctx context.Context, limit int) ([]incidents.RemediationDecisionPublication, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.event_id, p.incident_id, p.generation, p.revision, p.candidate_digest, p.event_json, p.published_at, p.superseded_at
		FROM incident_remediation_publications p
		JOIN incidents i ON i.id=p.incident_id AND i.remediation_decision_event_id=p.event_id AND i.event_count=p.generation
		WHERE p.published_at IS NULL AND p.superseded_at IS NULL AND i.status IN ('open','acknowledged')
		ORDER BY p.created_at, p.event_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]incidents.RemediationDecisionPublication, 0)
	for rows.Next() {
		var item incidents.RemediationDecisionPublication
		var publishedRaw, supersededRaw any
		if err := rows.Scan(&item.EventID, &item.IncidentID, &item.Generation, &item.Revision, &item.CandidateDigest, &item.EventJSON, &publishedRaw, &supersededRaw); err != nil {
			return nil, err
		}
		item.PublishedAt = parseOptionalTimePtr(publishedRaw)
		item.SupersededAt = parseOptionalTimePtr(supersededRaw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) AcknowledgeRemediationDecisionPublication(ctx context.Context, eventID string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE incident_remediation_publications SET published_at=? WHERE event_id=? AND published_at IS NULL`, time.Now().UTC().Format(time.RFC3339Nano), eventID)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err != nil {
		return err
	} else if changed != 1 {
		return fmt.Errorf("decision publication %q is no longer pending", eventID)
	}
	return nil
}
