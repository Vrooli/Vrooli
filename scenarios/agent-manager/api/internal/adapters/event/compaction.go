package event

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// compactionScanWindow bounds the run_events rowids examined per call, so
	// one reconcile tick reads a fixed slice of history rather than all of it.
	compactionScanWindow = 20_000
	// compactionPassPause spaces complete passes over history. Payloads only
	// become eligible as they age past the cutoff, so an earlier pass would
	// find little to do.
	compactionPassPause = 6 * time.Hour
	// compactionRewriteBatch bounds the rows rewritten per write transaction.
	compactionRewriteBatch = 100
	// compactedValueMarker labels an excerpt so no reader mistakes it for the
	// complete original.
	compactedValueMarker = "agent-manager compacted"
)

// compactionCursor is in-memory: a restart re-examines history from the start
// once, in the same bounded windows.
type compactionCursor struct {
	mu       sync.Mutex
	next     int64
	resumeAt time.Time
}

type compactionCandidate struct {
	RowID     int64  `db:"row_id"`
	ID        string `db:"id"`
	RunID     string `db:"run_id"`
	EventType string `db:"event_type"`
	Data      string `db:"data"`
}

type compactionRewrite struct {
	id, runID, before, after string
}

// CompactImportedToolPayloads rewrites up to limit tool_call and tool_result
// payloads of imported runs that are older than cutoff and larger than
// minBytes. Each oversized value keeps minBytes/4 bytes at its head and at its
// tail, joined by a marker with the original byte count and sha256. Prose and
// every other event type are untouched, and no event is deleted.
//
// Each rewrite queues a conversation-search change in the same transaction,
// so the lexical projection can never keep text its source no longer holds.
func (s *SQLiteStore) CompactImportedToolPayloads(ctx context.Context, cutoff time.Time, minBytes, limit int) (int, error) {
	if minBytes <= 0 || limit <= 0 {
		return 0, fmt.Errorf("imported payload compaction needs a positive size threshold and batch limit")
	}
	s.compaction.mu.Lock()
	defer s.compaction.mu.Unlock()
	now := time.Now()
	if now.Before(s.compaction.resumeAt) {
		return 0, nil
	}
	var maxRowID int64
	if err := s.db.GetContext(ctx, &maxRowID, `SELECT COALESCE(MAX(rowid), 0) FROM run_events`); err != nil {
		return 0, dbError("compaction_max_rowid", err)
	}
	start := s.compaction.next
	end := start + compactionScanWindow
	var candidates []compactionCandidate
	// CROSS JOIN keeps run_events as the outer loop, so the rowid window, not
	// the imported-run count, bounds the scan.
	if err := s.db.SelectContext(ctx, &candidates, `SELECT e.rowid AS row_id, e.id, e.run_id, e.event_type, e.data
FROM run_events e CROSS JOIN runs r
WHERE e.rowid > ? AND e.rowid <= ? AND r.id = e.run_id AND r.execution_mode = 'imported'
  AND e.event_type IN ('tool_call', 'tool_result') AND e.timestamp < ? AND length(e.data) > ?
ORDER BY e.rowid LIMIT ?`, start, end, sqliteTime(cutoff), minBytes, limit); err != nil {
		return 0, dbError("select_compaction_candidates", err)
	}
	next := end
	if len(candidates) == limit {
		next = candidates[len(candidates)-1].RowID
	}
	rewrites := make([]compactionRewrite, 0, len(candidates))
	for _, candidate := range candidates {
		if compacted, ok := compactToolPayload(candidate.EventType, []byte(candidate.Data), minBytes/4); ok {
			rewrites = append(rewrites, compactionRewrite{id: candidate.ID, runID: candidate.RunID, before: candidate.Data, after: string(compacted)})
		}
	}
	compacted := 0
	for len(rewrites) > 0 {
		batch := rewrites[:min(len(rewrites), compactionRewriteBatch)]
		rewrites = rewrites[len(batch):]
		applied, err := s.applyCompaction(ctx, batch, now)
		compacted += applied
		if err != nil {
			return compacted, err
		}
	}
	if next >= maxRowID {
		s.compaction.next, s.compaction.resumeAt = 0, now.Add(compactionPassPause)
	} else {
		s.compaction.next = next
	}
	return compacted, nil
}

func (s *SQLiteStore) applyCompaction(ctx context.Context, rewrites []compactionRewrite, now time.Time) (int, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, dbError("compaction_connection", err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, dbError("begin_compaction", err)
	}
	defer tx.Rollback()
	stamp := now.UTC().Format(time.RFC3339Nano)
	applied := 0
	for _, rewrite := range rewrites {
		// Matching the selected bytes skips a payload that changed since. Append
		// stores data as a BLOB, and SQLite never equates BLOB with TEXT, so
		// both sides compare as bytes and the rewrite keeps Append's class.
		result, err := tx.ExecContext(ctx, `UPDATE run_events SET data = ? WHERE id = ? AND CAST(data AS BLOB) = CAST(? AS BLOB)`, []byte(rewrite.after), rewrite.id, rewrite.before)
		if err != nil {
			return 0, dbError("compact_event_payload", err)
		}
		if changed, err := result.RowsAffected(); err != nil || changed == 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_search_changes (operation, source_run_id, source_event_id, created_at) VALUES ('upsert_run', ?, ?, ?)`, rewrite.runID, rewrite.id, stamp); err != nil {
			return 0, dbError("queue_compaction_reindex", err)
		}
		applied++
	}
	if err := tx.Commit(); err != nil {
		return 0, dbError("commit_compaction", err)
	}
	return applied, nil
}

// compactToolPayload returns the compacted payload, or false when the event is
// not a compactable tool payload or compaction would not shrink it.
func compactToolPayload(eventType string, raw []byte, keep int) ([]byte, bool) {
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil || payload == nil {
		return nil, false
	}
	switch eventType {
	case "tool_result":
		for _, key := range []string{"output", "error"} {
			if text, ok := payload[key].(string); ok {
				payload[key] = compactText(text, keep)
			}
		}
	case "tool_call":
		input, ok := payload["input"].(map[string]any)
		if !ok {
			return nil, false
		}
		// Per value, so small parameters such as a command or path survive
		// whole beside an excerpted file body.
		for key, value := range input {
			input[key] = compactValue(value, keep)
		}
	default:
		return nil, false
	}
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return nil, false
	}
	compacted := bytes.TrimRight(out.Bytes(), "\n")
	if len(compacted) >= len(raw) {
		return nil, false
	}
	return compacted, true
}

func compactValue(value any, keep int) any {
	switch typed := value.(type) {
	case string:
		return compactText(typed, keep)
	case map[string]any, []any:
		encoded, err := json.Marshal(typed)
		if err != nil || len(encoded) <= compactedLimit(keep) {
			return value
		}
		return compactText(string(encoded), keep)
	default:
		return value
	}
}

// compactedLimit is the longest value left whole. A compacted value is shorter
// than it, so a second pass never compacts an excerpt again.
func compactedLimit(keep int) int { return 2*keep + 256 }

func compactText(text string, keep int) string {
	if len(text) <= compactedLimit(keep) {
		return text
	}
	head := text[:runeFloor(text, keep)]
	tail := text[runeCeil(text, len(text)-keep):]
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%s\n…[%s %d bytes, sha256:%x; kept first %d and last %d bytes]…\n%s",
		head, compactedValueMarker, len(text), sum, len(head), len(tail), tail)
}

func runeFloor(s string, i int) int {
	for i > 0 && i < len(s) && !utf8.RuneStart(s[i]) {
		i--
	}
	return i
}

func runeCeil(s string, i int) int {
	for i < len(s) && !utf8.RuneStart(s[i]) {
		i++
	}
	return i
}

var _ PayloadCompactor = (*SQLiteStore)(nil)
