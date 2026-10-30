package interventions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"
	"agent-manager/internal/transcriptredact"
)

// Store persists classifications in operator_interventions and reads the
// candidate operator messages from run_events.
type Store struct{ db sqlcompat.DB }

func NewStore(db sqlcompat.DB) *Store { return &Store{db: db} }

// Cursor is a position in the (timestamp, run_id, sequence) order of message
// events: the order of the partial index idx_run_events_conversation_source.
type Cursor struct {
	Timestamp string
	RunID     string
	Sequence  int64
}

// Candidate is an operator message event without a current classification, or
// whose unclassified attempt is due again.
type Candidate struct {
	EventID    string
	RunID      string
	Sequence   int64
	OccurredAt string
	Text       string
	Attempts   int
}

// Candidates scans at most limit user-role message events after from and
// before upper, returning the operator messages among them that need a
// classification for version (not recorded, or unclassified with the attempt
// due), the cursor after the last scanned row, or upper itself once the range
// [from, upper) has no more rows, and how many rows it scanned. Recorded
// messages count as scanned, so a re-walk of the window spends the pass cap
// like new work, but their payload is never read: only operator messages are
// recorded, so their role needs no check.
func (s *Store) Candidates(ctx context.Context, from Cursor, upper, version string, now time.Time, limit int) (candidates []Candidate, next Cursor, scanned int, err error) {
	// The redundant event_type IN (...) makes SQLite use the partial index
	// idx_run_events_conversation_source, which is in exactly this order.
	rows, err := s.db.QueryxContext(ctx, `
SELECT e.id, e.run_id, e.sequence, e.timestamp, COALESCE(i.attempts, 0),
  CASE WHEN i.status IS NULL OR (i.status = ? AND COALESCE(i.next_attempt_at, '') <= ?) THEN e.data END AS data
FROM run_events e
LEFT JOIN operator_interventions i ON i.event_id = e.id AND i.classifier_version = ?
WHERE e.event_type IN ('message', 'tool_call', 'tool_result') AND e.event_type = 'message'
  AND (e.timestamp, e.run_id, e.sequence) > (?, ?, ?) AND e.timestamp < ?
  AND CASE WHEN i.status IS NULL THEN json_extract(e.data, '$.role') IN ('user', 'operator') ELSE 1 END
ORDER BY e.timestamp, e.run_id, e.sequence
LIMIT ?`, string(StatusUnclassified), formatTime(now), version, from.Timestamp, from.RunID, from.Sequence, upper, limit)
	if err != nil {
		return nil, from, 0, fmt.Errorf("scan operator messages: %w", err)
	}
	defer rows.Close()
	next = from
	for rows.Next() {
		var (
			c    Candidate
			data sql.NullString
		)
		if err := rows.Scan(&c.EventID, &c.RunID, &c.Sequence, &c.OccurredAt, &c.Attempts, &data); err != nil {
			return nil, from, 0, fmt.Errorf("scan operator message: %w", err)
		}
		scanned++
		next = Cursor{Timestamp: c.OccurredAt, RunID: c.RunID, Sequence: c.Sequence}
		if !data.Valid {
			continue // recorded, or an unclassified attempt not yet due
		}
		payload, err := domain.DecodeEventPayload(domain.EventTypeMessage, []byte(data.String))
		if err != nil {
			continue
		}
		message, ok := payload.(*domain.MessageEventData)
		if !ok || domain.MessageSource(&domain.RunEvent{EventType: domain.EventTypeMessage, Data: message}) != domain.MessageSourceOperator {
			continue
		}
		c.Text = message.Content
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, from, 0, fmt.Errorf("scan operator messages: %w", err)
	}
	if scanned < limit {
		next = Cursor{Timestamp: upper}
	}
	return candidates, next, scanned, nil
}

// Work is the agent's work between the previous operator message in the run
// and this one: provider requests, tool calls and tokens (all components).
type Work struct {
	Requests  int64
	ToolCalls int64
	Tokens    int64
}

func (w *Work) add(other Work) {
	w.Requests += other.Requests
	w.ToolCalls += other.ToolCalls
	w.Tokens += other.Tokens
}

// pooledHint reports whether a classification that no proven category admits
// is admitted by the pooled friction bucket, and the friction category to show
// as its unproven hint (empty when the answer names no friction category).
func pooledHint(gate Gate, model string, category Category, distribution string) (Category, bool) {
	var probabilities map[Category]float64
	_ = json.Unmarshal([]byte(distribution), &probabilities)
	if !gate.ShownPooled(model, Answer{Probabilities: probabilities}.FrictionMass()) {
		return "", false
	}
	best, bestP := category, -1.0
	for _, c := range Categories() {
		if p, ok := probabilities[c]; ok && c.Friction() && p > bestP {
			best, bestP = c, p
		}
	}
	if !best.Friction() {
		return "", true
	}
	return best, true
}

// Record is one stored classification.
type Record struct {
	Candidate
	Version       string
	MessageKey    string
	Answer        Answer
	NextAttemptAt time.Time
	Work          Work
	RepeatOf      string
}

// Save upserts records.
func (s *Store) Save(ctx context.Context, records []Record, now time.Time) error {
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
		attempts := r.Attempts
		next := ""
		if r.Answer.Status == StatusUnclassified {
			attempts++
			next = formatTime(r.NextAttemptAt)
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO operator_interventions (event_id, classifier_version, run_id, sequence, occurred_at, message_key, status, category, probability,
  probabilities_json, method, provider, model, resource_role, unclassified_reason, attempts, next_attempt_at, work_requests, work_tool_calls,
  work_tokens, repeat_of_event_id, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(event_id, classifier_version) DO UPDATE SET status = excluded.status, category = excluded.category,
  probability = excluded.probability, probabilities_json = excluded.probabilities_json, method = excluded.method,
  provider = excluded.provider, model = excluded.model, resource_role = excluded.resource_role,
  unclassified_reason = excluded.unclassified_reason, attempts = excluded.attempts, next_attempt_at = excluded.next_attempt_at,
  work_requests = excluded.work_requests, work_tool_calls = excluded.work_tool_calls, work_tokens = excluded.work_tokens,
  repeat_of_event_id = excluded.repeat_of_event_id, recorded_at = excluded.recorded_at`,
			r.EventID, r.Version, r.RunID, r.Sequence, r.OccurredAt, r.MessageKey, string(r.Answer.Status), string(r.Answer.Category), probability,
			probabilities, r.Answer.Method, r.Answer.Provider, r.Answer.Model, r.Answer.ResourceRole, r.Answer.Reason, attempts, next,
			r.Work.Requests, r.Work.ToolCalls, r.Work.Tokens, r.RepeatOf, formatTime(now)); err != nil {
			return fmt.Errorf("save intervention %s: %w", r.EventID, err)
		}
	}
	return nil
}

// RecordedKeys maps each message key already recorded for version by another
// event (not as a duplicate) to that event.
func (s *Store) RecordedKeys(ctx context.Context, version string, keys []string) (map[string]string, error) {
	out := map[string]string{}
	if len(keys) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, version)
	for _, key := range keys {
		args = append(args, key)
	}
	rows, err := s.db.QueryxContext(ctx, `SELECT message_key, event_id FROM operator_interventions
WHERE classifier_version = ? AND status <> 'duplicate' AND message_key IN (`+placeholders(len(keys))+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("read message keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, id string
		if err := rows.Scan(&key, &id); err != nil {
			return nil, err
		}
		out[key] = id
	}
	return out, rows.Err()
}

// Categories returns the classified category of each listed event.
func (s *Store) Categories(ctx context.Context, version string, eventIDs []string) (map[string]Category, error) {
	out := map[string]Category{}
	if len(eventIDs) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(eventIDs)+1)
	args = append(args, version)
	for _, id := range eventIDs {
		args = append(args, id)
	}
	rows, err := s.db.QueryxContext(ctx, `SELECT event_id, category FROM operator_interventions
WHERE classifier_version = ? AND status = 'classified' AND event_id IN (`+placeholders(len(eventIDs))+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("read categories: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, category string
		if err := rows.Scan(&id, &category); err != nil {
			return nil, err
		}
		out[id] = Category(category)
	}
	return out, rows.Err()
}

// PruneVersions deletes rows of every classifier version but keep: they were
// produced by questions no longer asked and no gate applies to them.
func (s *Store) PruneVersions(ctx context.Context, keep string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM operator_interventions WHERE classifier_version <> ?`, keep)
	if err != nil {
		return 0, fmt.Errorf("prune intervention versions: %w", err)
	}
	return result.RowsAffected()
}

// CategorySummary is one category over a window.
type CategorySummary struct {
	Category Category
	// Classified counts every message the model put in the category; Shown
	// counts those the gate admits. Only Shown may be reported as friction.
	Classified       int64
	Shown            int64
	Repeats          int64 // shown quality_miss reports that repeat an earlier one
	MedianWorkTokens int64 // over shown messages: agent work before the correction
	Work             Work  // summed over shown messages
}

// CategoryHint counts the pooled messages whose most likely friction category
// is Category. It is a hint, not a proven count.
type CategoryHint struct {
	Category Category
	Count    int64
}

// PooledSummary is the messages the gate admits as "operator friction
// (category unproven)": friction by probability mass, in no proven category.
type PooledSummary struct {
	Shown   int64
	Repeats int64
	Work    Work
	Hints   []CategoryHint // taxonomy order, only categories with messages
}

// Summary is the classified operator messages of a window.
type Summary struct {
	Categories   []CategorySummary // taxonomy order
	Pooled       PooledSummary
	Classified   int64
	Unclassified int64
	AgentHours   float64
}

// Coverage is the classified share of operator interventions, or nil when the
// window has none recorded.
func (s Summary) Coverage() *float64 {
	total := s.Classified + s.Unclassified
	if total == 0 {
		return nil
	}
	value := float64(s.Classified) / float64(total)
	return &value
}

// Summarize reads the window [from, to) for version under gate. Agent-hours
// are ten-minute (run, bucket) slots with at least one provider request in
// transcript_usage, divided by six.
func (s *Store) Summarize(ctx context.Context, version string, gate Gate, from, to time.Time) (Summary, error) {
	rows, err := s.db.QueryxContext(ctx, `SELECT status, category, probability, probabilities_json, model, work_requests, work_tool_calls, work_tokens, repeat_of_event_id
FROM operator_interventions WHERE classifier_version = ? AND occurred_at >= ? AND occurred_at < ? AND status IN ('classified', 'unclassified')`,
		version, formatTime(from), formatTime(to))
	if err != nil {
		return Summary{}, fmt.Errorf("summarize interventions: %w", err)
	}
	defer rows.Close()
	summary := Summary{}
	byCategory := map[Category]*CategorySummary{}
	work := map[Category][]int64{}
	hints := map[Category]int64{}
	for _, category := range Categories() {
		byCategory[category] = &CategorySummary{Category: category}
	}
	for rows.Next() {
		var (
			status, category, model, repeatOf, distribution string
			probability                                     sql.NullFloat64
			w                                               Work
		)
		if err := rows.Scan(&status, &category, &probability, &distribution, &model, &w.Requests, &w.ToolCalls, &w.Tokens, &repeatOf); err != nil {
			return Summary{}, err
		}
		if Status(status) == StatusUnclassified {
			summary.Unclassified++
			continue
		}
		summary.Classified++
		entry, ok := byCategory[Category(category)]
		if !ok {
			continue
		}
		entry.Classified++
		var p *float64
		if probability.Valid {
			p = &probability.Float64
		}
		if !gate.Shown(model, entry.Category, p) {
			if hint, ok := pooledHint(gate, model, Category(category), distribution); ok {
				summary.Pooled.Shown++
				summary.Pooled.Work.add(w)
				if repeatOf != "" {
					summary.Pooled.Repeats++
				}
				hints[hint]++
			}
			continue
		}
		entry.Shown++
		work[entry.Category] = append(work[entry.Category], w.Tokens)
		entry.Work.add(w)
		if repeatOf != "" {
			entry.Repeats++
		}
	}
	if err := rows.Err(); err != nil {
		return Summary{}, err
	}
	for _, category := range Categories() {
		entry := byCategory[category]
		entry.MedianWorkTokens = median(work[category])
		summary.Categories = append(summary.Categories, *entry)
		if hints[category] > 0 {
			summary.Pooled.Hints = append(summary.Pooled.Hints, CategoryHint{Category: category, Count: hints[category]})
		}
	}
	var slots int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT run_id || '|' || substr(observed_at, 1, 15)) FROM transcript_usage
WHERE observed_at >= ? AND observed_at < ?`, formatTime(from), formatTime(to)).Scan(&slots); err != nil {
		return Summary{}, fmt.Errorf("count agent hours: %w", err)
	}
	summary.AgentHours = float64(slots) / 6
	return summary, nil
}

// Row is one stored classification the gate admits, with its message.
type Row struct {
	EventID, RunID string
	Sequence       int64
	OccurredAt     string
	Category       Category
	// Pooled marks a row admitted only as "operator friction (category
	// unproven)": Category is then the most likely friction category, a hint.
	Pooled      bool
	Probability *float64
	Model       string
	Work        Work
	RepeatOf    string
	// Text is the operator message, redacted and shortened for display.
	Text string
}

// exampleLength is how much of a message Row.Text keeps, in characters.
const exampleLength = 240

// Rows lists the classifications of [from, to) for version that gate admits,
// in a proven category or pooled, newest first, at most limit (0 means all); runID, when set, limits them to
// one run.
func (s *Store) Rows(ctx context.Context, version string, gate Gate, from, to time.Time, runID string, limit int) ([]Row, error) {
	query := `SELECT i.event_id, i.run_id, i.sequence, i.occurred_at, i.category, i.probability, i.model, i.work_requests, i.work_tool_calls,
  i.work_tokens, i.repeat_of_event_id, i.probabilities_json, e.data
FROM operator_interventions i LEFT JOIN run_events e ON e.id = i.event_id
WHERE i.classifier_version = ? AND i.status = 'classified' AND i.occurred_at >= ? AND i.occurred_at < ?`
	args := []any{version, formatTime(from), formatTime(to)}
	if runID != "" {
		query += ` AND i.run_id = ?`
		args = append(args, runID)
	}
	rows, err := s.db.QueryxContext(ctx, query+` ORDER BY i.occurred_at DESC, i.sequence DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("read interventions: %w", err)
	}
	defer rows.Close()
	var out []Row
	for rows.Next() && (limit <= 0 || len(out) < limit) {
		var (
			r            Row
			category     string
			probability  sql.NullFloat64
			distribution string
			data         sql.NullString
		)
		if err := rows.Scan(&r.EventID, &r.RunID, &r.Sequence, &r.OccurredAt, &category, &probability, &r.Model, &r.Work.Requests, &r.Work.ToolCalls, &r.Work.Tokens, &r.RepeatOf, &distribution, &data); err != nil {
			return nil, err
		}
		r.Category = Category(category)
		if probability.Valid {
			r.Probability = &probability.Float64
		}
		if !gate.Shown(r.Model, r.Category, r.Probability) {
			hint, ok := pooledHint(gate, r.Model, r.Category, distribution)
			if !ok {
				continue
			}
			r.Category, r.Pooled = hint, true
		}
		r.Text = exampleText(data.String)
		out = append(out, r)
	}
	return out, rows.Err()
}

// exampleText is the operator message of a message event's data, redacted and
// cut to exampleLength characters.
func exampleText(data string) string {
	payload, err := domain.DecodeEventPayload(domain.EventTypeMessage, []byte(data))
	if err != nil {
		return ""
	}
	message, ok := payload.(*domain.MessageEventData)
	if !ok {
		return ""
	}
	text := strings.Join(strings.Fields(transcriptredact.Redact(message.Content)), " ")
	if runes := []rune(text); len(runes) > exampleLength {
		text = string(runes[:exampleLength]) + "…"
	}
	return text
}

func median(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values[len(values)/2]
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// formatTime matches the RFC3339Nano UTC text run_events stores.
func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
