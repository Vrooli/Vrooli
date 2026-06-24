package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// TraceStore persists the controller's per-iteration decision trace to the
// decision_trace table so the reasoning survives run finalization (the live
// state.Trace is removed when the run completes).
//
// seam: TraceStore is the durable write side of the decision trace; the
// orchestrator appends a row at SELECT time and fills the realized delta after
// MEASURE.
type TraceStore struct {
	db *sql.DB
}

// NewTraceStore creates a TraceStore.
func NewTraceStore(db *sql.DB) *TraceStore {
	return &TraceStore{db: db}
}

// Append inserts a new decision-trace row (ScoreAfter/RealizedDelta are filled
// later by SetRealized once the chosen skill has run and been re-audited).
func (s *TraceStore) Append(taskID, profileID, scenarioName string, e DecisionTraceEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	scoresJSON, err := json.Marshal(e.DimensionScores)
	if err != nil {
		return fmt.Errorf("marshal dimension scores: %w", err)
	}
	query := `
		INSERT INTO decision_trace (
			task_id, profile_id, scenario_name, iteration, chosen_skill,
			heaviest_dimension, rationale, dimension_scores, fingerprint,
			score_before, score_after, realized_delta
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`
	_, err = s.db.Exec(query,
		taskID, profileID, scenarioName, e.Iteration, e.ChosenSkill,
		e.HeaviestDimension, e.Rationale, scoresJSON, e.Fingerprint,
		e.ScoreBefore, e.ScoreAfter, e.RealizedDelta,
	)
	if err != nil {
		return fmt.Errorf("append decision trace: %w", err)
	}
	return nil
}

// SetRealized fills in the realized outcome of an iteration after re-audit: the
// new score, realized delta, and the anti-gaming verdict.
func (s *TraceStore) SetRealized(taskID string, e DecisionTraceEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	query := `
		UPDATE decision_trace
		SET score_after = ?, realized_delta = ?, gaming_cause = ?
		WHERE task_id = ? AND iteration = ?
	`
	if _, err := s.db.Exec(query, e.ScoreAfter, e.RealizedDelta, e.GamingCause,
		taskID, e.Iteration); err != nil {
		return fmt.Errorf("set realized decision trace: %w", err)
	}
	return nil
}

// SetHalt records the terminal halt reason on an iteration's trace row.
func (s *TraceStore) SetHalt(taskID string, iteration int, reason string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if _, err := s.db.Exec(
		`UPDATE decision_trace SET halt_reason = ? WHERE task_id = ? AND iteration = ?`,
		reason, taskID, iteration,
	); err != nil {
		return fmt.Errorf("set halt reason: %w", err)
	}
	return nil
}

// GetTrace returns the decision trace for a task ordered by iteration.
func (s *TraceStore) GetTrace(taskID string) ([]DecisionTraceEntry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	query := `
		SELECT iteration, chosen_skill, heaviest_dimension, rationale,
		       dimension_scores, fingerprint, score_before, score_after,
		       realized_delta, gaming_cause, halt_reason, created_at
		FROM decision_trace
		WHERE task_id = ?
		ORDER BY iteration ASC
	`
	rows, err := s.db.Query(query, taskID)
	if err != nil {
		return nil, fmt.Errorf("query decision trace: %w", err)
	}
	defer rows.Close()

	out := make([]DecisionTraceEntry, 0)
	for rows.Next() {
		var e DecisionTraceEntry
		var scoresJSON []byte
		if err := rows.Scan(
			&e.Iteration, &e.ChosenSkill, &e.HeaviestDimension, &e.Rationale,
			&scoresJSON, &e.Fingerprint, &e.ScoreBefore, &e.ScoreAfter,
			&e.RealizedDelta, &e.GamingCause, &e.HaltReason, &e.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan decision trace: %w", err)
		}
		if !isNullJSON(scoresJSON) {
			if err := json.Unmarshal(scoresJSON, &e.DimensionScores); err != nil {
				return nil, fmt.Errorf("unmarshal dimension scores: %w", err)
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
