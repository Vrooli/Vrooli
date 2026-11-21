// This file reads the transcript facts the transcript step-back triggers use
// (stepback.Transcript) with narrow aggregate queries, so a run tree or effort
// board never decodes whole event streams to compute them.
package event

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/stepback"

	"github.com/google/uuid"
)

// transcriptFactsBatch bounds the run IDs bound into one statement.
const transcriptFactsBatch = 500

// TranscriptFacts reads each run's turns, compactions and sub-agent spawns
// from its stored events. Turns come from message events that carry a turn
// ID: a turn ends at its terminal record, and the newest unterminated turn
// runs to the run's latest stored event. Runs without stored events are
// absent from the result.
func (s *SQLiteStore) TranscriptFacts(ctx context.Context, runIDs []uuid.UUID) (map[uuid.UUID]stepback.Transcript, error) {
	result := make(map[uuid.UUID]stepback.Transcript, len(runIDs))
	for start := 0; start < len(runIDs); start += transcriptFactsBatch {
		ids := runIDs[start:min(start+transcriptFactsBatch, len(runIDs))]
		if err := s.readTranscriptFacts(ctx, ids, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *SQLiteStore) readTranscriptFacts(ctx context.Context, ids []uuid.UUID, result map[uuid.UUID]stepback.Transcript) error {
	in := "run_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + ")"
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	var latest []struct {
		RunID uuid.UUID  `db:"run_id"`
		At    sqliteTime `db:"timestamp"`
	}
	query := fmt.Sprintf(`SELECT e.run_id, e.timestamp FROM run_events e
		JOIN (SELECT run_id, MAX(sequence) AS high FROM run_events WHERE %s GROUP BY run_id) m
		ON e.run_id = m.run_id AND e.sequence = m.high`, in)
	if err := s.db.SelectContext(ctx, &latest, query, args...); err != nil {
		return dbError("transcript_facts_latest", err)
	}
	lastEvent := make(map[uuid.UUID]sqliteTime, len(latest))
	for _, row := range latest {
		lastEvent[row.RunID] = row.At
		result[row.RunID] = stepback.Transcript{}
	}

	var turns []struct {
		RunID   uuid.UUID  `db:"run_id"`
		TurnID  string     `db:"turn_id"`
		FirstAt sqliteTime `db:"first_at"`
		LastAt  sqliteTime `db:"last_at"`
		EndedAt sqliteTime `db:"ended_at"`
	}
	query = fmt.Sprintf(`SELECT run_id, json_extract(data, '$.turnId') AS turn_id,
		MIN(timestamp) AS first_at, MAX(timestamp) AS last_at,
		MAX(CASE WHEN json_extract(data, '$.terminal') = 1 THEN timestamp END) AS ended_at
		FROM run_events WHERE %s AND event_type = ? AND COALESCE(json_extract(data, '$.turnId'), '') != ''
		GROUP BY run_id, turn_id`, in)
	if err := s.db.SelectContext(ctx, &turns, query, append(args, string(domain.EventTypeMessage))...); err != nil {
		return dbError("transcript_facts_turns", err)
	}
	byRun := make(map[uuid.UUID][]stepback.Turn)
	for _, row := range turns {
		turn := stepback.Turn{ID: row.TurnID, Start: stepback.TurnStart(row.TurnID, row.FirstAt.Time()), End: row.LastAt.Time()}
		if ended := row.EndedAt.Time(); !ended.IsZero() {
			turn.End, turn.Ended = ended, true
		}
		byRun[row.RunID] = append(byRun[row.RunID], turn)
	}
	for runID, runTurns := range byRun {
		sort.Slice(runTurns, func(i, j int) bool { return runTurns[i].Start.Before(runTurns[j].Start) })
		// Tool calls and compactions continue after a turn's last message; the
		// newest open turn lasts until the run's latest stored event.
		if newest := &runTurns[len(runTurns)-1]; !newest.Ended {
			if last := lastEvent[runID].Time(); last.After(newest.End) {
				newest.End = last
			}
		}
		facts := result[runID]
		facts.Turns = runTurns
		result[runID] = facts
	}

	var compactions []struct {
		RunID    uuid.UUID      `db:"run_id"`
		At       sqliteTime     `db:"timestamp"`
		Goal     sql.NullString `db:"goal"`
		Operator sql.NullInt64  `db:"operator_kept"`
	}
	query = fmt.Sprintf(`SELECT run_id, timestamp, json_extract(data, '$.goalObjective') AS goal,
		json_extract(data, '$.operatorMessagesKept') AS operator_kept
		FROM run_events WHERE %s AND event_type = ? ORDER BY run_id, sequence`, in)
	if err := s.db.SelectContext(ctx, &compactions, query, append(args, string(domain.EventTypeCompaction))...); err != nil {
		return dbError("transcript_facts_compactions", err)
	}
	for _, row := range compactions {
		facts := result[row.RunID]
		compaction := stepback.Compaction{At: row.At.Time(), GoalObjective: row.Goal.String}
		if row.Operator.Valid {
			kept := int(row.Operator.Int64)
			compaction.OperatorMessagesKept = &kept
		}
		facts.Compactions = append(facts.Compactions, compaction)
		result[row.RunID] = facts
	}

	var spawns []struct {
		RunID     uuid.UUID `db:"run_id"`
		Spawns    int       `db:"spawns"`
		Inherited int       `db:"inherited"`
	}
	query = fmt.Sprintf(`SELECT run_id, COUNT(*) AS spawns,
		SUM(COALESCE(json_extract(data, '$.input.fork_turns'), '') != 'none') AS inherited
		FROM run_events WHERE %s AND event_type = ? AND json_extract(data, '$.toolName') = 'spawn_agent'
		GROUP BY run_id`, in)
	if err := s.db.SelectContext(ctx, &spawns, query, append(args, string(domain.EventTypeToolCall))...); err != nil {
		return dbError("transcript_facts_spawns", err)
	}
	for _, row := range spawns {
		facts := result[row.RunID]
		facts.Spawns, facts.SpawnsWithHistory = row.Spawns, row.Inherited
		result[row.RunID] = facts
	}
	return nil
}
