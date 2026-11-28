package runannotations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"
	"agent-manager/internal/toolfacts"
)

// Bounds for the repository adapter. They keep one run's read cheap; a text
// selector only ever sees these bounded facts, never a full transcript.
const (
	maxOperatorMessages = 12
	maxAgentMessages    = 8
	// maxOperatorHistory bounds the operator-source messages kept for provenance
	// and message-scope labels. The digest still reads only the newest
	// maxOperatorMessages of them.
	maxOperatorHistory = 48
	// maxOpeningChars bounds the opening of a run kept to tell who launched it.
	maxOpeningChars = 240
)

// SQLiteFeatureSource reads one run's bounded facts from the Agent Manager
// database. It does not compute step-back triggers: the existing stepback
// projection is the owner of those, and the orchestrator can enrich
// DeterministicFlags without changing any label.
type SQLiteFeatureSource struct {
	DB  sqlcompat.DB
	Now func() time.Time
	// Parsers resolves a harness's transcript parser, so the tool facts of a run
	// that has only a transcript file (an imported session whose events were
	// never retained) are still read. Nil leaves such runs with source none.
	Parsers toolfacts.ParserResolver

	toolsOnce sync.Once
	tools     *toolfacts.Reader
	toolsErr  error
}

// NewSQLiteFeatureSource returns a repository-backed feature source. parsers
// may be nil (see SQLiteFeatureSource.Parsers).
func NewSQLiteFeatureSource(db sqlcompat.DB, parsers toolfacts.ParserResolver) *SQLiteFeatureSource {
	return &SQLiteFeatureSource{DB: db, Now: time.Now, Parsers: parsers}
}

// toolReader is the one reader of a run's tool activity, built once from the
// embedded rules the rule environment is declared from.
func (s *SQLiteFeatureSource) toolReader() (*toolfacts.Reader, error) {
	s.toolsOnce.Do(func() {
		if toolRulesErr != nil {
			s.toolsErr = fmt.Errorf("load tool fact rules: %w", toolRulesErr)
			return
		}
		s.tools = &toolfacts.Reader{DB: s.DB, Parsers: s.Parsers, Rules: toolRules}
	})
	return s.tools, s.toolsErr
}

// Facts implements FeatureSource.
func (s *SQLiteFeatureSource) Facts(ctx context.Context, scopeKind Scope, scopeID string) (RunFacts, error) {
	if scopeKind != ScopeRun {
		return RunFacts{}, ErrScopeNotFound
	}
	if s.DB == nil {
		return RunFacts{}, fmt.Errorf("feature source has no database")
	}
	facts, started, ended, err := s.runFacts(ctx, scopeID)
	if err != nil {
		return RunFacts{}, err
	}
	if err := s.messageFacts(ctx, scopeID, &facts); err != nil {
		return RunFacts{}, err
	}
	if err := s.toolFacts(ctx, scopeID, &facts); err != nil {
		return RunFacts{}, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	if !started.IsZero() {
		end := now()
		if !ended.IsZero() {
			end = ended
		}
		facts.Timings.DurationMinutes = roundMinutes(end.Sub(started))
	}
	if first := s.firstToolCall(ctx, scopeID); !first.IsZero() && !started.IsZero() && !first.Before(started) {
		minutes := roundMinutes(first.Sub(started))
		facts.Timings.TimeToFirstToolMin = &minutes
	}
	return facts, nil
}

func (s *SQLiteFeatureSource) runFacts(ctx context.Context, runID string) (RunFacts, time.Time, time.Time, error) {
	var row struct {
		ID            string         `db:"id"`
		WorkloadKind  string         `db:"workload_kind"`
		WorkloadKey   sql.NullString `db:"workload_key"`
		HarnessKind   sql.NullString `db:"harness_kind"`
		RunMode       sql.NullString `db:"run_mode"`
		Status        sql.NullString `db:"status"`
		TerminalClass sql.NullString `db:"terminal_class"`
		StopReason    sql.NullString `db:"stop_reason"`
		StartedAt     sql.NullString `db:"started_at"`
		EndedAt       sql.NullString `db:"ended_at"`
		ExitCode      sql.NullInt64  `db:"exit_code"`
		ParentRunID   sql.NullString `db:"parent_run_id"`
		ImportedFrom  sql.NullString `db:"import_source_harness"`
		Depth         sql.NullInt64  `db:"depth"`
	}
	err := s.DB.GetContext(ctx, &row, `
SELECT id, workload_kind, workload_key, harness_kind, run_mode, status, terminal_class, stop_reason,
       started_at, ended_at, exit_code, parent_run_id, import_source_harness, depth
FROM runs WHERE id = ?`, runID)
	if err != nil {
		if err == sql.ErrNoRows {
			return RunFacts{}, time.Time{}, time.Time{}, ErrScopeNotFound
		}
		return RunFacts{}, time.Time{}, time.Time{}, fmt.Errorf("read run %s: %w", runID, err)
	}
	facts := RunFacts{
		RunID:       row.ID,
		Workload:    WorkloadFacts{Kind: row.WorkloadKind, Key: row.WorkloadKey.String},
		HarnessKind: row.HarnessKind.String,
		RunMode:     row.RunMode.String,
		Terminal: TerminalFacts{
			Status: row.Status.String, Class: row.TerminalClass.String, StopReason: row.StopReason.String,
		},
		Lineage: LineageFacts{
			Imported: strings.TrimSpace(row.ImportedFrom.String) != "", ParentRunID: strings.TrimSpace(row.ParentRunID.String),
			Depth: int(row.Depth.Int64),
		},
		ExistingSignals: map[string]SignalFact{},
	}
	if row.ExitCode.Valid {
		exitCode := int(row.ExitCode.Int64)
		facts.Terminal.ExitCode = &exitCode
	}
	started := parseDBTime(row.StartedAt.String)
	ended := parseDBTime(row.EndedAt.String)
	var compaction int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_events WHERE run_id = ? AND event_type = ?`, runID, string(domain.EventTypeCompaction)).Scan(&compaction); err != nil {
		return RunFacts{}, time.Time{}, time.Time{}, fmt.Errorf("count compaction events: %w", err)
	}
	facts.DeterministicFlags.CompactionCount = compaction
	return facts, started, ended, nil
}

// maxMessageScan bounds how far back the reader pages while looking for
// operator text. Reading a fixed tail of recent messages is wrong: a run that
// ends with a long stretch of assistant messages reports
// operator_message_count == 0 and every operator label is skipped even though
// the operator wrote throughout. Paging until the bounded tails are full keeps
// the read cheap without losing the operator's own words.
const maxMessageScan = 4000

func (s *SQLiteFeatureSource) messageFacts(ctx context.Context, runID string, facts *RunFacts) error {
	var operators, agents []MessageFact
	const pageSize = 200
	// exhausted is true when the scan reached the run's first event, so the
	// oldest operator message read is the one that opened the run.
	exhausted := false
	for scanned := 0; scanned < maxMessageScan; scanned += pageSize {
		if len(operators) >= maxOperatorHistory && len(agents) >= maxAgentMessages {
			break
		}
		rows, err := s.DB.QueryxContext(ctx, `
SELECT id, timestamp, data FROM run_events
WHERE run_id = ? AND event_type = ? ORDER BY sequence DESC LIMIT ? OFFSET ?`,
			runID, string(domain.EventTypeMessage), pageSize, scanned)
		if err != nil {
			return fmt.Errorf("read message events: %w", err)
		}
		count := 0
		for rows.Next() {
			count++
			var id, timestamp, data string
			if err := rows.Scan(&id, &timestamp, &data); err != nil {
				rows.Close()
				return err
			}
			payload, err := domain.DecodeEventPayload(domain.EventTypeMessage, []byte(data))
			if err != nil {
				continue
			}
			message, ok := payload.(*domain.MessageEventData)
			if !ok {
				continue
			}
			fact := MessageFact{EventID: id, At: parseDBTime(timestamp), Text: BoundText(message.Content, 2000), Sender: message.Sender}
			if fact.Text == "" {
				continue
			}
			source := domain.MessageSource(&domain.RunEvent{EventType: domain.EventTypeMessage, Data: message})
			// A stored source is trusted, except for text that is nothing but
			// harness blocks: no person types those, and older rows recorded
			// "operator" before the block was recognised (for example the
			// <skills> block the harness inlines into a workflow's first turn).
			if source == domain.MessageSourceOperator && domain.IsHarnessOnlyText(message.Content) {
				source = domain.MessageSourceInjectedContext
			}
			if source == domain.MessageSourceOperator {
				operators = append(operators, fact)
			} else if message.Role == "assistant" {
				agents = append(agents, fact)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if count < pageSize {
			exhausted = true
			break
		}
	}
	// The pages arrive newest-first, so reversing restores chronological
	// order. Sequence, not the parsed timestamp, is authoritative: a run can
	// contain events with equal or missing timestamps and the tail must still
	// be the newest messages.
	reverseMessages(operators)
	reverseMessages(agents)
	if exhausted && len(operators) > 0 {
		operators[0].Seed = true
		facts.Lineage.Opening = BoundText(operators[0].Text, maxOpeningChars)
	}
	operators = tailMessages(operators, maxOperatorHistory)
	facts.OperatorHistory = operators
	facts.OperatorMessages = tailMessages(operators, maxOperatorMessages)
	facts.AgentText = tailMessages(agents, maxAgentMessages)
	return nil
}

func (s *SQLiteFeatureSource) toolFacts(ctx context.Context, runID string, facts *RunFacts) error {
	reader, err := s.toolReader()
	if err != nil {
		return err
	}
	found, err := reader.Facts(ctx, runID)
	if err != nil {
		return fmt.Errorf("read tool facts: %w", err)
	}
	facts.ToolSummary = found
	return nil
}

func (s *SQLiteFeatureSource) firstToolCall(ctx context.Context, runID string) time.Time {
	var timestamp sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT MIN(timestamp) FROM run_events WHERE run_id = ? AND event_type = ?`,
		runID, string(domain.EventTypeToolCall)).Scan(&timestamp)
	if err != nil || !timestamp.Valid {
		return time.Time{}
	}
	return parseDBTime(timestamp.String)
}

// SQLiteSignalReader reads the current operator-signal version for a run.
type SQLiteSignalReader struct{ DB sqlcompat.DB }

// NewSQLiteSignalReader returns a repository-backed signal reader.
func NewSQLiteSignalReader(db sqlcompat.DB) *SQLiteSignalReader { return &SQLiteSignalReader{DB: db} }

// RunSignals implements SignalReader.
func (s *SQLiteSignalReader) RunSignals(ctx context.Context, runID string) (map[string]SignalFact, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("signal reader has no database")
	}
	var version sql.NullString
	err := s.DB.QueryRowContext(ctx, `
SELECT signal_version FROM operator_signals
WHERE run_id = ? AND status = 'classified' AND label <> ''
ORDER BY recorded_at DESC LIMIT 1`, runID).Scan(&version)
	if err == sql.ErrNoRows {
		return map[string]SignalFact{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read signal version: %w", err)
	}
	rows, err := s.DB.QueryxContext(ctx, `
SELECT label, MAX(COALESCE(probability, 0)), COUNT(*),
  (SELECT event_id FROM operator_signals latest
   WHERE latest.run_id = signals.run_id AND latest.signal_version = signals.signal_version
     AND latest.label = signals.label AND latest.status = 'classified'
   ORDER BY latest.recorded_at DESC LIMIT 1)
FROM operator_signals signals
WHERE run_id = ? AND signal_version = ? AND status = 'classified' AND label <> ''
GROUP BY label`, runID, version.String)
	if err != nil {
		return nil, fmt.Errorf("read run signals: %w", err)
	}
	defer rows.Close()
	out := map[string]SignalFact{}
	for rows.Next() {
		var (
			fact        SignalFact
			probability sql.NullFloat64
			count       int
			latest      sql.NullString
		)
		if err := rows.Scan(&fact.Label, &probability, &count, &latest); err != nil {
			return nil, err
		}
		if probability.Valid {
			value := probability.Float64
			fact.Probability = &value
		}
		fact.Count = count
		fact.LatestEventID = latest.String
		fact.Version = version.String
		out[fact.Label] = fact
	}
	return out, rows.Err()
}

// SQLiteRunSelector selects recent runs for a bounded batch.
type SQLiteRunSelector struct{ DB sqlcompat.DB }

// NewSQLiteRunSelector returns a repository-backed selector.
func NewSQLiteRunSelector(db sqlcompat.DB) *SQLiteRunSelector { return &SQLiteRunSelector{DB: db} }

// CandidateRunsWindow implements WindowSelector.
func (s *SQLiteRunSelector) CandidateRunsWindow(ctx context.Context, since, until time.Time, limit int) ([]string, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("run selector has no database")
	}
	if limit <= 0 {
		limit = DefaultRunsPerPass
	}
	query := `SELECT r.id FROM runs r`
	args := []any{}
	var conditions []string
	if !since.IsZero() {
		conditions = append(conditions, `datetime(started_at) >= datetime(?)`)
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		conditions = append(conditions, `datetime(started_at) <= datetime(?)`)
		args = append(args, formatTime(until))
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	query += ` ORDER BY ` + evidenceFirstOrder + `, datetime(r.started_at) DESC, r.id ASC LIMIT ?`
	args = append(args, limit)
	return s.scanIDs(ctx, query, args...)
}

// CandidateRunsWindowFor selects recent runs in the window, preferring runs
// this pack has not answered yet and then the least recently answered ones.
//
// Without the rotation a bounded pass would re-select the same newest rows on
// every interval: `Evaluate` reuses their cached answers, so the pass is
// wasted and older runs inside the horizon are never annotated. Ordering by
// the newest observation for the current definition hashes makes each pass
// advance the frontier while still filling a fresh database newest-first.
func (s *SQLiteRunSelector) CandidateRunsWindowFor(ctx context.Context, since, until time.Time, limit int, definitionHashes []string) ([]string, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("run selector has no database")
	}
	if limit <= 0 {
		limit = DefaultRunsPerPass
	}
	hashes := make([]string, 0, len(definitionHashes))
	seen := map[string]bool{}
	for _, hash := range definitionHashes {
		hash = strings.TrimSpace(hash)
		if hash == "" || seen[hash] {
			continue
		}
		seen[hash] = true
		hashes = append(hashes, hash)
	}
	// No hashes means the caller cannot describe the pack; the plain recent
	// order is the honest fallback rather than a query that never rotates.
	if len(hashes) == 0 {
		return s.CandidateRunsWindow(ctx, since, until, limit)
	}

	query := `SELECT r.id FROM runs r
LEFT JOIN (
  SELECT scope_id, MAX(observed_at) AS last_annotated
  FROM run_annotations
  WHERE scope_kind = ? AND state IN ('observed', 'compatible', 'not_observed')
    AND definition_hash IN (` + placeholders(len(hashes)) + `)
  GROUP BY scope_id
) a ON a.scope_id = r.id`
	args := []any{string(ScopeRun)}
	for _, hash := range hashes {
		args = append(args, hash)
	}
	var conditions []string
	if !since.IsZero() {
		conditions = append(conditions, `datetime(r.started_at) >= datetime(?)`)
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		conditions = append(conditions, `datetime(r.started_at) <= datetime(?)`)
		args = append(args, formatTime(until))
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	// NULL (never annotated by this pack) sorts before any timestamp in an
	// ascending order; the newest start breaks ties so a fresh database fills
	// newest-first.
	query += ` ORDER BY a.last_annotated ASC, ` + evidenceFirstOrder + `, datetime(r.started_at) DESC, r.id ASC LIMIT ?`
	args = append(args, limit)
	return s.scanIDs(ctx, query, args...)
}

// CandidateRuns implements RunSelector.
func (s *SQLiteRunSelector) CandidateRuns(ctx context.Context, since time.Time, limit int) ([]string, error) {
	return s.CandidateRunsWindow(ctx, since, time.Time{}, limit)
}

// evidenceFirstOrder ranks runs that carry message or tool events ahead of
// runs that recorded only metrics, quotas or logs. The annotation corpus is
// dominated by synthetic runs with no conversational evidence; without this
// preference a newest-first pass walks an endless supply of them and never
// reaches the interactive runs that every decision label needs. The predicate
// is an indexed lookup on (run_id, event_type).
const evidenceFirstOrder = `CASE WHEN EXISTS (
  SELECT 1 FROM run_events e WHERE e.run_id = r.id AND e.event_type IN ('message', 'tool_call')
) THEN 0 ELSE 1 END`

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func (s *SQLiteRunSelector) scanIDs(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := s.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select runs: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func reverseMessages(messages []MessageFact) {
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
}

func tailMessages(messages []MessageFact, count int) []MessageFact {
	if len(messages) <= count {
		return messages
	}
	return messages[len(messages)-count:]
}

func roundMinutes(duration time.Duration) float64 {
	minutes := duration.Minutes()
	if minutes < 0 {
		return 0
	}
	return minutes
}

func parseDBTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		if at, err := time.Parse(layout, value); err == nil {
			return at.UTC()
		}
	}
	return time.Time{}
}
