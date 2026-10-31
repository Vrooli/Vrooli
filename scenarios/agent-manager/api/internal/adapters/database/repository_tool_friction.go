// Responsibility: store and read the tool friction features projected with
// each command fact, so a profile is SQL over stored rows and never reopens
// run events.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageattribution"

	"github.com/jmoiron/sqlx"
)

var _ usageattribution.ToolCallStore = (*UsageAttributionRepository)(nil)

// toolColumns are the fact columns holding toolfriction.Features. They live on
// the fact row rather than a side table: they are 1:1 with the fact, share its
// lifecycle (replaced with the run, deleted with it) and are only read through
// the partial index on tool_profiled.
var toolColumns = []string{"tool_profiled", "tool_command", "tool_key", "tool_help", "tool_filtered", "tool_wrapped", "tool_truncated", "tool_shape_guess", "tool_json_paths", "tool_signature", "tool_error_line", "tool_notice", "tool_notice_line", "tool_bypass_owner", "tool_bypass_method"}

// toolValues is a fact's features in toolColumns order; a fact the profiles do
// not read (nil) stores the column defaults.
func toolValues(t *toolfriction.Features) []any {
	if t == nil {
		return []any{0, "", "", 0, 0, 0, 0, 0, "", "", "", "", "", "", ""}
	}
	return []any{1, t.Command, t.Key, t.Help, t.Filtered, t.Wrapped, t.Truncated, t.ShapeGuess, strings.Join(t.JSONPaths, "\n"), t.Signature, t.ErrorLine, t.Notice, t.NoticeLine, t.BypassOwner, t.BypassMethod}
}

type toolCallRow struct {
	RunID         string        `db:"run_id"`
	CallEventID   string        `db:"call_event_id"`
	ResultEventID string        `db:"result_event_id"`
	OutputEventID string        `db:"output_event_id"`
	Segment       int           `db:"segment_index"`
	OccurredAt    string        `db:"occurred_at"`
	Runner        string        `db:"runner_type"`
	Capability    string        `db:"capability"`
	Executable    string        `db:"executable"`
	CommandPath   string        `db:"command_path"`
	Ownership     string        `db:"ownership"`
	Outcome       string        `db:"outcome"`
	ExitCode      sql.NullInt64 `db:"exit_code"`
	DurationMS    sql.NullInt64 `db:"duration_ms"`
	ResultTokens  int64         `db:"result_tokens"`
	TokenBasis    string        `db:"token_basis"`
	OutputBytes   int64         `db:"output_bytes"`
	Residency     int64         `db:"residency_requests"`
	ResidencySeg  int64         `db:"residency_segment"`
	SkillID       string        `db:"skill_id"`
	SemanticsKind string        `db:"semantics_kind"`
	Command       string        `db:"tool_command"`
	Key           string        `db:"tool_key"`
	Help          bool          `db:"tool_help"`
	Filtered      bool          `db:"tool_filtered"`
	Wrapped       bool          `db:"tool_wrapped"`
	Truncated     bool          `db:"tool_truncated"`
	ShapeGuess    bool          `db:"tool_shape_guess"`
	JSONPaths     string        `db:"tool_json_paths"`
	Signature     string        `db:"tool_signature"`
	ErrorLine     string        `db:"tool_error_line"`
	Notice        string        `db:"tool_notice"`
	NoticeLine    string        `db:"tool_notice_line"`
	BypassOwner   string        `db:"tool_bypass_owner"`
	BypassMethod  string        `db:"tool_bypass_method"`
	// The run's transcript record: NULL when none was kept.
	ParserHarness sql.NullString `db:"parser_harness"`
	ParserVersion sql.NullInt64  `db:"parser_version"`
}

// WithTranscriptParserVersions names the current transcript parser version of
// a harness, so ToolCalls can mark the calls of runs an older parser read. A
// repository without it does not judge parser versions.
func (r *UsageAttributionRepository) WithTranscriptParserVersions(current func(harness string) int) *UsageAttributionRepository {
	r.parserVersion = current
	return r
}

// parser reports whether a run's events were derived by an older transcript
// parser (AM-12a), or the run has no transcript record, so its parser is
// unknown: neither is evidence of complete outcomes, and they are counted
// apart.
func (r *UsageAttributionRepository) parser(row toolCallRow) (older, noTranscript bool) {
	if r.parserVersion == nil {
		return false, false
	}
	if !row.ParserVersion.Valid {
		return false, true
	}
	return int(row.ParserVersion.Int64) < r.parserVersion(row.ParserHarness.String), false
}

// WithToolCallMemo shares ToolCalls loads: windows whose bounds fall in the
// same ttl-long slot are one load, read once and kept for ttl (see
// toolCallMemo). The usage surfaces refetch every minute and report and
// dismiss recompute the list, so without it each read materialises the week.
func (r *UsageAttributionRepository) WithToolCallMemo(ttl time.Duration) *UsageAttributionRepository {
	r.memo = newToolCallMemo(ttl, memoMaxCalls, time.Now)
	return r
}

// ToolCalls reads every command fact in [from, to) projected at the current
// invocation fact version (code excluded; file reads included) with its
// stored features, in run and time order. Each call carries whether its run
// was read by an older transcript parser (see WithTranscriptParserVersions)
// and whether it reaches the internals of the scenario its run works on
// (markOwnWork). Facts of an older version were derived by older rules
// (features missing or meaning something else), so they are not read: the
// window's Stale counts their runs, command calls and calls per command path
// for the coverage line and each finding's base. With a memo the bounds are
// its slot's (at most its ttl earlier) and the window is shared: callers must
// not change it.
func (r *UsageAttributionRepository) ToolCalls(ctx context.Context, from, to time.Time) (toolfriction.Window, error) {
	if r.memo != nil {
		return r.memo.load(ctx, from, to, r.toolWindow)
	}
	return r.toolWindow(ctx, from, to)
}

func (r *UsageAttributionRepository) toolWindow(ctx context.Context, from, to time.Time) (toolfriction.Window, error) {
	start, end := from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano)
	calls, err := r.toolCalls(ctx, toolWindow, start, end)
	if err != nil {
		return toolfriction.Window{}, err
	}
	if err := r.markOwnWork(ctx, calls); err != nil {
		return toolfriction.Window{}, err
	}
	// One pass over the window's stale facts: calls per path and run, summed
	// here into calls per path, the distinct runs and the total.
	var rows []struct {
		Path  string `db:"command_path"`
		RunID string `db:"run_id"`
		Calls int    `db:"calls"`
	}
	if err := r.db.SelectContext(ctx, &rows, staleCallsQuery, runsignal.InvocationFactVersion, runsignal.KindFileRead, start, end); err != nil {
		return toolfriction.Window{}, wrapDBError("tool_calls_stale", "InvocationFact", "", err)
	}
	stale := toolfriction.Stale{Paths: map[string]int{}}
	runs := map[string]bool{}
	for _, row := range rows {
		runs[row.RunID] = true
		stale.Calls += row.Calls
		if row.Path != "" {
			stale.Paths[row.Path] += row.Calls
		}
	}
	stale.Runs = len(runs)
	return toolfriction.Window{Calls: calls, Stale: stale}, nil
}

// MaxOutputChars bounds the text ToolOutputs returns per result: Claude
// Code's inline limit for a command's output (BASH_MAX_OUTPUT_LENGTH, 30,000
// characters); beyond it the agent saw a persisted-output pointer, not the
// text, so an argument found further on was not picked from what it read.
const MaxOutputChars = 30_000

// ToolOutputs reads the text the agent saw for each result event id (the
// result's error, output and the text of its blocks, joined by newlines and
// cut at MaxOutputChars), keyed by id. Only those fields are selected, so the
// raw JSON copies a result also stores are never read. An id with no stored
// event is absent. Callers pass a bounded set: the opportunities dependency
// sample.
func (r *UsageAttributionRepository) ToolOutputs(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	const batch = 200
	for start := 0; start < len(ids); start += batch {
		chunk := ids[start:min(len(ids), start+batch)]
		query, args, err := sqlx.In(toolOutputsQuery, MaxOutputChars, chunk)
		if err != nil {
			return nil, wrapDBError("tool_outputs", "RunEvent", "", err)
		}
		var rows []struct {
			ID   string `db:"id"`
			Text string `db:"text"`
		}
		if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
			return nil, wrapDBError("tool_outputs", "RunEvent", "", err)
		}
		for _, row := range rows {
			out[row.ID] = row.Text
		}
	}
	return out, nil
}

// toolOutputsQuery selects the text fields of tool results (args: the cut,
// the ids); a payload that is not JSON gives no row.
const toolOutputsQuery = `SELECT id, substr(COALESCE(json_extract(d, '$.error'), '') || char(10) || COALESCE(json_extract(d, '$.output'), '')
		|| char(10) || COALESCE((SELECT group_concat(COALESCE(json_extract(b.value, '$.text'), ''), char(10)) FROM json_each(d, '$.outputBlocks') b), ''), 1, ?) AS text
	FROM (SELECT id, CAST(data AS TEXT) AS d FROM run_events WHERE event_type = 'tool_result' AND id IN (?)) WHERE json_valid(d)`

// RunToolFriction scores one run's command calls projected at the current
// fact version (supervision's friction trigger reads it) and says how far its
// events are projected at that version (RunScore.Through, zero when they are
// not); a run with no such calls, or whose outcomes are mostly unknown, is not
// scorable (RunScore.Scorable): it reads as "cannot tell".
func (r *UsageAttributionRepository) RunToolFriction(ctx context.Context, runID string) (toolfriction.RunScore, error) {
	calls, err := r.toolCalls(ctx, "f.run_id = ?", runID)
	if err != nil {
		return toolfriction.RunScore{}, err
	}
	score := toolfriction.ScoreRun(calls)
	var through SQLiteTime
	err = r.db.GetContext(ctx, &through, `SELECT last_event_at FROM invocation_read_model_watermarks WHERE run_id = ? AND classifier_version = ?`, runID, runsignal.InvocationFactVersion)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return toolfriction.RunScore{}, wrapDBError("run_tool_friction_watermark", "InvocationFact", runID, err)
	}
	score.Through = time.Time(through)
	return score, nil
}

// markOwnWork sets Call.OwnWork on the calls that reach the internals of the
// scenario their run works on (toolfriction.WorksOn): for each run and
// scenario it reaches, the run's tool calls that name any scenario's source
// tree (`scenarios/<name>/` in the first ownWorkPrefix characters of the
// call: commands, file paths and working directories) and those that name the
// reached scenario's own source, outside its data directory.
func (r *UsageAttributionRepository) markOwnWork(ctx context.Context, calls []toolfriction.Call) error {
	type pair struct{ run, owner string }
	var pairs []pair
	seen := map[pair]bool{}
	for _, c := range calls {
		if p := (pair{c.RunID, c.BypassOwner}); c.BypassOwner != "" && !seen[p] {
			seen[p] = true
			pairs = append(pairs, p)
		}
	}
	works := map[pair]bool{}
	const batch = 200
	for start := 0; start < len(pairs); start += batch {
		chunk := pairs[start:min(len(pairs), start+batch)]
		values := make([]string, len(chunk))
		args := make([]any, 0, 2*len(chunk))
		for i, p := range chunk {
			values[i] = "(?, ?)"
			args = append(args, p.run, p.owner)
		}
		var rows []struct {
			RunID string `db:"run_id"`
			Owner string `db:"owner"`
			Named int    `db:"named"`
			Own   int    `db:"own"`
		}
		if err := r.db.SelectContext(ctx, &rows, fmt.Sprintf(ownWorkQuery, strings.Join(values, ", ")), args...); err != nil {
			return wrapDBError("tool_calls_own_work", "RunEvent", "", err)
		}
		for _, row := range rows {
			works[pair{row.RunID, row.Owner}] = toolfriction.WorksOn(row.Own, row.Named)
		}
	}
	for i, c := range calls {
		calls[i].OwnWork = c.BypassOwner != "" && works[pair{c.RunID, c.BypassOwner}]
	}
	return nil
}

// ownWorkPrefix bounds how much of a tool call's payload markOwnWork reads:
// the command, path or script head, never a written file's whole body.
const ownWorkPrefix = 2000

// ownWorkQuery counts, per (run, scenario) pair (%s: the VALUES rows), the
// run's tool calls naming any scenario's source tree and those naming the
// scenario's own outside its data directory.
var ownWorkQuery = `WITH p(run_id, owner) AS (VALUES %s)
	SELECT p.run_id, p.owner, COALESCE(SUM(instr(e.h, 'scenarios/') > 0), 0) AS named,
		COALESCE(SUM(instr(e.h, 'scenarios/' || p.owner || '/') > 0 AND instr(e.h, 'scenarios/' || p.owner || '/data') = 0), 0) AS own
	FROM p JOIN (SELECT run_id, substr(CAST(data AS TEXT), 1, ` + fmt.Sprint(ownWorkPrefix) + `) AS h FROM run_events
		WHERE event_type = 'tool_call' AND run_id IN (SELECT run_id FROM p)) e ON e.run_id = p.run_id
	GROUP BY p.run_id, p.owner`

// toolCallMemo shares the loads of tool windows (ToolCalls). A window's key
// is its bounds floored to a ttl-long slot and the fact version, so reads in
// one slot (the friction list, the recommendations and the opportunities a
// page refetches together, a report right after a listing) share one load,
// read with the slot's bounds; concurrent reads of a key wait for the one
// load in flight. The load runs apart from its callers (each stops waiting
// when its own context ends; the load has memoLoadTimeout), and a failed or
// panicking load is not kept. An entry is dropped ttl after it loaded, and the
// entries kept hold at most maxCalls calls together (oldest dropped first;
// memoMaxCalls in production), so the memo holds about one week's window.
// Windows longer than memoMaxSpan (the 30-day recall windows) are not
// memoized: one alone outgrows the bound, so each read loads its own. A new
// slot's load drops the earlier slots of its span, which reads ending now no
// longer reach, so a reload holds one copy of a week beside the one loading.
type toolCallMemo struct {
	ttl      time.Duration
	maxCalls int
	now      func() time.Time
	mu       sync.Mutex
	entries  map[toolCallKey]*toolCallEntry
}

func newToolCallMemo(ttl time.Duration, maxCalls int, now func() time.Time) *toolCallMemo {
	return &toolCallMemo{ttl: ttl, maxCalls: maxCalls, now: now, entries: map[toolCallKey]*toolCallEntry{}}
}

const (
	// memoMaxCalls bounds the calls the memo keeps (a week is ≈150k).
	memoMaxCalls = 200_000
	// memoMaxSpan is the longest window the memo shares: the default week
	// with a day's slack.
	memoMaxSpan = 8 * 24 * time.Hour
	// memoLoadTimeout bounds a shared load, which no caller's context ends.
	memoLoadTimeout = 2 * time.Minute
)

type toolCallKey struct {
	from, to time.Time
	version  string
}

type toolCallEntry struct {
	done   chan struct{}
	window toolfriction.Window
	err    error
	loaded time.Time
}

type toolWindowRead func(context.Context, time.Time, time.Time) (toolfriction.Window, error)

func (m *toolCallMemo) load(ctx context.Context, from, to time.Time, read toolWindowRead) (toolfriction.Window, error) {
	if to.Sub(from) > memoMaxSpan {
		return read(ctx, from, to)
	}
	key := toolCallKey{from: from.UTC().Truncate(m.ttl), to: to.UTC().Truncate(m.ttl), version: runsignal.InvocationFactVersion}
	m.mu.Lock()
	m.evict()
	entry, ok := m.entries[key]
	if !ok {
		for other, held := range m.entries {
			if !held.loaded.IsZero() && other.version == key.version && other.to.Sub(other.from) == key.to.Sub(key.from) {
				delete(m.entries, other)
			}
		}
		entry = &toolCallEntry{done: make(chan struct{})}
		m.entries[key] = entry
		go m.fill(context.WithoutCancel(ctx), key, entry, read)
	}
	m.mu.Unlock()
	select {
	case <-entry.done:
		return entry.window, entry.err
	case <-ctx.Done():
		return toolfriction.Window{}, ctx.Err()
	}
}

// fill runs one shared load and always releases its waiters: a panic becomes
// the load's error, and a failed load is dropped so the next read retries.
func (m *toolCallMemo) fill(ctx context.Context, key toolCallKey, entry *toolCallEntry, read toolWindowRead) {
	ctx, cancel := context.WithTimeout(ctx, memoLoadTimeout)
	defer cancel()
	defer func() {
		if p := recover(); p != nil {
			entry.window, entry.err = toolfriction.Window{}, fmt.Errorf("load tool window: panic: %v", p)
		}
		m.mu.Lock()
		entry.loaded = m.now()
		if entry.err != nil && m.entries[key] == entry {
			delete(m.entries, key)
		}
		m.evict()
		m.mu.Unlock()
		close(entry.done)
	}()
	entry.window, entry.err = read(ctx, key.from, key.to)
}

// evict drops loaded entries older than ttl, then the oldest loaded ones
// until the rest hold at most maxCalls calls. Loads in flight stay.
func (m *toolCallMemo) evict() {
	now, held := m.now(), 0
	for key, entry := range m.entries {
		switch {
		case entry.loaded.IsZero():
		case now.Sub(entry.loaded) >= m.ttl:
			delete(m.entries, key)
		default:
			held += len(entry.window.Calls)
		}
	}
	for held > m.maxCalls {
		var oldest toolCallKey
		var at time.Time
		for key, entry := range m.entries {
			if !entry.loaded.IsZero() && (at.IsZero() || entry.loaded.Before(at)) {
				oldest, at = key, entry.loaded
			}
		}
		held -= len(m.entries[oldest].window.Calls)
		delete(m.entries, oldest)
	}
}

// toolWindow selects the facts of [from, to).
const toolWindow = "f.occurred_at >= ? AND f.occurred_at < ?"

// staleCallsQuery counts a window's facts of other fact versions per command
// path and run in one pass (args: current version, file-read kind, start, end).
const staleCallsQuery = `SELECT COALESCE(f.command_path, '') AS command_path, f.run_id, COUNT(*) AS calls
	FROM invocation_read_model_facts f
	WHERE f.classifier_version <> ? AND f.capability <> 'code' AND (COALESCE(f.command_path, '') <> '' OR f.capability = ?) AND ` + toolWindow + `
	GROUP BY f.command_path, f.run_id`

// toolCallsQuery reads the profiled facts at the current version matching
// where (over facts f) with their run's transcript record, read for the runs
// in scope only rather than aggregated over every run's. Its args are the
// version and where's args, twice.
func toolCallsQuery(where string) string {
	scope := "f.tool_profiled = 1 AND f.classifier_version = ? AND " + where
	return `SELECT f.run_id, f.call_event_id, COALESCE(f.result_event_id, '') AS result_event_id, COALESCE(f.output_event_id, '') AS output_event_id, f.segment_index, f.occurred_at, f.runner_type, f.capability,
			COALESCE(f.executable, '') AS executable, COALESCE(f.command_path, '') AS command_path, f.ownership, f.outcome, f.exit_code, f.duration_ms,
			f.result_tokens, f.token_basis, f.output_bytes, f.residency_requests, f.residency_segment, f.skill_id, f.semantics_kind, f.` + strings.Join(toolColumns[1:], ", f.") + `,
			t.harness AS parser_harness, t.parser_version
		FROM invocation_read_model_facts f
		LEFT JOIN (SELECT run_id, MIN(harness) AS harness, MIN(parser_version) AS parser_version FROM transcript_files
			WHERE run_id IN (SELECT f.run_id FROM invocation_read_model_facts f WHERE ` + scope + `) GROUP BY run_id) t ON t.run_id = f.run_id
		WHERE ` + scope + `
		ORDER BY f.run_id, f.occurred_at, f.call_event_id, f.segment_index`
}

// toolCalls reads the profiled facts matching where with their run's
// transcript record (toolCallsQuery).
func (r *UsageAttributionRepository) toolCalls(ctx context.Context, where string, args ...any) ([]toolfriction.Call, error) {
	scoped := append([]any{runsignal.InvocationFactVersion}, args...)
	var rows []toolCallRow
	if err := r.db.SelectContext(ctx, &rows, toolCallsQuery(where), append(scoped, scoped...)...); err != nil {
		return nil, wrapDBError("tool_calls", "InvocationFact", "", err)
	}
	out := make([]toolfriction.Call, 0, len(rows))
	for _, row := range rows {
		at, _ := parseEventTime(row.OccurredAt)
		fact := toolfriction.Fact{RunID: row.RunID, At: at, Runner: row.Runner}
		fact.CallEventID, fact.ResultEventID, fact.OutputEventID, fact.SegmentIndex = row.CallEventID, row.ResultEventID, row.OutputEventID, row.Segment
		fact.Capability, fact.Executable, fact.CommandPath = row.Capability, row.Executable, row.CommandPath
		fact.Ownership, fact.Outcome, fact.SemanticsKind = row.Ownership, row.Outcome, row.SemanticsKind
		fact.ResultTokens, fact.OutputBytes, fact.ResidencyRequests = row.ResultTokens, row.OutputBytes, row.Residency
		fact.TokenBasis = tokenaccounting.Basis(row.TokenBasis)
		fact.ResidencySegment, fact.SkillID = row.ResidencySeg, row.SkillID
		if row.ExitCode.Valid {
			code := int(row.ExitCode.Int64)
			fact.ExitCode = &code
		}
		if row.DurationMS.Valid {
			ms := row.DurationMS.Int64
			fact.DurationMS = &ms
		}
		features := toolfriction.Features{
			Command: row.Command, Key: row.Key, Help: row.Help, Filtered: row.Filtered, Wrapped: row.Wrapped, Truncated: row.Truncated, ShapeGuess: row.ShapeGuess,
			Signature: row.Signature, ErrorLine: row.ErrorLine, Notice: row.Notice, NoticeLine: row.NoticeLine, BypassOwner: row.BypassOwner, BypassMethod: row.BypassMethod,
		}
		if row.JSONPaths != "" {
			features.JSONPaths = strings.Split(row.JSONPaths, "\n")
		}
		call := toolfriction.Observe(fact, features)
		call.OlderParser, call.NoTranscript = r.parser(row)
		out = append(out, call)
	}
	return out, nil
}
