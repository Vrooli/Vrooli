package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/interventions"
	"agent-manager/internal/outcomeledger"

	"github.com/vrooli/cli-core/usagemode"
)

// A run's facts come from its durable rows, and the job reads no run_events.
func TestRunLedgerReadsDurableRowsAcrossPaths(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	at := func(minutes int) string {
		return now.Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339Nano)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	created := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	exec(`INSERT INTO runs (id, execution_mode, status, import_source_harness, import_source_session_id, started_at, ended_at, created_at, updated_at, run_mode) VALUES ('imp', 'imported', 'complete', 'codex', 's1', ?, ?, ?, ?, 'in_place')`, at(-30), at(5), created, created)
	exec(`INSERT INTO runs (id, execution_mode, status, started_at, ended_at, created_at, updated_at, run_mode, resolved_config) VALUES ('head', 'codec_pipe', 'complete', ?, ?, ?, ?, 'sandboxed', '{"runnerType":"codex","roleRef":"code.default","effort":"medium","model":"gpt-6-luna","policySnapshot":{"selectionReason":"role;usage_mode:um-1"}}')`, at(-20), at(-10), created, created)
	exec(`INSERT INTO runs (id, execution_mode, status, harness_kind, started_at, ended_at, created_at, updated_at, run_mode) VALUES ('att', 'attached', 'complete', 'grok', ?, ?, ?, ?, 'in_place')`, at(-20), at(-19), created, created)
	exec(`INSERT INTO transcript_files (file_key, path, root_kind, root_path, harness, run_id, status, read_offset, size, mtime_unix_nano, parser_version, first_seen_at) VALUES ('f1', '/x', 'codex-home', '/r', 'codex', 'imp', 'indexed', 10, 10, 1, 1, ?)`, at(-30))
	for i, minute := range []int{-30, -29, -28, -5} {
		exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, tool_calls, tool_kinds_json) VALUES ('f1', ?, 0, 'imp', ?, 'gpt-6.1-terra', 1, '{"shell":1,"file-edit":2}')`, i, at(minute))
	}
	exec(`INSERT INTO invocation_read_model_facts (run_id, call_event_id, occurred_at, tool_name, ownership, outcome, fingerprint, availability, classifier_version, run_status, tool_command) VALUES ('imp', 'c1', ?, 'exec', 'x', 'success', 'f', 'a', 'v', 'complete', 'go test ./...')`, at(-5))

	repo := NewRunLedgerRepository(db, func(string) int { return 3 })
	job := &outcomeledger.Job{Store: repo, Modes: func(time.Time) []outcomeledger.ModePeriod {
		return []outcomeledger.ModePeriod{{ID: "um-1", Runner: "codex", Targets: []string{"gpt-6-luna"}, StartedAt: now.Add(-time.Hour)}}
	}}
	if pending, err := job.Pass(ctx); err != nil || pending != 3 {
		t.Fatalf("pass: %d %v", pending, err)
	}
	rows, err := repo.FactsInWindow(ctx, now.Add(-2*time.Hour), now.Add(time.Hour))
	if err != nil || len(rows) != 3 {
		t.Fatalf("facts: %d %v", len(rows), err)
	}
	byID := map[string]outcomeledger.RunFacts{}
	for _, row := range rows {
		byID[row.RunID] = row
	}
	imp := byID["imp"]
	if imp.Path != outcomeledger.PathImported || imp.ActualModel != "gpt-6.1-terra" || imp.Requests != 4 || imp.Edits != 8 || imp.TranscriptToolCalls != 4 || imp.Commands != 1 || imp.KnownCommands != 1 || !imp.OlderParser {
		t.Fatalf("imported facts: %+v", imp)
	}
	// Three minutes of work then 23 of silence: not active.
	if imp.WallMS != (25*time.Minute).Milliseconds() || imp.ActiveMS != (2*time.Minute).Milliseconds() {
		t.Fatalf("time: wall %d active %d", imp.WallMS, imp.ActiveMS)
	}
	if imp.UnitOutcome != outcomeledger.OutcomeAccepted || imp.UnitSource != outcomeledger.SourceValidation || imp.ModeRelation != outcomeledger.ModeNotChanged || imp.WorkUnitID != "imp" {
		t.Fatalf("imported unit: %+v", imp)
	}
	head := byID["head"]
	if head.Path != outcomeledger.PathHeadless || head.Role != "code.default" || head.Effort != "medium" || head.RequestedModel != "gpt-6-luna" || head.ModeRelation != outcomeledger.ModeApplied || head.ModeID != "um-1" {
		t.Fatalf("headless facts: %+v", head)
	}
	if att := byID["att"]; att.AttachedClass != outcomeledger.AttachedHelper || att.Harness != "grok" || att.WorkUnitID != "" {
		t.Fatalf("attached facts: %+v", att)
	}
	if pending, err := repo.PendingCount(ctx, now.Add(-outcomeledger.Lookback)); err != nil || pending != 0 {
		t.Fatalf("pending after pass: %d %v", pending, err)
	}
	var marker int
	if err := db.GetContext(ctx, &marker, `SELECT COUNT(*) FROM schema_migrations_done WHERE name = ?`, migrationRunLedgerBackfill); err != nil || marker != 1 {
		t.Fatalf("backfill marker: %d %v", marker, err)
	}
	// A mode that left the file is closed, and the period stays on record.
	job.Modes = func(time.Time) []outcomeledger.ModePeriod { return nil }
	if _, err := job.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	periods, err := repo.ModePeriods(ctx)
	if err != nil || len(periods) != 1 || periods[0].EndedAt.IsZero() {
		t.Fatalf("periods: %+v %v", periods, err)
	}
}

// Operator-signal labels reach the outcome estimate with their source, capped
// by the classifier's measured precision, and a signal recorded after the row
// was built refreshes it.
func TestRunLedgerFeedsOperatorSignalsIntoEstimates(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	original := interventions.SignalHoldout
	interventions.SignalHoldout = interventions.SignalMeasurements{"winnow:e4b": {interventions.ReportedFailure: {TP: 6, Pred: 10, Gold: 11}}}
	defer func() { interventions.SignalHoldout = original }()
	now := time.Now().UTC().Truncate(time.Second)
	at := func(minutes int) string {
		return now.Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339Nano)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	created := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	exec(`INSERT INTO runs (id, execution_mode, status, import_source_harness, import_source_session_id, started_at, ended_at, created_at, updated_at, run_mode) VALUES ('imp', 'imported', 'complete', 'claude', 's1', ?, ?, ?, ?, 'in_place')`, at(-30), at(5), created, created)
	exec(`INSERT INTO transcript_files (file_key, path, root_kind, root_path, harness, run_id, status, read_offset, size, mtime_unix_nano, parser_version, first_seen_at) VALUES ('f1', '/x', 'claude-home', '/r', 'claude', 'imp', 'indexed', 10, 10, 1, 1, ?)`, at(-30))
	exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, tool_calls, tool_kinds_json) VALUES ('f1', 1, 0, 'imp', ?, 'claude-x', 1, '{}')`, at(-30))
	intervention := func(id string, minute int) {
		exec(`INSERT INTO operator_interventions (event_id, classifier_version, run_id, sequence, occurred_at, message_key, status, recorded_at) VALUES (?, ?, 'imp', ?, ?, ?, 'classified', ?)`,
			id, interventions.ClassifierVersion, minute+100, at(minute), id, at(minute))
	}
	signal := func(id string, minute int, label, distribution, recorded string) {
		exec(`INSERT INTO operator_signals (event_id, signal_version, run_id, occurred_at, status, label, probability, probabilities_json, model, recorded_at) VALUES (?, ?, 'imp', ?, 'classified', ?, 0.9, ?, 'winnow:e4b', ?)`,
			id, interventions.SignalVersion, at(minute), label, distribution, recorded)
	}
	intervention("e1", -20)
	intervention("e2", -10)
	// Most likely neutral, and over the measured reported_failure point (0.15): the
	// hold-out precision was measured at this point, so the message counts.
	signal("e1", -20, "neutral", `{"reported_failure":0.2,"neutral":0.7,"correction":0.1}`, at(-9))
	// The argmax cut of 0.5 was never measured; under the point nothing is read.
	signal("e2", -10, "neutral", `{"reported_failure":0.1,"neutral":0.8,"correction":0.1}`, at(-9))

	repo := NewRunLedgerRepository(db, func(string) int { return 3 })
	job := &outcomeledger.Job{Store: repo}
	if _, err := job.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	unit := func() outcomeledger.RunFacts {
		t.Helper()
		rows, err := repo.FactsInWindow(ctx, now.Add(-2*time.Hour), now.Add(time.Hour))
		if err != nil || len(rows) != 1 {
			t.Fatalf("facts: %d %v", len(rows), err)
		}
		return rows[0]
	}
	if got := unit(); got.UnitOutcome != outcomeledger.OutcomeCorrected || got.UnitSource != outcomeledger.SourceOperatorReportedFailure {
		t.Fatalf("earlier failure report: %+v, want corrected from operator_reported_failure", got)
	}
	// The operator ends the session on a new failure report: a later signal
	// refreshes the finished row, and the unit failed.
	intervention("e3", -2)
	signal("e3", -2, "reported_failure", `{"reported_failure":0.9,"neutral":0.1}`, at(-1))
	if pending, err := repo.PendingCount(ctx, now.Add(-outcomeledger.Lookback)); err != nil || pending != 1 {
		t.Fatalf("pending after a new signal = %d %v, want the run refreshed", pending, err)
	}
	if _, err := job.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	if got := unit(); got.UnitOutcome != outcomeledger.OutcomeFailed || got.UnitSource != outcomeledger.SourceOperatorReportedFailure {
		t.Fatalf("final failure report: %+v, want failed", got)
	}
	// An unmeasured classifier moves nothing.
	interventions.SignalHoldout = interventions.SignalMeasurements{}
	signal("e4", -1, "reported_failure", `{"reported_failure":0.9,"neutral":0.1}`, at(0))
	exec(`DELETE FROM operator_signals WHERE event_id IN ('e1', 'e3')`)
	if _, err := job.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	if got := unit(); got.UnitSource == outcomeledger.SourceOperatorReportedFailure {
		t.Fatalf("an unmeasured signal fed the estimate: %+v", got)
	}
}

// A Learn comparison survives the database: the managed run's canary arm is a
// fact, the operator session's arm is inferred from the opted-in day-block
// calendar, and the learn period keeps how it randomizes.
func TestRunLedgerRecordsLearnArmsFromCanaryAndCalendar(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	created := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	exec(`INSERT INTO runs (id, execution_mode, status, started_at, ended_at, created_at, updated_at, run_mode, canary_arm, resolved_config) VALUES ('managed', 'codec_pipe', 'complete', ?, ?, ?, ?, 'sandboxed', 'challenger', '{"runnerType":"codex","roleRef":"code.economy.delivery","model":"gpt-6.1-terra","policySnapshot":{"selectionReason":"role;usage_mode_learn:um-l"}}')`,
		now.Add(-20*time.Minute).Format(time.RFC3339Nano), now.Add(-10*time.Minute).Format(time.RFC3339Nano), created, created)
	exec(`INSERT INTO runs (id, execution_mode, status, import_source_harness, import_source_session_id, started_at, ended_at, created_at, updated_at, run_mode) VALUES ('session', 'imported', 'complete', 'codex', 's1', ?, ?, ?, ?, 'in_place')`,
		now.Add(-30*time.Minute).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), created, created)
	exec(`INSERT INTO transcript_files (file_key, path, root_kind, root_path, harness, run_id, status, read_offset, size, mtime_unix_nano, parser_version, first_seen_at) VALUES ('f1', '/x', 'codex-home', '/r', 'codex', 'session', 'indexed', 10, 10, 1, 3, ?)`, now.Add(-30*time.Minute).Format(time.RFC3339Nano))
	exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, tool_calls, tool_kinds_json) VALUES ('f1', 1, 0, 'session', ?, 'gpt-6.1-terra', 1, '{}')`, now.Add(-30*time.Minute).Format(time.RFC3339Nano))
	exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, tool_calls, tool_kinds_json) VALUES ('f1', 2, 0, 'managed', ?, 'gpt-6.1-terra', 1, '{}')`, now.Add(-15*time.Minute).Format(time.RFC3339Nano))

	repo := NewRunLedgerRepository(db, func(string) int { return 3 })
	job := &outcomeledger.Job{Store: repo, Markers: func(time.Time) []usagemode.LaunchMarker { return nil }, Modes: func(time.Time) []outcomeledger.ModePeriod {
		return []outcomeledger.ModePeriod{{ID: "um-l", Kind: usagemode.KindLearn, Runner: "codex", Role: "code.economy.delivery", Split: 0.5, SessionOptIn: true, BlockDays: 1, StartedAt: now.Add(-2 * time.Hour)}}
	}}
	if _, err := job.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.FactsInWindow(ctx, now.Add(-3*time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]outcomeledger.RunFacts{}
	for _, row := range rows {
		byID[row.RunID] = row
	}
	managed, session := byID["managed"], byID["session"]
	if managed.CanaryArm != "challenger" || managed.Arm != "challenger" || managed.ArmMode != "um-l" || managed.ArmBasis != outcomeledger.ArmBasisCanary {
		t.Fatalf("managed arm facts: %+v", managed)
	}
	wantArm := usagemode.SessionArm("um-l", session.SessionStart, 0.5, 1)
	if session.Arm != wantArm || session.ArmMode != "um-l" || session.ArmBasis != outcomeledger.ArmBasisCalendar {
		t.Fatalf("session arm = %q basis %q mode %q, want %s from the calendar", session.Arm, session.ArmBasis, session.ArmMode, wantArm)
	}
	periods, err := repo.ModePeriods(ctx)
	if err != nil || len(periods) != 1 || periods[0].Kind != usagemode.KindLearn || periods[0].Split != 0.5 || !periods[0].SessionOptIn || periods[0].BlockDays != 1 {
		t.Fatalf("learn period did not round-trip: %+v %v", periods, err)
	}
}

// A change the sandbox policy applied at run end was never judged; only a
// person's approval of a run in review is acceptance.
func TestRunLedgerCountsOnlyAnOperatorApprovalAsAcceptance(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	at := func(minutes int) string {
		return now.Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339Nano)
	}
	created := now.Add(-time.Hour).Format("2006-01-02 15:04:05")
	for i, approver := range []string{"applyAtRunEnd", "operator-1"} {
		id, key := []string{"auto", "manual"}[i], []string{"fa", "fm"}[i]
		for _, query := range []string{
			`INSERT INTO runs (id, execution_mode, status, approval_state, approved_by, started_at, ended_at, created_at, updated_at, run_mode, resolved_config) VALUES ('` + id + `', 'codec_pipe', 'complete', 'approved', '` + approver + `', '` + at(-20) + `', '` + at(-10) + `', '` + created + `', '` + created + `', 'sandboxed', '{"runnerType":"codex","roleRef":"code.default"}')`,
			`INSERT INTO transcript_files (file_key, path, root_kind, root_path, harness, run_id, status, read_offset, size, mtime_unix_nano, parser_version, first_seen_at) VALUES ('` + key + `', '/x', 'codex-home', '/r', 'codex', '` + id + `', 'indexed', 10, 10, 1, 1, '` + at(-20) + `')`,
			`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at, model, tool_calls, tool_kinds_json) VALUES ('` + key + `', 0, 0, '` + id + `', '` + at(-15) + `', 'gpt-6.1-terra', 0, '{}')`,
		} {
			if _, err := db.ExecContext(ctx, query); err != nil {
				t.Fatalf("%s: %v", query, err)
			}
		}
	}
	repo := NewRunLedgerRepository(db, func(string) int { return 3 })
	job := &outcomeledger.Job{Store: repo, Modes: func(time.Time) []outcomeledger.ModePeriod { return nil }}
	if _, err := job.Pass(ctx); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.FactsInWindow(ctx, now.Add(-2*time.Hour), now.Add(time.Hour))
	if err != nil || len(rows) != 2 {
		t.Fatalf("facts: %d %v", len(rows), err)
	}
	want := map[string][2]string{
		"auto":   {outcomeledger.OutcomeCompleted, outcomeledger.SourceHeadlessStatus},
		"manual": {outcomeledger.OutcomeAccepted, outcomeledger.SourceOperatorApproval},
	}
	for _, row := range rows {
		if got := [2]string{row.UnitOutcome, row.UnitSource}; got != want[row.RunID] {
			t.Errorf("%s: got %v, want %v", row.RunID, got, want[row.RunID])
		}
	}
}
