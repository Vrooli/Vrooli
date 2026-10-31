// Responsibility: read and write the run ledger (run_ledger_facts,
// usage_mode_periods) for package outcomeledger. The ledger reads transcript
// usage, projected facts and run rows; it never reads run_events.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/interventions"
	"agent-manager/internal/outcomeledger"
)

const migrationRunLedgerBackfill = "2026-10-30-run-ledger-backfill"

// RunLedgerRepository implements outcomeledger.Store and Reader.
type RunLedgerRepository struct {
	db            *DB
	parserVersion func(harness string) int
}

// NewRunLedgerRepository builds the store; parserVersion names the current
// transcript parser of a harness.
func NewRunLedgerRepository(db *DB, parserVersion func(string) int) *RunLedgerRepository {
	return &RunLedgerRepository{db: db, parserVersion: parserVersion}
}

var (
	_ outcomeledger.Store  = (*RunLedgerRepository)(nil)
	_ outcomeledger.Reader = (*RunLedgerRepository)(nil)
)

// The stamp includes the run's newest operator-signal record, so a signal that
// arrives after the row was built (back-fill, retry) refreshes the row.
var ledgerStamp = `COALESCE(r.updated_at, '') || '|' || COALESCE(w.projected_at, '') || '|' || COALESCE(f.indexed_at, '') || '|' ||
	COALESCE((SELECT MAX(recorded_at) FROM operator_signals os WHERE os.run_id = r.id AND os.signal_version = '` + interventions.SignalVersion + `'), '')`

var ledgerPendingFrom = `FROM runs r
	LEFT JOIN transcript_files f ON f.run_id = r.id
	LEFT JOIN invocation_read_model_watermarks w ON w.run_id = r.id
	LEFT JOIN run_ledger_facts l ON l.run_id = r.id
	WHERE r.created_at >= ? AND (l.run_id IS NULL OR l.source_stamp <> ` + ledgerStamp + `)`

type ledgerRunRow struct {
	ID            string         `db:"id"`
	Stamp         string         `db:"stamp"`
	Mode          sql.NullString `db:"execution_mode"`
	Status        sql.NullString `db:"status"`
	Terminal      sql.NullString `db:"terminal_class"`
	Stop          sql.NullString `db:"stop_reason"`
	Approval      sql.NullString `db:"approval_state"`
	ApprovedBy    sql.NullString `db:"approved_by"`
	Parent        sql.NullString `db:"parent_run_id"`
	Task          sql.NullString `db:"task_id"`
	Goal          sql.NullString `db:"goal_id"`
	SessionID     sql.NullString `db:"session_id"`
	ImportSession sql.NullString `db:"import_source_session_id"`
	ImportHarness sql.NullString `db:"import_source_harness"`
	HarnessKind   sql.NullString `db:"harness_kind"`
	Console       sql.NullString `db:"web_console_session_id"`
	WorkloadKind  sql.NullString `db:"workload_kind"`
	WorkloadKey   sql.NullString `db:"workload_key"`
	Sources       sql.NullString `db:"source_run_ids"`
	Created       sql.NullString `db:"created_at"`
	Started       sql.NullString `db:"started_at"`
	Ended         sql.NullString `db:"ended_at"`
	Updated       sql.NullString `db:"updated_at"`
	Requested     sql.NullString `db:"requested_model"`
	Actual        sql.NullString `db:"actual_model"`
	CanaryArm     sql.NullString `db:"canary_arm"`
	Runner        sql.NullString `db:"runner"`
	Role          sql.NullString `db:"role"`
	Effort        sql.NullString `db:"effort"`
	ConfigModel   sql.NullString `db:"config_model"`
	Selection     sql.NullString `db:"selection_reason"`
	RootKind      sql.NullString `db:"root_kind"`
	FileHarness   sql.NullString `db:"file_harness"`
	FileStatus    sql.NullString `db:"file_status"`
	FileOffset    sql.NullInt64  `db:"file_offset"`
	FileSize      sql.NullInt64  `db:"file_size"`
	FileMtime     sql.NullInt64  `db:"file_mtime"`
	FileEvicted   sql.NullInt64  `db:"file_evicted"`
	FileParser    sql.NullInt64  `db:"file_parser"`
	FileKnown     bool           `db:"file_known"`
}

// LoadPending fills per-run facts for the runs changed since their row.
func (r *RunLedgerRepository) LoadPending(ctx context.Context, since time.Time, limit int) ([]outcomeledger.RunFacts, error) {
	var rows []ledgerRunRow
	query := `SELECT r.id, ` + ledgerStamp + ` AS stamp, r.execution_mode, r.status, r.terminal_class, r.stop_reason, r.approval_state, r.approved_by,
		r.parent_run_id, r.task_id, r.goal_id, r.session_id, r.import_source_session_id, r.import_source_harness, r.harness_kind,
		r.web_console_session_id, r.workload_kind, r.workload_key, r.source_run_ids, r.created_at, r.started_at, r.ended_at, r.updated_at,
		r.requested_model, r.actual_model, r.canary_arm,
		COALESCE(json_extract(r.resolved_config, '$.runnerType'), '') AS runner,
		COALESCE(json_extract(r.resolved_config, '$.roleRef'), '') AS role,
		COALESCE(json_extract(r.resolved_config, '$.effort'), '') AS effort,
		COALESCE(json_extract(r.resolved_config, '$.model'), '') AS config_model,
		COALESCE(json_extract(r.resolved_config, '$.policySnapshot.selectionReason'), '') AS selection_reason,
		f.root_kind, f.harness AS file_harness, f.status AS file_status, f.read_offset AS file_offset, f.size AS file_size,
		f.mtime_unix_nano AS file_mtime, f.recall_evicted AS file_evicted, f.parser_version AS file_parser, f.file_key IS NOT NULL AS file_known
		` + ledgerPendingFrom + ` ORDER BY r.created_at DESC LIMIT ?`
	if err := r.db.SelectContext(ctx, &rows, query, ledgerSince(since), limit); err != nil {
		return nil, wrapDBError("ledger_pending", "Run", "", err)
	}
	out := make([]outcomeledger.RunFacts, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.ID] {
			continue // a run with several files: the first row carries its facts
		}
		seen[row.ID] = true
		facts, err := r.runFacts(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, facts)
	}
	return out, nil
}

func ledgerSince(since time.Time) string { return since.UTC().Format("2006-01-02 15:04:05") }

func (r *RunLedgerRepository) runFacts(ctx context.Context, row ledgerRunRow) (outcomeledger.RunFacts, error) {
	status := row.Status.String
	facts := outcomeledger.RunFacts{
		RunID: row.ID, SourceStamp: row.Stamp, Path: outcomeledger.PathFor(row.Mode.String, row.RootKind.String),
		Status: status, Live: !domain.RunStatus(status).IsTerminal(), TerminalClass: row.Terminal.String, StopReason: row.Stop.String,
		ApprovalState: row.Approval.String, ParentRunID: row.Parent.String, GoalID: row.Goal.String, WebConsoleSession: row.Console.String,
		RequestedModel: row.Requested.String, ActualModel: row.Actual.String, Effort: row.Effort.String, Role: row.Role.String, SelectionReason: row.Selection.String, CanaryArm: row.CanaryArm.String,
		FileStatus: row.FileStatus.String, FileFullyRead: row.FileKnown && row.FileOffset.Int64 >= row.FileSize.Int64, RecallEvicted: row.FileEvicted.Int64 != 0,
	}
	if facts.ApprovalState == "approved" && row.ApprovedBy.String == outcomeledger.AutoApprover {
		// The sandbox policy applied the change at run end; nobody judged it.
		facts.ApprovalState = outcomeledger.ApprovalAutomatic
	}
	facts.Harness = firstNonEmpty(row.ImportHarness.String, row.FileHarness.String, row.Runner.String, row.HarnessKind.String, "unknown")
	if facts.Harness == "claude" {
		facts.Harness = string(domain.RunnerTypeClaudeCode)
	}
	if facts.RequestedModel == "" {
		facts.RequestedModel = row.ConfigModel.String
	}
	if row.FileKnown && r.parserVersion != nil && int(row.FileParser.Int64) < r.parserVersion(row.FileHarness.String) {
		facts.OlderParser = true
	}
	switch row.Mode.String {
	case "imported":
		facts.SessionKey = facts.Harness + ":" + outcomeledger.SessionBase(row.ImportSession.String)
	case "attached":
	default:
		facts.SessionKey = row.SessionID.String
		facts.TaskID = row.Task.String
		if row.WorkloadKey.String != "" {
			facts.WorkloadKey = row.WorkloadKind.String + ":" + row.WorkloadKey.String
		}
	}
	if row.Sources.Valid && row.Sources.String != "" {
		_ = json.Unmarshal([]byte(row.Sources.String), &facts.SourceRunIDs)
	}
	facts.CreatedAt = ledgerTime(row.Created.String)
	started, ended, updated := ledgerTime(row.Started.String), ledgerTime(row.Ended.String), ledgerTime(row.Updated.String)
	reports, err := r.usageFacts(ctx, row.ID, &facts)
	if err != nil {
		return facts, err
	}
	wall, active := outcomeledger.ActiveTime(reports)
	facts.ActiveMS = active.Milliseconds()
	switch {
	case len(reports) > 0:
		facts.FirstActivity, facts.LastActivity = reports[0], reports[len(reports)-1]
		facts.WallMS = wall.Milliseconds()
	case row.Mode.String == "imported" && row.FileMtime.Int64 > 0:
		// An imported session with no reports has no event times: its started
		// and ended columns hold the import time, the file's mtime is real.
		mtime := time.Unix(0, row.FileMtime.Int64).UTC()
		facts.FirstActivity, facts.LastActivity = mtime, mtime
	default:
		facts.FirstActivity = started
		if facts.FirstActivity.IsZero() {
			facts.FirstActivity = facts.CreatedAt
		}
		facts.LastActivity = ended
		if facts.LastActivity.IsZero() {
			facts.LastActivity = updated
		}
	}
	facts.SessionStart = facts.FirstActivity
	if row.Mode.String != "imported" {
		facts.SessionStart = started
		// Lifecycle time is what an open managed run costs in wall clock.
		if !started.IsZero() && !ended.IsZero() {
			facts.WallMS = ended.Sub(started).Milliseconds()
		}
	} else if len(reports) > 0 && !started.IsZero() && started.Before(facts.SessionStart) {
		facts.SessionStart = started
	}
	if err := r.callFacts(ctx, row.ID, &facts); err != nil {
		return facts, err
	}
	return facts, nil
}

type ledgerUsageRow struct {
	At        string `db:"observed_at"`
	Model     string `db:"model"`
	MessageID string `db:"message_id"`
	ToolCalls int64  `db:"tool_calls"`
	Kinds     string `db:"tool_kinds_json"`
}

// usageFacts reads a run's usage rows and returns the report times in order.
func (r *RunLedgerRepository) usageFacts(ctx context.Context, runID string, facts *outcomeledger.RunFacts) ([]time.Time, error) {
	var rows []ledgerUsageRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT observed_at, model, message_id, tool_calls, tool_kinds_json FROM transcript_usage WHERE run_id = ? ORDER BY observed_at`, runID); err != nil {
		return nil, wrapDBError("ledger_usage", "Run", runID, err)
	}
	models := map[string]int{}
	messages := map[string]bool{}
	reports := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		if at := ledgerTime(row.At); !at.IsZero() {
			reports = append(reports, at)
		}
		if row.MessageID == "" {
			facts.Requests++
		} else if !messages[row.MessageID] {
			messages[row.MessageID] = true
			facts.Requests++
		}
		facts.TranscriptToolCalls += row.ToolCalls
		if model := strings.TrimSpace(row.Model); model != "" && model != "unknown" {
			models[model]++
		}
		if strings.Contains(row.Kinds, "file-edit") {
			var kinds map[string]int64
			if json.Unmarshal([]byte(row.Kinds), &kinds) == nil {
				facts.Edits += kinds["file-edit"]
			}
		}
	}
	best, count := "", 0
	for model, n := range models {
		if n > count || (n == count && model < best) {
			best, count = model, n
		}
	}
	if best != "" {
		facts.ActualModel = best
	}
	return reports, nil
}

type ledgerCallRow struct {
	Call    string         `db:"call_event_id"`
	At      string         `db:"occurred_at"`
	Tool    string         `db:"tool_name"`
	Path    sql.NullString `db:"command_path"`
	Command string         `db:"tool_command"`
	Help    bool           `db:"tool_help"`
	Outcome string         `db:"outcome"`
	Basis   string         `db:"pairing_basis"`
}

func (r *RunLedgerRepository) callFacts(ctx context.Context, runID string, facts *outcomeledger.RunFacts) error {
	var rows []ledgerCallRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT call_event_id, occurred_at, tool_name, command_path, tool_command, tool_help, outcome, pairing_basis FROM invocation_read_model_facts WHERE run_id = ?`, runID); err != nil {
		return wrapDBError("ledger_calls", "Run", runID, err)
	}
	calls := make([]outcomeledger.CallFact, len(rows))
	for i, row := range rows {
		calls[i] = outcomeledger.CallFact{CallID: row.Call, At: ledgerTime(row.At), Tool: row.Tool, Command: firstNonEmpty(row.Command, row.Path.String), Help: row.Help, Outcome: row.Outcome, Basis: row.Basis}
	}
	summary := outcomeledger.SummarizeCalls(calls, !facts.OlderParser)
	facts.FactRows, facts.Commands, facts.KnownCommands = summary.Rows, summary.Commands, summary.Known
	facts.ValidationsPassed, facts.ValidationsFailed = summary.ValidationsPassed, summary.ValidationsFailed
	facts.LastValidation, facts.LastValidationAt = summary.LastValidation, summary.LastValidationAt
	return nil
}

// ledgerTime reads the timestamp spellings runs and usage rows hold.
func ledgerTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999", "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05"} {
		if at, err := time.Parse(layout, value); err == nil {
			return at.UTC()
		}
	}
	return time.Time{}
}

func ledgerFormat(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339Nano)
}

const ledgerBatch = 200

// SaveFacts upserts per-run facts and leaves the unit fields to SaveUnits, in
// short transactions so API writes keep flowing.
func (r *RunLedgerRepository) SaveFacts(ctx context.Context, rows []outcomeledger.RunFacts) error {
	for start := 0; start < len(rows); start += ledgerBatch {
		chunk := rows[start:min(len(rows), start+ledgerBatch)]
		if err := r.db.retryBusy(ctx, func() error { return r.saveFactsChunk(ctx, chunk) }); err != nil {
			return wrapDBError("ledger_save", "Run", "", err)
		}
	}
	return nil
}

func (r *RunLedgerRepository) saveFactsChunk(ctx context.Context, rows []outcomeledger.RunFacts) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	const columns = `run_id, source_stamp, path, harness, status, live, terminal_class, stop_reason, approval_state, parent_run_id, task_id, goal_id, session_key,
		web_console_session, workload_key, source_run_ids, role, created_at, session_start, first_activity_at, last_activity_at, requested_model, actual_model, effort,
		selection_reason, requests, transcript_tool_calls, fact_rows, commands, known_commands, older_parser, recall_evicted, file_status, file_fully_read,
		edits, validations_passed, validations_failed, last_validation, last_validation_at, wall_ms, active_ms, canary_arm`
	query := `INSERT INTO run_ledger_facts (` + columns + `) VALUES (?` + strings.Repeat(",?", 41) + `)
		ON CONFLICT(run_id) DO UPDATE SET source_stamp=excluded.source_stamp, path=excluded.path, harness=excluded.harness, status=excluded.status, live=excluded.live,
		terminal_class=excluded.terminal_class, stop_reason=excluded.stop_reason, approval_state=excluded.approval_state, parent_run_id=excluded.parent_run_id, task_id=excluded.task_id,
		goal_id=excluded.goal_id, session_key=excluded.session_key, web_console_session=excluded.web_console_session, workload_key=excluded.workload_key, source_run_ids=excluded.source_run_ids,
		role=excluded.role, created_at=excluded.created_at, session_start=excluded.session_start, first_activity_at=excluded.first_activity_at, last_activity_at=excluded.last_activity_at,
		requested_model=excluded.requested_model, actual_model=excluded.actual_model, effort=excluded.effort, selection_reason=excluded.selection_reason, requests=excluded.requests,
		transcript_tool_calls=excluded.transcript_tool_calls, fact_rows=excluded.fact_rows, commands=excluded.commands, known_commands=excluded.known_commands, older_parser=excluded.older_parser,
		recall_evicted=excluded.recall_evicted, file_status=excluded.file_status, file_fully_read=excluded.file_fully_read, edits=excluded.edits, validations_passed=excluded.validations_passed,
		validations_failed=excluded.validations_failed, last_validation=excluded.last_validation, last_validation_at=excluded.last_validation_at, wall_ms=excluded.wall_ms, active_ms=excluded.active_ms, canary_arm=excluded.canary_arm`
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, query, row.RunID, row.SourceStamp, row.Path, row.Harness, row.Status, boolInt(row.Live), row.TerminalClass, row.StopReason, row.ApprovalState, row.ParentRunID, row.TaskID, row.GoalID, row.SessionKey,
			row.WebConsoleSession, row.WorkloadKey, strings.Join(row.SourceRunIDs, ","), row.Role, ledgerFormat(row.CreatedAt), ledgerFormat(row.SessionStart), ledgerFormat(row.FirstActivity), ledgerFormat(row.LastActivity),
			row.RequestedModel, row.ActualModel, row.Effort, row.SelectionReason, row.Requests, row.TranscriptToolCalls, row.FactRows, row.Commands, row.KnownCommands, boolInt(row.OlderParser), boolInt(row.RecallEvicted),
			row.FileStatus, boolInt(row.FileFullyRead), row.Edits, row.ValidationsPassed, row.ValidationsFailed, row.LastValidation, ledgerFormat(row.LastValidationAt), row.WallMS, row.ActiveMS, row.CanaryArm); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SaveUnits writes the fields derived across runs.
func (r *RunLedgerRepository) SaveUnits(ctx context.Context, rows []outcomeledger.RunFacts) error {
	for start := 0; start < len(rows); start += ledgerBatch {
		chunk := rows[start:min(len(rows), start+ledgerBatch)]
		err := r.db.retryBusy(ctx, func() error {
			tx, err := r.db.BeginTxx(ctx, nil)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback() }()
			for _, row := range chunk {
				if _, err := tx.ExecContext(ctx, `UPDATE run_ledger_facts SET mode_id=?, mode_relation=?, mode_basis=?, mode_confidence=?, arm=?, arm_mode=?, arm_basis=?, arm_exception=?, operator_messages=?, attached_class=?, linked_run_id=?, work_unit_id=?, link_kind=?, link_confidence=?, unit_outcome=?, unit_source=? WHERE run_id=?`,
					row.ModeID, row.ModeRelation, row.ModeBasis, row.ModeConfidence, row.Arm, row.ArmMode, row.ArmBasis, row.ArmException, row.OperatorMessages, row.AttachedClass, row.LinkedRunID, row.WorkUnitID, row.LinkKind, row.LinkConfidence, row.UnitOutcome, row.UnitSource, row.RunID); err != nil {
					return err
				}
			}
			return tx.Commit()
		})
		if err != nil {
			return wrapDBError("ledger_units", "Run", "", err)
		}
	}
	return nil
}

type ledgerFactRow struct {
	RunID          string  `db:"run_id"`
	Stamp          string  `db:"source_stamp"`
	Path           string  `db:"path"`
	Harness        string  `db:"harness"`
	Status         string  `db:"status"`
	Live           bool    `db:"live"`
	Terminal       string  `db:"terminal_class"`
	Stop           string  `db:"stop_reason"`
	Approval       string  `db:"approval_state"`
	Parent         string  `db:"parent_run_id"`
	Task           string  `db:"task_id"`
	Goal           string  `db:"goal_id"`
	Session        string  `db:"session_key"`
	Console        string  `db:"web_console_session"`
	Workload       string  `db:"workload_key"`
	Sources        string  `db:"source_run_ids"`
	Role           string  `db:"role"`
	Created        string  `db:"created_at"`
	SessionStart   string  `db:"session_start"`
	First          string  `db:"first_activity_at"`
	Last           string  `db:"last_activity_at"`
	Requested      string  `db:"requested_model"`
	Actual         string  `db:"actual_model"`
	Effort         string  `db:"effort"`
	ModeID         string  `db:"mode_id"`
	ModeRelation   string  `db:"mode_relation"`
	ModeBasis      string  `db:"mode_basis"`
	ModeConfidence float64 `db:"mode_confidence"`
	CanaryArm      string  `db:"canary_arm"`
	Arm            string  `db:"arm"`
	ArmMode        string  `db:"arm_mode"`
	ArmBasis       string  `db:"arm_basis"`
	ArmException   string  `db:"arm_exception"`
	Selection      string  `db:"selection_reason"`
	Requests       int64   `db:"requests"`
	ToolCalls      int64   `db:"transcript_tool_calls"`
	FactRows       int64   `db:"fact_rows"`
	Commands       int64   `db:"commands"`
	Known          int64   `db:"known_commands"`
	Older          bool    `db:"older_parser"`
	Evicted        bool    `db:"recall_evicted"`
	FileStatus     string  `db:"file_status"`
	FileRead       bool    `db:"file_fully_read"`
	Edits          int64   `db:"edits"`
	Passed         int64   `db:"validations_passed"`
	Failed         int64   `db:"validations_failed"`
	LastValidation string  `db:"last_validation"`
	ValidationAt   string  `db:"last_validation_at"`
	Messages       int64   `db:"operator_messages"`
	Wall           int64   `db:"wall_ms"`
	Active         int64   `db:"active_ms"`
	Class          string  `db:"attached_class"`
	Linked         string  `db:"linked_run_id"`
	Unit           string  `db:"work_unit_id"`
	LinkKind       string  `db:"link_kind"`
	LinkConfidence float64 `db:"link_confidence"`
	Outcome        string  `db:"unit_outcome"`
	Source         string  `db:"unit_source"`
}

func (row ledgerFactRow) facts() outcomeledger.RunFacts {
	out := outcomeledger.RunFacts{
		RunID: row.RunID, SourceStamp: row.Stamp, Path: row.Path, Harness: row.Harness, Status: row.Status, Live: row.Live, TerminalClass: row.Terminal, StopReason: row.Stop,
		ApprovalState: row.Approval, ParentRunID: row.Parent, TaskID: row.Task, GoalID: row.Goal, SessionKey: row.Session, WebConsoleSession: row.Console, WorkloadKey: row.Workload,
		Role: row.Role, CreatedAt: ledgerTime(row.Created), SessionStart: ledgerTime(row.SessionStart), FirstActivity: ledgerTime(row.First), LastActivity: ledgerTime(row.Last),
		RequestedModel: row.Requested, ActualModel: row.Actual, Effort: row.Effort, ModeID: row.ModeID, ModeRelation: row.ModeRelation, ModeBasis: row.ModeBasis, ModeConfidence: row.ModeConfidence,
		CanaryArm: row.CanaryArm, Arm: row.Arm, ArmMode: row.ArmMode, ArmBasis: row.ArmBasis, ArmException: row.ArmException,
		SelectionReason: row.Selection, Requests: row.Requests, TranscriptToolCalls: row.ToolCalls, FactRows: row.FactRows, Commands: row.Commands, KnownCommands: row.Known, OlderParser: row.Older,
		RecallEvicted: row.Evicted, FileStatus: row.FileStatus, FileFullyRead: row.FileRead, Edits: row.Edits, ValidationsPassed: row.Passed, ValidationsFailed: row.Failed,
		LastValidation: row.LastValidation, LastValidationAt: ledgerTime(row.ValidationAt), OperatorMessages: row.Messages, WallMS: row.Wall, ActiveMS: row.Active,
		AttachedClass: row.Class, LinkedRunID: row.Linked, WorkUnitID: row.Unit, LinkKind: row.LinkKind, LinkConfidence: row.LinkConfidence, UnitOutcome: row.Outcome, UnitSource: row.Source,
	}
	if row.Sources != "" {
		out.SourceRunIDs = strings.Split(row.Sources, ",")
	}
	return out
}

func (r *RunLedgerRepository) selectFacts(ctx context.Context, operation, where string, args ...any) ([]outcomeledger.RunFacts, error) {
	var rows []ledgerFactRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT run_id, source_stamp, path, harness, status, live, terminal_class, stop_reason, approval_state, parent_run_id, task_id, goal_id, session_key,
		web_console_session, workload_key, source_run_ids, role, created_at, session_start, first_activity_at, last_activity_at, requested_model, actual_model, effort,
		mode_id, mode_relation, mode_basis, mode_confidence, canary_arm, arm, arm_mode, arm_basis, arm_exception, selection_reason, requests, transcript_tool_calls, fact_rows, commands, known_commands, older_parser, recall_evicted, file_status,
		file_fully_read, edits, validations_passed, validations_failed, last_validation, last_validation_at, operator_messages, wall_ms, active_ms, attached_class, linked_run_id,
		work_unit_id, link_kind, link_confidence, unit_outcome, unit_source FROM run_ledger_facts WHERE `+where, args...); err != nil {
		return nil, wrapDBError(operation, "Run", "", err)
	}
	out := make([]outcomeledger.RunFacts, len(rows))
	for i, row := range rows {
		out[i] = row.facts()
	}
	return out, nil
}

// Facts reads every row with activity since since.
func (r *RunLedgerRepository) Facts(ctx context.Context, since time.Time) ([]outcomeledger.RunFacts, error) {
	return r.selectFacts(ctx, "ledger_facts", `last_activity_at >= ?`, ledgerFormat(since))
}

// FactsInWindow reads the runs active in [from, to], with their whole unit's
// rows not implied: a unit's outcome is stored on each of its rows.
func (r *RunLedgerRepository) FactsInWindow(ctx context.Context, from, to time.Time) ([]outcomeledger.RunFacts, error) {
	return r.selectFacts(ctx, "ledger_window", `last_activity_at >= ? AND first_activity_at <= ?`, ledgerFormat(from), ledgerFormat(to))
}

// BackfillCount counts old sessions first imported in the window: their row is
// new, their activity is not.
func (r *RunLedgerRepository) BackfillCount(ctx context.Context, from, to time.Time) (int64, error) {
	var count int64
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM run_ledger_facts WHERE created_at >= ? AND created_at <= ? AND last_activity_at < ? AND path IN (?, ?)`,
		ledgerFormat(from), ledgerFormat(to), ledgerFormat(from), outcomeledger.PathImported, outcomeledger.PathImportedWebConsole); err != nil {
		return 0, wrapDBError("ledger_backfill_count", "Run", "", err)
	}
	return count, nil
}

// PendingCount counts runs changed since their row.
func (r *RunLedgerRepository) PendingCount(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(DISTINCT r.id) `+ledgerPendingFrom, ledgerSince(since)); err != nil {
		return 0, wrapDBError("ledger_pending_count", "Run", "", err)
	}
	return count, nil
}

type ledgerSignalRow struct {
	RunID       string          `db:"run_id"`
	Status      string          `db:"status"`
	Category    string          `db:"category"`
	Probability sql.NullFloat64 `db:"probability"`
}

// Signals reads the classified operator messages of the current classifier and
// the operator-signal labels of the current signal version. A message counts as
// a label at the operating point its hold-out precision was measured at: the
// label's own probability against interventions.SignalOperatingPoints, whether
// or not it is the most likely label. A model or label with no measured operating
// point is not read, so an unmeasured classifier never moves an estimate.
func (r *RunLedgerRepository) Signals(ctx context.Context, since time.Time) (map[string]outcomeledger.Signal, error) {
	var rows []ledgerSignalRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT run_id, status, category, probability FROM operator_interventions WHERE classifier_version = ? AND occurred_at >= ? AND status <> 'duplicate'`,
		interventions.ClassifierVersion, since.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, wrapDBError("ledger_signals", "OperatorIntervention", "", err)
	}
	out := map[string]outcomeledger.Signal{}
	for _, row := range rows {
		signal := out[row.RunID]
		signal.Messages++
		if row.Status == "classified" && outcomeledger.IsCorrection(row.Category) {
			signal.Corrections = append(signal.Corrections, outcomeledger.Correction{RunID: row.RunID, Category: row.Category, Probability: row.Probability.Float64})
		}
		out[row.RunID] = signal
	}
	var labelled []ledgerOperatorSignalRow
	if err := r.db.SelectContext(ctx, &labelled, `SELECT run_id, event_id, occurred_at, probabilities_json, model FROM operator_signals
WHERE signal_version = ? AND status = 'classified' AND occurred_at >= ? ORDER BY run_id, occurred_at`,
		interventions.SignalVersion, since.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, wrapDBError("ledger_operator_signals", "OperatorSignal", "", err)
	}
	// The newest operator message of each run, whatever its label: a reported
	// failure there is the operator ending on the failure.
	var finals []struct {
		RunID   string `db:"run_id"`
		EventID string `db:"event_id"`
	}
	if err := r.db.SelectContext(ctx, &finals, `SELECT run_id, event_id FROM (
  SELECT run_id, event_id, ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY occurred_at DESC, sequence DESC) AS rn
  FROM operator_interventions WHERE classifier_version = ? AND status IN ('classified', 'unclassified')) WHERE rn = 1`, interventions.ClassifierVersion); err != nil {
		return nil, wrapDBError("ledger_final_messages", "OperatorIntervention", "", err)
	}
	final := make(map[string]string, len(finals))
	for _, f := range finals {
		final[f.RunID] = f.EventID
	}
	for _, row := range labelled {
		var distribution map[interventions.Signal]float64
		if err := json.Unmarshal([]byte(row.Probabilities), &distribution); err != nil || len(distribution) == 0 {
			continue
		}
		for _, label := range []interventions.Signal{interventions.ReportedFailure, interventions.Correction} {
			probability := distribution[label]
			if !interventions.SignalOperatingPoints.Shown(row.Model, label, &probability) {
				continue
			}
			precision := interventions.SignalHoldout.Precision(row.Model, label)
			if precision == nil {
				continue
			}
			signal := out[row.RunID]
			signal.Corrections = append(signal.Corrections, outcomeledger.Correction{
				RunID: row.RunID, Signal: string(label), Probability: probability, Final: final[row.RunID] == row.EventID, Precision: *precision,
			})
			out[row.RunID] = signal
		}
	}
	return out, nil
}

type ledgerOperatorSignalRow struct {
	RunID         string `db:"run_id"`
	EventID       string `db:"event_id"`
	OccurredAt    string `db:"occurred_at"`
	Probabilities string `db:"probabilities_json"`
	Model         string `db:"model"`
}

// ObserveModes records each active mode and closes those no longer active. The
// mode file keeps no history, so this table is the record of mode periods.
func (r *RunLedgerRepository) ObserveModes(ctx context.Context, active []outcomeledger.ModePeriod, now time.Time) error {
	return r.db.retryBusy(ctx, func() error {
		tx, err := r.db.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()
		ids := make(map[string]bool, len(active))
		for _, mode := range active {
			ids[mode.ID] = true
			if _, err := tx.ExecContext(ctx, `INSERT INTO usage_mode_periods (id, runner, role, profile, targets, started_at, expires_at, kind, label, split, session_opt_in, block_days) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET targets=excluded.targets, expires_at=excluded.expires_at, ended_at='', kind=excluded.kind, label=excluded.label, split=excluded.split, session_opt_in=excluded.session_opt_in, block_days=excluded.block_days`,
				mode.ID, mode.Runner, mode.Role, mode.Profile, strings.Join(mode.Targets, ","), ledgerFormat(mode.StartedAt), ledgerFormat(mode.ExpiresAt), mode.Kind, mode.Label, mode.Split, boolInt(mode.SessionOptIn), mode.BlockDays); err != nil {
				return err
			}
		}
		var open []string
		rows, err := tx.QueryContext(ctx, `SELECT id FROM usage_mode_periods WHERE ended_at = ''`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			if !ids[id] {
				open = append(open, id)
			}
		}
		rows.Close()
		for _, id := range open {
			if _, err := tx.ExecContext(ctx, `UPDATE usage_mode_periods SET ended_at = ? WHERE id = ?`, ledgerFormat(now), id); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

type ledgerModeRow struct {
	ID      string  `db:"id"`
	Runner  string  `db:"runner"`
	Role    string  `db:"role"`
	Profile string  `db:"profile"`
	Targets string  `db:"targets"`
	Started string  `db:"started_at"`
	Expires string  `db:"expires_at"`
	Ended   string  `db:"ended_at"`
	Kind    string  `db:"kind"`
	Label   string  `db:"label"`
	Split   float64 `db:"split"`
	OptIn   int     `db:"session_opt_in"`
	Block   int     `db:"block_days"`
}

// ModePeriods reads every recorded mode period, oldest first.
func (r *RunLedgerRepository) ModePeriods(ctx context.Context) ([]outcomeledger.ModePeriod, error) {
	var rows []ledgerModeRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT id, runner, role, profile, targets, started_at, expires_at, ended_at, kind, label, split, session_opt_in, block_days FROM usage_mode_periods ORDER BY started_at`); err != nil {
		return nil, wrapDBError("ledger_modes", "UsageMode", "", err)
	}
	out := make([]outcomeledger.ModePeriod, len(rows))
	for i, row := range rows {
		out[i] = outcomeledger.ModePeriod{ID: row.ID, Runner: row.Runner, Role: row.Role, Profile: row.Profile, StartedAt: ledgerTime(row.Started), ExpiresAt: ledgerTime(row.Expires), EndedAt: ledgerTime(row.Ended),
			Kind: row.Kind, Label: row.Label, Split: row.Split, SessionOptIn: row.OptIn != 0, BlockDays: row.Block}
		if row.Targets != "" {
			out[i].Targets = strings.Split(row.Targets, ",")
		}
	}
	return out, nil
}

// MarkBackfilled records that a pass found fewer pending runs than a batch.
func (r *RunLedgerRepository) MarkBackfilled(ctx context.Context) error {
	return r.db.retryBusy(ctx, func() error {
		_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations_done (name, completed_at) VALUES (?, ?)`, migrationRunLedgerBackfill, time.Now().UTC().Format(time.RFC3339Nano))
		return err
	})
}
