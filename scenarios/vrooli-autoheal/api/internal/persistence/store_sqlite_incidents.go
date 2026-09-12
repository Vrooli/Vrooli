package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

func (s *Store) upsertIncidentSQLite(ctx context.Context, input incidents.UpsertInput) (*incidents.Incident, error) {
	if input.ObservedAt.IsZero() {
		input.ObservedAt = time.Now().UTC()
	}
	if input.Fingerprint == "" {
		input.Fingerprint = incidents.Fingerprint(string(input.Type), input.SourceCheckID, input.Title)
	}
	id := "inc_" + strings.TrimPrefix(input.Fingerprint, "incfp_")
	sourceCheckIDs, _ := json.Marshal(nonEmptyUnique([]string{input.SourceCheckID}))
	sourceResultIDs, _ := json.Marshal([]string{})
	evidenceJSON, _ := json.Marshal(input.Evidence)
	recommendationsJSON, _ := json.Marshal(input.Recommendations)
	evidenceItemsJSON, _ := json.Marshal(input.EvidenceItems)
	corroborationNeededJSON, _ := json.Marshal(input.CorroborationNeeded)
	safeActionsJSON, _ := json.Marshal(input.SafeActions)
	operatorActionsJSON, _ := json.Marshal(input.OperatorActions)
	rollbackOrFallbackJSON, _ := json.Marshal(input.RollbackOrFallback)
	postChecksJSON, _ := json.Marshal(input.PostChecks)
	remediationCandidatesJSON, _ := json.Marshal(input.RemediationCandidates)
	remediationArtifactsJSON, _ := json.Marshal(input.RemediationArtifacts)
	outcomeJSON, _ := json.Marshal(input.Outcome)
	var outcomeValue any
	if input.Outcome != nil {
		outcomeValue = outcomeJSON
	}
	_, err := s.db.ExecContext(
		ctx, `
		INSERT INTO incidents (
			id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			boot_id, previous_boot_id, source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count
		) VALUES (?, ?, ?, ?, 'open', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0)
		ON CONFLICT(fingerprint) DO UPDATE SET
			severity = CASE
				WHEN incidents.severity = 'critical' OR excluded.severity = 'critical' THEN 'critical'
				WHEN incidents.severity = 'warning' OR excluded.severity = 'warning' THEN 'warning'
				ELSE excluded.severity
			END,
			status = CASE WHEN incidents.status = 'resolved' THEN 'open' ELSE incidents.status END,
			title = excluded.title,
			summary = excluded.summary,
			last_seen_at = excluded.last_seen_at,
			updated_at = excluded.updated_at,
			resolved_at = CASE WHEN incidents.status = 'resolved' THEN NULL ELSE incidents.resolved_at END,
			boot_id = excluded.boot_id,
			previous_boot_id = excluded.previous_boot_id,
			source_check_ids_json = excluded.source_check_ids_json,
			evidence_json = excluded.evidence_json,
			recommendations_json = excluded.recommendations_json,
			diagnosis = excluded.diagnosis,
			confidence = excluded.confidence,
			evidence_items_json = excluded.evidence_items_json,
			corroboration_needed_json = excluded.corroboration_needed_json,
			safe_actions_json = excluded.safe_actions_json,
			operator_actions_json = excluded.operator_actions_json,
			rollback_or_fallback_json = excluded.rollback_or_fallback_json,
			post_checks_json = excluded.post_checks_json,
			remediation_candidates_json = excluded.remediation_candidates_json,
			remediation_artifacts_json = CASE
				WHEN excluded.remediation_artifacts_json IS NULL OR CAST(excluded.remediation_artifacts_json AS TEXT) IN ('null', '[]') THEN incidents.remediation_artifacts_json
				ELSE excluded.remediation_artifacts_json
			END,
			outcome_json = COALESCE(excluded.outcome_json, incidents.outcome_json),
			event_count = CASE WHEN incidents.status = 'resolved' THEN incidents.event_count + 1 ELSE incidents.event_count END
	`, id, input.Fingerprint, input.Type, input.Severity, input.Title, input.Summary,
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
		nullableString(input.BootID),
		nullableString(input.PreviousBootID),
		sourceCheckIDs,
		sourceResultIDs,
		evidenceJSON,
		recommendationsJSON,
		input.Diagnosis,
		input.Confidence,
		evidenceItemsJSON,
		corroborationNeededJSON,
		safeActionsJSON,
		operatorActionsJSON,
		rollbackOrFallbackJSON,
		postChecksJSON,
		remediationCandidatesJSON,
		remediationArtifactsJSON,
		outcomeValue,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert incident: %w", err)
	}
	incident, err := s.getIncidentByFingerprintSQLite(ctx, input.Fingerprint)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, sql.ErrNoRows
	}
	obsEvidenceJSON, _ := json.Marshal(input.Evidence)
	shouldRecord, err := s.shouldRecordIncidentObservationSQLite(ctx, incident.ID, input, obsEvidenceJSON)
	if err != nil {
		return nil, err
	}
	if shouldRecord {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO incident_observations (incident_id, observed_at, source_check_id, severity, status, message, evidence_json)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, incident.ID, input.ObservedAt.UTC().Format(time.RFC3339Nano), input.SourceCheckID, input.Severity, string(incident.Status), input.Summary, obsEvidenceJSON); err != nil {
			return nil, fmt.Errorf("insert incident observation: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE incidents
			SET observation_count = observation_count + 1
			WHERE id = ?
		`, incident.ID); err != nil {
			return nil, fmt.Errorf("update incident observation count: %w", err)
		}
		incident, err = s.getIncidentSQLite(ctx, incident.ID)
		if err != nil {
			return nil, err
		}
	}
	return incident, nil
}

func (s *Store) ensureIncidentContractColumns(ctx context.Context) error {
	columns := []struct {
		name string
		def  string
	}{
		{"diagnosis", "TEXT NOT NULL DEFAULT ''"},
		{"confidence", "TEXT NOT NULL DEFAULT ''"},
		{"evidence_items_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"corroboration_needed_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"safe_actions_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"operator_actions_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"rollback_or_fallback_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"post_checks_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"remediation_candidates_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"remediation_artifacts_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"outcome_json", "TEXT"},
	}
	for _, column := range columns {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE incidents ADD COLUMN %s %s", column.name, column.def)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") ||
				strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			return fmt.Errorf("add incidents.%s column: %w", column.name, err)
		}
	}
	return nil
}

func (s *Store) shouldRecordIncidentObservationSQLite(ctx context.Context, incidentID string, input incidents.UpsertInput, evidenceJSON []byte) (bool, error) {
	const observationQuietWindow = 30 * time.Minute
	row := s.db.QueryRowContext(ctx, `
		SELECT observed_at, severity, COALESCE(status, ''), message, evidence_json
		FROM incident_observations
		WHERE incident_id = ? AND COALESCE(source_check_id, '') = COALESCE(?, '')
		ORDER BY observed_at DESC
		LIMIT 1
	`, incidentID, nullableString(input.SourceCheckID))
	var observedRaw any
	var severity incidents.Severity
	var status string
	var message string
	var previousEvidenceJSON []byte
	if err := row.Scan(&observedRaw, &severity, &status, &message, &previousEvidenceJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("query latest incident observation: %w", err)
	}
	observedAt, err := parseDBTime(observedRaw)
	if err != nil {
		return false, fmt.Errorf("parse latest incident observation time: %w", err)
	}
	sameEvidence := string(previousEvidenceJSON) == string(evidenceJSON)
	sameObservation := severity == input.Severity && message == input.Summary && sameEvidence
	if sameObservation && input.ObservedAt.Sub(observedAt) < observationQuietWindow {
		return false, nil
	}
	return true, nil
}

func (s *Store) listIncidentsSQLite(ctx context.Context, filters incidents.ListFilters) (*incidents.ListResponse, error) {
	if filters.Limit <= 0 || filters.Limit > 200 {
		filters.Limit = 50
	}
	where := []string{"1=1"}
	args := []any{}
	if filters.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filters.Status)
	}
	if filters.Severity != "" {
		where = append(where, "severity = ?")
		args = append(args, filters.Severity)
	}
	if filters.Type != "" {
		where = append(where, "type = ?")
		args = append(args, filters.Type)
	}
	if filters.Since != nil {
		where = append(where, "updated_at >= ?")
		args = append(args, filters.Since.UTC().Format(time.RFC3339Nano))
	}
	if filters.Until != nil {
		where = append(where, "updated_at <= ?")
		args = append(args, filters.Until.UTC().Format(time.RFC3339Nano))
	}
	args = append(args, filters.Limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
			source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count, operator_notes
		FROM incidents
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY updated_at DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()
	var list []incidents.Incident
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *incident)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &incidents.ListResponse{Incidents: list, Total: len(list), Filters: filters}, nil
}

func (s *Store) getIncidentSQLite(ctx context.Context, id string) (*incidents.Incident, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
			source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count, operator_notes
		FROM incidents
		WHERE id = ?
	`, id)
	return scanIncident(row)
}

func (s *Store) getIncidentByFingerprintSQLite(ctx context.Context, fingerprint string) (*incidents.Incident, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
			source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count, operator_notes
		FROM incidents
		WHERE fingerprint = ?
	`, fingerprint)
	return scanIncident(row)
}

func (s *Store) listIncidentObservationsSQLite(ctx context.Context, incidentID string, limit int) ([]incidents.Observation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, incident_id, observed_at, COALESCE(source_check_id, ''), severity, COALESCE(status, ''), message, evidence_json
		FROM incident_observations
		WHERE incident_id = ?
		ORDER BY observed_at DESC
		LIMIT ?
	`, incidentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var observations []incidents.Observation
	for rows.Next() {
		var obs incidents.Observation
		var observedRaw any
		var evidenceJSON []byte
		if err := rows.Scan(&obs.ID, &obs.IncidentID, &observedRaw, &obs.SourceCheckID, &obs.Severity, &obs.Status, &obs.Message, &evidenceJSON); err != nil {
			return nil, err
		}
		observedAt, err := parseDBTime(observedRaw)
		if err != nil {
			return nil, err
		}
		obs.ObservedAt = observedAt.UTC()
		_ = json.Unmarshal(evidenceJSON, &obs.Evidence)
		observations = append(observations, obs)
	}
	return observations, rows.Err()
}

func (s *Store) updateIncidentStatusSQLite(ctx context.Context, incidentID string, status incidents.Status, note string) (*incidents.Incident, error) {
	current, err := s.getIncidentSQLite(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}
	now := time.Now().UTC()
	var acknowledgedAt any
	var resolvedAt any
	var ignoredAt any
	if status == incidents.StatusAcknowledged {
		acknowledgedAt = now.Format(time.RFC3339Nano)
	}
	if status == incidents.StatusResolved {
		resolvedAt = now.Format(time.RFC3339Nano)
	}
	if status == incidents.StatusIgnored {
		ignoredAt = now.Format(time.RFC3339Nano)
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = ?, updated_at = ?, acknowledged_at = COALESCE(?, acknowledged_at),
			resolved_at = COALESCE(?, resolved_at), ignored_at = COALESCE(?, ignored_at),
			operator_notes = CASE WHEN ? = '' THEN operator_notes ELSE trim(operator_notes || char(10) || ?) END
		WHERE id = ?
	`, status, now.Format(time.RFC3339Nano), acknowledgedAt, resolvedAt, ignoredAt, note, note, incidentID)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO incident_status_history (incident_id, from_status, to_status, note, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, incidentID, current.Status, status, note, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.getIncidentSQLite(ctx, incidentID)
}

func (s *Store) recordIncidentRemediationArtifactSQLite(ctx context.Context, incidentID string, artifact incidents.RemediationArtifact) (*incidents.Incident, error) {
	current, err := s.getIncidentSQLite(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}
	artifacts := upsertRemediationArtifact(current.RemediationArtifacts, artifact)
	payload, err := json.Marshal(artifacts)
	if err != nil {
		return nil, fmt.Errorf("marshal remediation artifacts: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET remediation_artifacts_json = ?, updated_at = ?
		WHERE id = ?
	`, payload, now, incidentID); err != nil {
		return nil, fmt.Errorf("record remediation artifact: %w", err)
	}
	return s.getIncidentSQLite(ctx, incidentID)
}

func (s *Store) recordIncidentRemediationOutcomeSQLite(ctx context.Context, incidentID string, outcome incidents.Outcome) (*incidents.Incident, error) {
	current, err := s.getIncidentSQLite(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}
	if outcome.ReportedAt.IsZero() {
		outcome.ReportedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(outcome)
	if err != nil {
		return nil, fmt.Errorf("marshal remediation outcome: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET outcome_json = ?, updated_at = ?,
			operator_notes = CASE WHEN ? = '' THEN operator_notes ELSE trim(operator_notes || char(10) || ?) END
		WHERE id = ?
	`, payload, now, outcome.Note, outcome.Note, incidentID); err != nil {
		return nil, fmt.Errorf("record remediation outcome: %w", err)
	}
	return s.getIncidentSQLite(ctx, incidentID)
}

func upsertRemediationArtifact(existing []incidents.RemediationArtifact, artifact incidents.RemediationArtifact) []incidents.RemediationArtifact {
	for i := range existing {
		if existing[i].ID == artifact.ID || existing[i].RemediationID == artifact.RemediationID {
			existing[i] = artifact
			return existing
		}
	}
	return append(existing, artifact)
}

func scanIncident(row rowScanner) (*incidents.Incident, error) {
	var incident incidents.Incident
	var detectedRaw, lastSeenRaw, updatedRaw any
	var resolvedRaw, acknowledgedRaw, ignoredRaw any
	var sourceChecksJSON, sourceResultsJSON, evidenceJSON, recommendationsJSON []byte
	var evidenceItemsJSON, corroborationNeededJSON, safeActionsJSON, operatorActionsJSON []byte
	var rollbackOrFallbackJSON, postChecksJSON, remediationCandidatesJSON, remediationArtifactsJSON []byte
	var outcomeJSON []byte
	if err := row.Scan(
		&incident.ID, &incident.Fingerprint, &incident.Type, &incident.Severity, &incident.Status,
		&incident.Title, &incident.Summary, &detectedRaw, &lastSeenRaw, &updatedRaw,
		&resolvedRaw, &acknowledgedRaw, &ignoredRaw, &incident.BootID, &incident.PreviousBootID,
		&sourceChecksJSON, &sourceResultsJSON, &evidenceJSON, &recommendationsJSON,
		&incident.Diagnosis, &incident.Confidence, &evidenceItemsJSON, &corroborationNeededJSON, &safeActionsJSON, &operatorActionsJSON,
		&rollbackOrFallbackJSON, &postChecksJSON, &remediationCandidatesJSON, &remediationArtifactsJSON, &outcomeJSON,
		&incident.EventCount, &incident.ObservationCount, &incident.OperatorNotes,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var err error
	if incident.DetectedAt, err = parseDBTime(detectedRaw); err != nil {
		return nil, err
	}
	if incident.LastSeenAt, err = parseDBTime(lastSeenRaw); err != nil {
		return nil, err
	}
	if incident.UpdatedAt, err = parseDBTime(updatedRaw); err != nil {
		return nil, err
	}
	incident.ResolvedAt = parseOptionalTimePtr(resolvedRaw)
	incident.AcknowledgedAt = parseOptionalTimePtr(acknowledgedRaw)
	incident.IgnoredAt = parseOptionalTimePtr(ignoredRaw)
	_ = json.Unmarshal(sourceChecksJSON, &incident.SourceCheckIDs)
	_ = json.Unmarshal(sourceResultsJSON, &incident.SourceResultIDs)
	_ = json.Unmarshal(evidenceJSON, &incident.Evidence)
	_ = json.Unmarshal(recommendationsJSON, &incident.Recommendations)
	_ = json.Unmarshal(evidenceItemsJSON, &incident.EvidenceItems)
	_ = json.Unmarshal(corroborationNeededJSON, &incident.CorroborationNeeded)
	_ = json.Unmarshal(safeActionsJSON, &incident.SafeActions)
	_ = json.Unmarshal(operatorActionsJSON, &incident.OperatorActions)
	_ = json.Unmarshal(rollbackOrFallbackJSON, &incident.RollbackOrFallback)
	_ = json.Unmarshal(postChecksJSON, &incident.PostChecks)
	_ = json.Unmarshal(remediationCandidatesJSON, &incident.RemediationCandidates)
	_ = json.Unmarshal(remediationArtifactsJSON, &incident.RemediationArtifacts)
	if len(outcomeJSON) > 0 && string(outcomeJSON) != "null" {
		var outcome incidents.Outcome
		if err := json.Unmarshal(outcomeJSON, &outcome); err == nil {
			incident.Outcome = &outcome
		}
	}
	return &incident, nil
}

func parseOptionalTimePtr(raw any) *time.Time {
	ts, ok := parseNullableDBTime(raw)
	if !ok {
		return nil
	}
	ts = ts.UTC()
	return &ts
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nonEmptyUnique(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// ensureIncidentTypeConstraint rebuilds a legacy incidents table whose CHECK
// constraint predates the host_pressure incident type. SQLite cannot alter a
// CHECK in place, so this is the documented twelve-step rebuild: create the
// table from the embedded schema under a temporary name, copy every row,
// drop the old table, rename, recreate the indexes. Foreign keys from
// incident_observations and remediation rows reference incidents(id); the
// rows keep their ids, so the references stay valid. A store whose table
// already admits host_pressure returns before touching anything.
func (s *Store) ensureIncidentTypeConstraint(ctx context.Context) error {
	var ddl string
	if err := s.db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='incidents'`).Scan(&ddl); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("read incidents table definition: %w", err)
	}
	if strings.Contains(ddl, "'"+string(incidents.TypeHostPressure)+"'") {
		return nil
	}
	create, ok := incidentsTableDDL()
	if !ok {
		return fmt.Errorf("embedded schema has no incidents table")
	}
	columns, err := s.incidentColumns(ctx)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys for incidents rebuild: %w", err)
	}
	defer func() { _, _ = s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`) }()
	// ALTER TABLE ... RENAME re-validates every view in the database, and a
	// production database on this host carries a dangling view
	// (latest_health_results over a health_results_legacy table an earlier
	// migration dropped). The rename must not fail on debt it did not create,
	// so it runs with the pre-3.26 rename semantics that skip that check.
	if _, err := s.db.ExecContext(ctx, `PRAGMA legacy_alter_table = ON`); err != nil {
		return fmt.Errorf("enable legacy rename for incidents rebuild: %w", err)
	}
	defer func() { _, _ = s.db.ExecContext(ctx, `PRAGMA legacy_alter_table = OFF`) }()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	statements := []string{
		strings.Replace(create, "CREATE TABLE IF NOT EXISTS incidents (", "CREATE TABLE incidents_rebuild (", 1),
		fmt.Sprintf("INSERT INTO incidents_rebuild (%s) SELECT %s FROM incidents", columns, columns),
		"DROP TABLE incidents",
		"ALTER TABLE incidents_rebuild RENAME TO incidents",
		"CREATE INDEX IF NOT EXISTS idx_incidents_status_updated ON incidents (status, updated_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_incidents_type_severity ON incidents (type, severity, updated_at DESC)",
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("rebuild incidents table for host_pressure: %w", err)
		}
	}
	return tx.Commit()
}

// incidentsTableDDL extracts the incidents CREATE statement from the
// embedded schema so the rebuild and a fresh database agree byte for byte.
func incidentsTableDDL() (string, bool) {
	const head = "CREATE TABLE IF NOT EXISTS incidents ("
	start := strings.Index(schemaSQL, head)
	if start < 0 {
		return "", false
	}
	end := strings.Index(schemaSQL[start:], "\n);")
	if end < 0 {
		return "", false
	}
	return schemaSQL[start : start+end+len("\n);")], true
}

// incidentColumns lists the live table's columns that the embedded schema
// also declares, in table order, so a rebuild copies only shared columns.
func (s *Store) incidentColumns(ctx context.Context) (string, error) {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(incidents)`)
	if err != nil {
		return "", fmt.Errorf("read incidents columns: %w", err)
	}
	defer rows.Close()
	create, _ := incidentsTableDDL()
	var names []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return "", err
		}
		if strings.Contains(create, "\n    "+name+" ") {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(names, ", "), nil
}
