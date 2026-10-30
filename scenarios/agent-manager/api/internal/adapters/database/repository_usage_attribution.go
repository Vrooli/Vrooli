// Responsibility: read the durable usage, run, episode and outcome facts that
// usage attribution and recommendations use, and repair legacy usage rows.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageattribution"

	"github.com/google/uuid"
)

// UsageAttributionRepository implements usageattribution.Store.
type UsageAttributionRepository struct {
	db            *DB
	parserVersion func(harness string) int
	memo          *toolCallMemo
}

// NewUsageAttributionRepository builds the attribution store.
func NewUsageAttributionRepository(db *DB) *UsageAttributionRepository {
	return &UsageAttributionRepository{db: db}
}

var _ usageattribution.Store = (*UsageAttributionRepository)(nil)

// usageBucket folds one session's Codex reports into a short bucket: usage
// is attributed to quota slices that are minutes to hours long, and a bucket
// of cumulative counters keeps runsignal.AccountUsage's result exact.
const usageBucketPrefix = 15 // "2006-01-02T15:0" — ten minutes

type usageAggregateRow struct {
	FileKey        string         `db:"file_key"`
	RunID          string         `db:"run_id"`
	MessageID      string         `db:"message_id"`
	Cumulative     bool           `db:"cumulative"`
	At             string         `db:"at"`
	Reports        int            `db:"reports"`
	Model          sql.NullString `db:"model"`
	Authority      bool           `db:"authority"`
	Input          int            `db:"input"`
	Output         int            `db:"output"`
	CacheRead      int            `db:"cache_read"`
	CacheCreation  int            `db:"cache_creation"`
	BeforeInput    int            `db:"before_input"`
	BeforeOutput   int            `db:"before_output"`
	BeforeRead     int            `db:"before_read"`
	BeforeCreation int            `db:"before_creation"`
	ImagesIn       int            `db:"images_in"`
	ImagesInUser   int            `db:"images_in_user"`
	ImagesInHigh   int            `db:"images_in_high_detail"`
	ImageBytesIn   int64          `db:"image_bytes_in"`
	ImagesOut      int            `db:"images_out"`
	WebSearches    int            `db:"web_searches"`
	WebFetches     int            `db:"web_fetches"`
	AudioIn        int            `db:"audio_in"`
	ToolCalls      int            `db:"tool_calls"`
	ToolKinds      sql.NullString `db:"tool_kinds"`
	Credits        string         `db:"credits"`
	Harness        string         `db:"harness"`
}

const usageAggregateFacts = `SUM(u.images_in) AS images_in, SUM(u.images_in_user) AS images_in_user,
	SUM(u.images_in_high_detail) AS images_in_high_detail, SUM(u.image_bytes_in) AS image_bytes_in, SUM(u.images_out) AS images_out,
	SUM(u.web_searches) AS web_searches, SUM(u.web_fetches) AS web_fetches, SUM(u.audio_in) AS audio_in, SUM(u.tool_calls) AS tool_calls,
	group_concat(CASE WHEN u.tool_kinds_json <> '{}' THEN u.tool_kinds_json END, char(30)) AS tool_kinds,
	COALESCE(MAX(u.credits_balance), '') AS credits, MAX(f.harness) AS harness,
	MAX(CASE WHEN u.model NOT IN ('', 'unknown') THEN u.model END) AS model`

// UsageTurns reads the durable transcript usage rows in [from, to]: the
// indexer records them for every file, keyed to the file's run (an
// operator session or an Agent Manager run), so each report is counted once
// from one source (D-16).
func (r *UsageAttributionRepository) UsageTurns(ctx context.Context, from, to time.Time) ([]usageattribution.Turn, error) {
	return r.indexedTurns(ctx, from, to)
}

func (r *UsageAttributionRepository) indexedTurns(ctx context.Context, from, to time.Time) ([]usageattribution.Turn, error) {
	lower, upper := from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano)
	scope := `FROM transcript_usage u JOIN transcript_files f ON f.file_key = u.file_key
		WHERE f.mtime_unix_nano >= ? AND u.observed_at >= ? AND u.observed_at <= ?`
	args := []any{from.UnixNano(), lower, upper}
	var cumulative []usageAggregateRow
	if err := r.db.SelectContext(ctx, &cumulative, `SELECT u.file_key, MAX(u.run_id) AS run_id, '' AS message_id, 1 AS cumulative, MAX(u.observed_at) AS at, COUNT(*) AS reports,
		0 AS authority, MAX(u.cum_input_tokens) AS input, MAX(u.cum_output_tokens) AS output, MAX(u.cum_cache_read_tokens) AS cache_read,
		MAX(u.cum_cache_creation_tokens) AS cache_creation, MIN(u.cum_input_tokens - u.input_tokens) AS before_input,
		MIN(u.cum_output_tokens - u.output_tokens) AS before_output, MIN(u.cum_cache_read_tokens - u.cache_read_tokens) AS before_read,
		MIN(u.cum_cache_creation_tokens - u.cache_creation_tokens) AS before_creation, `+usageAggregateFacts+`
		`+scope+` AND u.has_cumulative = 1 GROUP BY u.file_key, substr(u.observed_at, 1, `+fmt.Sprint(usageBucketPrefix)+`)`, args...); err != nil {
		return nil, wrapDBError("usage_turns", "TranscriptUsage", "", err)
	}
	var messages []usageAggregateRow
	if err := r.db.SelectContext(ctx, &messages, `SELECT u.file_key, MAX(u.run_id) AS run_id, u.message_id, 0 AS cumulative, MIN(u.observed_at) AS at, 1 AS reports,
		MAX(u.reconciliation_authority) AS authority, MAX(u.input_tokens) AS input, MAX(u.output_tokens) AS output,
		MAX(u.cache_read_tokens) AS cache_read, MAX(u.cache_creation_tokens) AS cache_creation,
		0 AS before_input, 0 AS before_output, 0 AS before_read, 0 AS before_creation, `+usageAggregateFacts+`
		`+scope+` AND u.has_cumulative = 0
		GROUP BY u.file_key, CASE WHEN u.message_id = '' THEN printf('%d:%d', u.line_offset, u.ordinal) ELSE u.message_id END`, args...); err != nil {
		return nil, wrapDBError("usage_turns", "TranscriptUsage", "", err)
	}
	byFile := map[string][]usageAggregateRow{}
	for _, row := range append(cumulative, messages...) {
		byFile[row.FileKey] = append(byFile[row.FileKey], row)
	}
	var turns []usageattribution.Turn
	for key, rows := range byFile {
		sort.Slice(rows, func(i, j int) bool { return rows[i].At < rows[j].At })
		events := make([]*domain.RunEvent, len(rows))
		for i, row := range rows {
			at, _ := time.Parse(time.RFC3339Nano, row.At)
			usage := &domain.UsageEventData{PayloadKind: domain.PayloadKindUsage, SessionID: key, MessageID: row.MessageID, ReconciliationAuthority: row.Authority}
			counters := domain.UsageCounters{InputTokens: row.Input, OutputTokens: row.Output, CacheReadTokens: row.CacheRead, CacheCreationTokens: row.CacheCreation}
			if row.Cumulative {
				// The bucket's highest counter and the counter before its first
				// report: AccountUsage credits exactly the bucket's usage.
				cumulative := counters
				usage.Cumulative = &cumulative
				counters = domain.UsageCounters{InputTokens: row.Input - row.BeforeInput, OutputTokens: row.Output - row.BeforeOutput, CacheReadTokens: row.CacheRead - row.BeforeRead, CacheCreationTokens: row.CacheCreation - row.BeforeCreation}
			}
			usage.InputTokens, usage.OutputTokens, usage.CacheReadTokens, usage.CacheCreationTokens = counters.InputTokens, counters.OutputTokens, counters.CacheReadTokens, counters.CacheCreationTokens
			events[i] = &domain.RunEvent{RunID: uuid.Nil, EventType: domain.EventTypeMetric, Timestamp: at, Data: usage}
		}
		account := runsignal.AccountUsage(events)
		for i, row := range rows {
			contribution := account.Contributions[i]
			if contribution.Total() == 0 {
				continue
			}
			turns = append(turns, usageattribution.Turn{
				At: events[i].Timestamp, RunID: row.RunID, Runner: harnessRunner(row.Harness), Model: modelOrUnknown(row.Model),
				Tokens: contribution, Reports: row.Reports, Facts: aggregateFacts(row),
			})
		}
	}
	return turns, nil
}

func harnessRunner(harness string) string {
	if harness == transcriptindex.HarnessClaude {
		return string(domain.RunnerTypeClaudeCode)
	}
	return string(domain.RunnerTypeCodex)
}

func modelOrUnknown(model sql.NullString) string {
	if model.Valid && strings.TrimSpace(model.String) != "" {
		return model.String
	}
	return "unknown"
}

func aggregateFacts(row usageAggregateRow) transcriptindex.TurnFacts {
	facts := transcriptindex.TurnFacts{
		ImagesIn: row.ImagesIn, ImagesInUser: row.ImagesInUser, ImagesInHighDetail: row.ImagesInHigh, ImageBytesIn: row.ImageBytesIn,
		ImagesOut: row.ImagesOut, WebSearches: row.WebSearches, WebFetches: row.WebFetches, AudioIn: row.AudioIn, ToolCalls: row.ToolCalls,
		CreditsBalance: row.Credits,
	}
	if row.ToolKinds.Valid {
		for _, encoded := range strings.Split(row.ToolKinds.String, "\x1e") {
			var kinds map[string]int
			if json.Unmarshal([]byte(encoded), &kinds) != nil {
				continue
			}
			for kind, count := range kinds {
				if facts.ToolKinds == nil {
					facts.ToolKinds = map[string]int{}
				}
				facts.ToolKinds[kind] += count
			}
		}
	}
	return facts
}

func parseEventTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if at, err := time.Parse(layout, value); err == nil {
			return at, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable event time %q", value)
}

type usageRunRow struct {
	ID               string `db:"id"`
	Label            string `db:"label"`
	Mode             string `db:"execution_mode"`
	ProfileID        string `db:"profile_id"`
	ProfileName      string `db:"profile_name"`
	WorkloadKind     string `db:"workload_kind"`
	WorkloadKey      string `db:"workload_key"`
	Status           string `db:"status"`
	Phase            string `db:"phase"`
	ParentRunID      string `db:"parent_run_id"`
	RunWebConsole    string `db:"run_web_console"`
	Runner           string `db:"runner"`
	FileWebConsole   string `db:"file_web_console"`
	FileSession      string `db:"file_session"`
	FileHeader       string `db:"file_header"`
	FileParent       string `db:"file_parent"`
	FileSidechain    bool   `db:"file_sidechain"`
	FileHarness      string `db:"file_harness"`
	ImportSessionID  string `db:"import_session"`
	ImportHarness    string `db:"import_harness"`
	HasIndexedSource bool   `db:"indexed"`
	StartedAt        string `db:"started_at"`
}

// UsageRuns resolves the grouping dimensions of runIDs. Sessions group a run
// with its sub-agents and forks; a session takes the label of its top-level
// run.
func (r *UsageAttributionRepository) UsageRuns(ctx context.Context, runIDs []string) (map[string]usageattribution.Run, error) {
	out := make(map[string]usageattribution.Run, len(runIDs))
	started := map[string]string{}
	for start := 0; start < len(runIDs); start += 500 {
		chunk := runIDs[start:min(len(runIDs), start+500)]
		query := `SELECT r.id, COALESCE(r.label, '') AS label, COALESCE(r.execution_mode, '') AS execution_mode,
			COALESCE(r.agent_profile_id, '') AS profile_id, COALESCE(p.name, '') AS profile_name,
			COALESCE(r.workload_kind, '') AS workload_kind, COALESCE(r.workload_key, '') AS workload_key,
			COALESCE(r.status, '') AS status, COALESCE(r.phase, '') AS phase,
			COALESCE(r.parent_run_id, '') AS parent_run_id, COALESCE(r.web_console_session_id, '') AS run_web_console,
			COALESCE(json_extract(r.resolved_config, '$.runnerType'), r.harness_kind, '') AS runner,
			COALESCE(f.web_console_session_id, '') AS file_web_console, COALESCE(f.session_id, '') AS file_session,
			COALESCE(f.header_session_id, '') AS file_header, COALESCE(f.parent_session_id, '') AS file_parent,
			COALESCE(f.sidechain, 0) AS file_sidechain, COALESCE(f.harness, '') AS file_harness,
			COALESCE(r.import_source_session_id, '') AS import_session, COALESCE(r.import_source_harness, '') AS import_harness,
			f.file_key IS NOT NULL AS indexed, COALESCE(r.started_at, r.created_at, '') AS started_at
			FROM runs r LEFT JOIN agent_profiles p ON p.id = r.agent_profile_id
			LEFT JOIN transcript_files f ON f.run_id = r.id AND f.owner = 'indexer'
			WHERE r.id IN (?` + strings.Repeat(",?", len(chunk)-1) + `)`
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		var rows []usageRunRow
		if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
			return nil, wrapDBError("usage_runs", "Run", "", err)
		}
		for _, row := range rows {
			out[row.ID] = usageRun(row)
			started[row.ID] = row.StartedAt
		}
	}
	// A session's label is its first top-level run's title; a harness's
	// generic "codex session <id>" label is used only when nothing else is.
	labels, labelStart := map[string]string{}, map[string]string{}
	for id, run := range out {
		if run.Child || run.Label == "" {
			continue
		}
		current, generic := labels[run.Session], genericSessionLabel(run.Label)
		switch {
		case current == "",
			genericSessionLabel(current) && !generic,
			genericSessionLabel(current) == generic && started[id] < labelStart[run.Session]:
			labels[run.Session], labelStart[run.Session] = run.Label, started[id]
		}
	}
	for id, run := range out {
		if label := labels[run.Session]; label != "" {
			run.SessionLabel = label
		}
		if run.SessionLabel == "" {
			run.SessionLabel = "session " + usageattribution.ShortID(strings.TrimPrefix(run.Session, "web-console:"))
		} else {
			run.SessionLabel = usageattribution.ShortID(strings.TrimPrefix(run.Session, "web-console:")) + " " + run.SessionLabel
		}
		out[id] = run
	}
	return out, nil
}

func genericSessionLabel(label string) bool {
	return strings.HasPrefix(label, "codex session ") || strings.HasPrefix(label, "claude session ") || strings.HasPrefix(label, "claude-code session ")
}

func usageRun(row usageRunRow) usageattribution.Run {
	run := usageattribution.Run{
		ID: row.ID, Label: strings.TrimSpace(row.Label), Runner: row.Runner, ProfileID: row.ProfileID, ProfileName: row.ProfileName,
		Status: row.Status, Succeeded: row.Status == string(domain.RunStatusComplete), ParentRunID: row.ParentRunID,
	}
	if row.HasIndexedSource || row.Mode == "imported" {
		run.Source = usageattribution.SourceIndexed
		run.Runner = harnessRunner(firstNonEmpty(row.FileHarness, row.ImportHarness))
		run.Child = row.FileSidechain || row.FileParent != "" || row.ParentRunID != ""
		session := firstNonEmpty(row.FileParent, row.FileHeader, row.FileSession, row.ImportSessionID, row.ID)
		if webConsole := firstNonEmpty(row.FileWebConsole, row.RunWebConsole); webConsole != "" {
			session = "web-console:" + webConsole
		}
		run.Session = session
		return run
	}
	run.Source = usageattribution.SourceAgentManager
	if row.WorkloadKey != "" {
		run.Workload = row.WorkloadKind + ":" + row.WorkloadKey
	}
	// Each Agent Manager run is its own harness session: a parent run
	// orchestrates it but does not share its context.
	run.Session = row.ID
	return run
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type usageOutcomeRow struct {
	RunID        string `db:"run_id"`
	Runner       string `db:"runner_type"`
	Model        string `db:"model"`
	ProfileID    string `db:"profile_id"`
	WorkloadKind string `db:"workload_kind"`
	WorkloadKey  string `db:"workload_key"`
	Status       string `db:"status"`
	EndedAt      string `db:"ended_at"`
	Input        int    `db:"input_tokens"`
	Output       int    `db:"output_tokens"`
	CacheRead    int    `db:"cache_read_tokens"`
	CacheWrite   int    `db:"cache_creation_tokens"`
}

// UsageOutcomes reads finished Agent Manager runs with a workload in
// [from, to] from the invocation read model.
func (r *UsageAttributionRepository) UsageOutcomes(ctx context.Context, from, to time.Time) ([]usageattribution.Outcome, error) {
	var rows []usageOutcomeRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT run_id, runner_type, model, profile_id, workload_kind, workload_key, status, COALESCE(ended_at, occurred_at) AS ended_at,
		input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens FROM invocation_read_model_runs WHERE workload_key <> '' AND COALESCE(ended_at, occurred_at) >= ? AND COALESCE(ended_at, occurred_at) <= ?`,
		from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, wrapDBError("usage_outcomes", "Run", "", err)
	}
	out := make([]usageattribution.Outcome, 0, len(rows))
	for _, row := range rows {
		at, _ := parseEventTime(row.EndedAt)
		out = append(out, usageattribution.Outcome{
			RunID: row.RunID, Runner: row.Runner, Model: row.Model, ProfileID: row.ProfileID, Workload: row.WorkloadKind + ":" + row.WorkloadKey, Succeeded: row.Status == string(domain.RunStatusComplete), At: at,
			Tokens: domain.UsageCounters{InputTokens: row.Input, OutputTokens: row.Output, CacheReadTokens: row.CacheRead, CacheCreationTokens: row.CacheWrite},
		})
	}
	return out, nil
}
