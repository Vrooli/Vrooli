package interventions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SignalCandidate is an operator message that has a classified intervention
// row and no current signal, or an unclassified signal whose retry is due.
type SignalCandidate struct {
	EventID    string
	RunID      string
	Sequence   int64
	OccurredAt string
	Attempts   int
}

// SignalRecord is one signal row to save.
type SignalRecord struct {
	SignalCandidate
	Answer        SignalAnswer
	NextAttemptAt time.Time
}

// PendingSignals returns up to limit candidates, newest first. A non-zero
// recordedSince limits them to interventions recorded since then (the live
// tail once the back-fill is done). Duplicate and initial messages are not
// candidates: a forked rollout's replay is not a second report, and the
// opening message follows no agent work.
func (s *Store) PendingSignals(ctx context.Context, version string, recordedSince, now time.Time, limit int) ([]SignalCandidate, error) {
	query := `SELECT i.event_id, i.run_id, i.sequence, i.occurred_at, COALESCE(s.attempts, 0)
FROM operator_interventions i
LEFT JOIN operator_signals s ON s.event_id = i.event_id AND s.signal_version = ?
WHERE i.classifier_version = ? AND i.status IN ('classified', 'unclassified')
  AND (s.event_id IS NULL OR (s.status = 'unclassified' AND s.next_attempt_at <= ?))`
	args := []any{version, ClassifierVersion, formatTime(now)}
	if !recordedSince.IsZero() {
		query += ` AND (i.recorded_at >= ? OR s.status = 'unclassified')`
		args = append(args, formatTime(recordedSince))
	}
	rows, err := s.db.QueryxContext(ctx, query+` ORDER BY i.occurred_at DESC, i.event_id LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("read pending signals: %w", err)
	}
	defer rows.Close()
	var out []SignalCandidate
	for rows.Next() {
		var c SignalCandidate
		if err := rows.Scan(&c.EventID, &c.RunID, &c.Sequence, &c.OccurredAt, &c.Attempts); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SaveSignals upserts records. Labels and probabilities are stored, never text.
func (s *Store) SaveSignals(ctx context.Context, version string, records []SignalRecord, now time.Time) error {
	for _, r := range records {
		probabilities := "{}"
		if len(r.Answer.Probabilities) > 0 {
			body, _ := json.Marshal(r.Answer.Probabilities)
			probabilities = string(body)
		}
		var probability any
		if p := r.Answer.Probability(); p != nil {
			probability = *p
		}
		attempts, next := r.Attempts, ""
		if r.Answer.Status == StatusUnclassified {
			attempts++
			next = formatTime(r.NextAttemptAt)
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO operator_signals (event_id, signal_version, run_id, occurred_at, status, label, probability, probabilities_json, method, provider,
  model, resource_role, unclassified_reason, attempts, next_attempt_at, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(event_id, signal_version) DO UPDATE SET status = excluded.status, label = excluded.label, probability = excluded.probability,
  probabilities_json = excluded.probabilities_json, method = excluded.method, provider = excluded.provider, model = excluded.model,
  resource_role = excluded.resource_role, unclassified_reason = excluded.unclassified_reason, attempts = excluded.attempts,
  next_attempt_at = excluded.next_attempt_at, recorded_at = excluded.recorded_at`,
			r.EventID, version, r.RunID, r.OccurredAt, string(r.Answer.Status), string(r.Answer.Signal), probability, probabilities, r.Answer.Method,
			r.Answer.Provider, r.Answer.Model, r.Answer.ResourceRole, r.Answer.Reason, attempts, next, formatTime(now)); err != nil {
			return fmt.Errorf("save signal %s: %w", r.EventID, err)
		}
	}
	return nil
}

// PruneSignalVersions deletes signals of every version but keep.
func (s *Store) PruneSignalVersions(ctx context.Context, keep string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM operator_signals WHERE signal_version <> ?`, keep)
	if err != nil {
		return 0, fmt.Errorf("prune signal versions: %w", err)
	}
	return result.RowsAffected()
}

// CountSignals counts stored signal rows.
func (s *Store) CountSignals(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operator_signals`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count signals: %w", err)
	}
	return count, nil
}

// PurgeSignals deletes every stored signal: what the off switch's owner runs to
// remove the derived labels as well as stop producing them.
func (s *Store) PurgeSignals(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM operator_signals`)
	if err != nil {
		return 0, fmt.Errorf("purge signals: %w", err)
	}
	return result.RowsAffected()
}

// signalBackfillMarker is the schema_migrations_done name recording that the
// back-fill of a signal version finished; a new version backfills again.
func signalBackfillMarker(version string) string { return "operator-signal-backfill:" + version }

// SignalBackfillDone reports whether the back-fill for version finished.
func (s *Store) SignalBackfillDone(ctx context.Context, version string) (bool, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM schema_migrations_done WHERE name = ?`, signalBackfillMarker(version)).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// MarkSignalBackfilled records the marker.
func (s *Store) MarkSignalBackfilled(ctx context.Context, version string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations_done (name, completed_at) VALUES (?, ?)`, signalBackfillMarker(version), formatTime(now))
	return err
}

// SignalObservation is one operator message of a window with its signal and
// the run facts the measure groups by.
type SignalObservation struct {
	EventID      string
	RunID        string
	OccurredAt   string
	Status       Status // classified | unclassified; Unclassified also covers "no signal yet"
	Signal       Signal
	Probability  *float64
	Distribution map[Signal]float64
	SignalModel  string
	ActualModel  string
	ModeID       string
	ModeRelation string
	Harness      string
}

// SignalObservations lists the operator messages of [from, to) that follow
// agent work, with their signal for version when one exists. A message with no
// signal yet is Status unclassified, so coverage counts it.
func (s *Store) SignalObservations(ctx context.Context, version string, from, to time.Time) ([]SignalObservation, error) {
	rows, err := s.db.QueryxContext(ctx, `SELECT i.event_id, i.run_id, i.occurred_at, COALESCE(s.status, 'unclassified'), COALESCE(s.label, ''), s.probability,
  COALESCE(s.probabilities_json, '{}'), COALESCE(s.model, ''), COALESCE(r.actual_model, ''), COALESCE(r.mode_id, ''), COALESCE(r.mode_relation, ''), COALESCE(r.harness, '')
FROM operator_interventions i
LEFT JOIN operator_signals s ON s.event_id = i.event_id AND s.signal_version = ?
LEFT JOIN run_ledger_facts r ON r.run_id = i.run_id
WHERE i.classifier_version = ? AND i.status IN ('classified', 'unclassified') AND i.occurred_at >= ? AND i.occurred_at < ?`,
		version, ClassifierVersion, formatTime(from), formatTime(to))
	if err != nil {
		return nil, fmt.Errorf("read signal observations: %w", err)
	}
	defer rows.Close()
	var out []SignalObservation
	for rows.Next() {
		var (
			o            SignalObservation
			status       string
			label        string
			probability  sql.NullFloat64
			distribution string
		)
		if err := rows.Scan(&o.EventID, &o.RunID, &o.OccurredAt, &status, &label, &probability, &distribution, &o.SignalModel, &o.ActualModel, &o.ModeID, &o.ModeRelation, &o.Harness); err != nil {
			return nil, err
		}
		o.Status, o.Signal = Status(status), Signal(label)
		if probability.Valid {
			o.Probability = &probability.Float64
		}
		if distribution != "" && distribution != "{}" {
			_ = json.Unmarshal([]byte(distribution), &o.Distribution)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
