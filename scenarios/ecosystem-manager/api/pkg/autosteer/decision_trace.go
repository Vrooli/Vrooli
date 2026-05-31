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
			score_before, score_after, realized_delta, tokens_used
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`
	_, err = s.db.Exec(query,
		taskID, profileID, scenarioName, e.Iteration, e.ChosenSkill,
		e.HeaviestDimension, e.Rationale, scoresJSON, e.Fingerprint,
		e.ScoreBefore, e.ScoreAfter, e.RealizedDelta, e.TokensUsed,
	)
	if err != nil {
		return fmt.Errorf("append decision trace: %w", err)
	}
	return nil
}

// SetRealized fills in the realized outcome of an iteration after re-audit: the
// new score, realized delta, token cost, and the per-dimension findings flow.
func (s *TraceStore) SetRealized(taskID string, e DecisionTraceEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	closedJSON, err := marshalDimCounts(e.ClosedByDimension)
	if err != nil {
		return fmt.Errorf("marshal closed_by_dimension: %w", err)
	}
	introducedJSON, err := marshalDimCounts(e.IntroducedByDimension)
	if err != nil {
		return fmt.Errorf("marshal introduced_by_dimension: %w", err)
	}
	query := `
		UPDATE decision_trace
		SET score_after = $1, realized_delta = $2, tokens_used = $3,
		    closed_by_dimension = $4, introduced_by_dimension = $5,
		    regressed = $6, veto_applied = $7
		WHERE task_id = $8 AND iteration = $9
	`
	if _, err := s.db.Exec(query, e.ScoreAfter, e.RealizedDelta, e.TokensUsed,
		closedJSON, introducedJSON, e.Regressed, e.VetoApplied, taskID, e.Iteration); err != nil {
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
		`UPDATE decision_trace SET halt_reason = $1 WHERE task_id = $2 AND iteration = $3`,
		reason, taskID, iteration,
	); err != nil {
		return fmt.Errorf("set halt reason: %w", err)
	}
	return nil
}

// marshalDimCounts marshals a per-dimension count map to JSON, returning a nil
// slice (SQL NULL) for an empty map so the column stays clean when there is no
// findings flow to record.
func marshalDimCounts(m map[string]int) ([]byte, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// GetTrace returns the decision trace for a task ordered by iteration.
func (s *TraceStore) GetTrace(taskID string) ([]DecisionTraceEntry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	query := `
		SELECT iteration, chosen_skill, heaviest_dimension, rationale,
		       dimension_scores, fingerprint, score_before, score_after,
		       realized_delta, tokens_used, closed_by_dimension,
		       introduced_by_dimension, regressed, veto_applied, halt_reason,
		       created_at
		FROM decision_trace
		WHERE task_id = $1
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
		var scoresJSON, closedJSON, introducedJSON []byte
		if err := rows.Scan(
			&e.Iteration, &e.ChosenSkill, &e.HeaviestDimension, &e.Rationale,
			&scoresJSON, &e.Fingerprint, &e.ScoreBefore, &e.ScoreAfter,
			&e.RealizedDelta, &e.TokensUsed, &closedJSON, &introducedJSON,
			&e.Regressed, &e.VetoApplied, &e.HaltReason, &e.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan decision trace: %w", err)
		}
		if !isNullJSON(scoresJSON) {
			if err := json.Unmarshal(scoresJSON, &e.DimensionScores); err != nil {
				return nil, fmt.Errorf("unmarshal dimension scores: %w", err)
			}
		}
		if !isNullJSON(closedJSON) {
			if err := json.Unmarshal(closedJSON, &e.ClosedByDimension); err != nil {
				return nil, fmt.Errorf("unmarshal closed_by_dimension: %w", err)
			}
		}
		if !isNullJSON(introducedJSON) {
			if err := json.Unmarshal(introducedJSON, &e.IntroducedByDimension); err != nil {
				return nil, fmt.Errorf("unmarshal introduced_by_dimension: %w", err)
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
