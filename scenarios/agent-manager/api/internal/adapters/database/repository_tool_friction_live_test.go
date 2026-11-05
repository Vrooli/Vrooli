//go:build livecheck

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/opportunities"
	"agent-manager/internal/pricing"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageattribution"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TestLiveToolFrictionReview prints what `usage friction`, `usage
// opportunities` and `usage recommendations` say about the review findings
// (stale facts, unknown tokens, roll-up shares, degraded bases, ritual
// savings, polling advice, capped costs) over this host's last week. Tool
// facts are read from a snapshot of the live database's fact and transcript
// tables (TOOL_FRICTION_LIVE_SNAPSHOT, created read-only from the live
// database when absent), so two runs compare the same data; quota levers read
// the live database read-only. Opt in with TOOL_FRICTION_LIVE=1.
func TestLiveToolFrictionReview(t *testing.T) {
	if os.Getenv("TOOL_FRICTION_LIVE") != "1" {
		t.Skip("set TOOL_FRICTION_LIVE=1 to review the live tool friction read-only")
	}
	ctx := context.Background()
	live := filepath.Join("..", "..", "..", "..", "data", "agent-manager.db")
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	liveDB, err := sqlx.Connect("sqlite", "file:"+live+"?mode=ro&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer liveDB.Close()
	source := &DB{DB: liveDB, log: log}

	snapshot := os.Getenv("TOOL_FRICTION_LIVE_SNAPSHOT")
	if snapshot == "" {
		snapshot = filepath.Join(t.TempDir(), "facts.db")
	}
	now := time.Now().UTC()
	if stamp, err := os.ReadFile(snapshot + ".now"); err == nil {
		now, _ = time.Parse(time.RFC3339Nano, strings.TrimSpace(string(stamp)))
	}
	snapDB, err := sqlx.Connect("sqlite", "file:"+snapshot+"?_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer snapDB.Close()
	snap := &DB{DB: snapDB, log: log}
	var have int
	_ = snap.GetContext(ctx, &have, `SELECT COUNT(*) FROM invocation_read_model_facts`)
	if have == 0 {
		if err := snap.initSchema(); err != nil {
			t.Fatal(err)
		}
		if _, err := snap.ExecContext(ctx, `ATTACH DATABASE ? AS src`, "file:"+live+"?mode=ro"); err != nil {
			t.Fatal(err)
		}
		since := now.Add(-8 * 24 * time.Hour).Format(time.RFC3339Nano)
		for _, c := range []struct{ table, where string }{
			{"invocation_read_model_facts", "occurred_at >= '" + since + "'"},
			{"transcript_files", "run_id <> ''"},
			// The tool calls of runs that reached a scenario's internals: whose
			// source tree they work on decides whether that is a bypass.
			{"run_events", "event_type = 'tool_call' AND run_id IN (SELECT run_id FROM src.invocation_read_model_facts WHERE tool_bypass_owner <> '' AND occurred_at >= '" + since + "')"},
		} {
			copied, err := copyLiveTable(ctx, snap, c.table, c.where)
			if err != nil {
				t.Fatalf("copy %s: %v", c.table, err)
			}
			t.Logf("snapshot %s: %d rows", c.table, copied)
		}
		_ = os.WriteFile(snapshot+".now", []byte(now.Format(time.RFC3339Nano)), 0o600)
	}
	readings := NewPricingRepository(source, log).(pricing.QuotaReadingSource)
	pricer := usageattribution.NewPriceBook(pricing.NewServiceWithModelResolver(NewPricingRepository(source, log), nil, log, pricing.NewCLIModelResolver()))
	review := func(label string, facts *DB) {
		t.Logf("===== %s", label)
		tools := NewUsageAttributionRepository(facts).WithTranscriptParserVersions(func(string) int { return 1 })
		service := usageattribution.Service{Store: NewUsageAttributionRepository(source), Readings: readings, Pricer: pricer, Tools: tools, Library: liveLibrary{}}
		reviewToolFriction(t, ctx, service, tools, now)
	}
	review("as served: facts at the current fact version only", snap)
	// Rule check: the same snapshot with every fact relabelled current, so the
	// rules run over the whole week while re-derivation is still under way.
	// The relabelled copy is a scratch file; neither the live database nor the
	// snapshot is changed.
	if os.Getenv("TOOL_FRICTION_LIVE_RELABEL") == "1" {
		relabelled := snapshot + ".relabelled"
		_ = os.Remove(relabelled)
		if _, err := snap.ExecContext(ctx, `VACUUM INTO ?`, relabelled); err != nil {
			t.Fatal(err)
		}
		allDB, err := sqlx.Connect("sqlite", "file:"+relabelled+"?_pragma=busy_timeout(10000)")
		if err != nil {
			t.Fatal(err)
		}
		defer allDB.Close()
		if _, err := allDB.ExecContext(ctx, `UPDATE invocation_read_model_facts SET classifier_version = ?`, runsignal.InvocationFactVersion); err != nil {
			t.Fatal(err)
		}
		reextract(t, ctx, liveDB, allDB, log)
		review("rule check: every snapshot fact relabelled current (older facts keep older features)", &DB{DB: allDB, log: log})
		// The window as it was served before the fact version moved on: the
		// newest snapshot version read as current, older versions stale.
		if _, err := allDB.ExecContext(ctx, `ATTACH DATABASE ? AS snap`, "file:"+snapshot+"?mode=ro"); err != nil {
			t.Fatal(err)
		}
		if _, err := allDB.ExecContext(ctx, `UPDATE invocation_read_model_facts SET classifier_version = 'stale' WHERE EXISTS (SELECT 1 FROM snap.invocation_read_model_facts s WHERE s.run_id = invocation_read_model_facts.run_id AND s.call_event_id = invocation_read_model_facts.call_event_id AND s.segment_index = invocation_read_model_facts.segment_index AND s.classifier_version <> (SELECT MAX(classifier_version) FROM snap.invocation_read_model_facts))`); err != nil {
			t.Fatal(err)
		}
		review("as served at the snapshot's newest fact version (older versions stale)", &DB{DB: allDB, log: log})
	}
}

func reviewToolFriction(t *testing.T, ctx context.Context, service usageattribution.Service, tools *UsageAttributionRepository, now time.Time) {
	from := now.Add(-usageattribution.ToolFrictionWindow)
	window, err := tools.ToolCalls(ctx, from, now)
	if err != nil {
		t.Fatal(err)
	}
	report := toolfriction.Analyze(window)
	t.Logf("M1 friction coverage: %s", report.Coverage.Sentence())
	found := opportunities.Analyze(opportunities.Input{Calls: window.Calls, Stale: window.Stale, Library: liveLibrary{}.Library(ctx)})
	t.Logf("M1 opportunities coverage: %s", found.Coverage.Sentence())

	for i, f := range report.Findings {
		switch {
		case f.RootCause == "identity":
			t.Logf("M3 identity #%d: cost %d %s, rank share %.1f%% | %s", i+1, f.Cost, f.Unit, 100*f.CostShare, f.Description)
			for _, share := range f.Shares {
				t.Logf("M3 identity share: %s", share.Sentence())
			}
		case f.Kind == toolfriction.KindDegraded && (strings.Contains(f.ErrorLine, "park") || f.Owner == "search-hub"):
			t.Logf("M4 degraded #%d of %d findings: owner %s, %d calls in %d runs, %s | %s", i+1, len(report.Findings), f.Owner, f.Calls, f.Runs, f.Shares[0].Sentence(), f.ErrorLine)
		case f.Kind == toolfriction.KindOutputWeight && f.CommandPath == "prompt-manager skill read":
			t.Logf("M7 skill read #%d: cost %d tokens (uncapped tax in profile), top run %.0f%% | %s", i+1, f.Cost, 100*f.TopRunShare, f.Description)
		case f.Kind == toolfriction.KindPolling:
			t.Logf("M6 finding #%d: %s", i+1, f.Description)
		}
	}
	for _, p := range report.Profiles {
		if p.CommandPath == "prompt-manager skill read" {
			t.Logf("M7 skill read profile: %d calls, median %d p90 %d max %d tokens, context tax %d", p.Calls, p.MedianTokens, p.P90Tokens, p.MaxTokens, p.ContextTax)
		}
	}
	for _, o := range found.Opportunities {
		if o.Mandated {
			t.Logf("M5 ritual: %s | %d occurrences in %d sessions, %d calls saved, %d tokens (%d calls with unknown tokens)", o.Subject, o.Occurrences, o.Sessions, o.CallsSaved, o.TokensSaved, o.TokensUnknown)
		}
	}

	recs, _, err := service.Recommendations(ctx, "", usageattribution.WindowKey{}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range recs.Coverage {
		t.Logf("coverage: %s", line)
	}
	for _, rule := range recs.Rules {
		if rule.Rule == usageattribution.RuleSkill || strings.HasPrefix(rule.Rule, "R-") {
			t.Logf("rule %-10s fired=%v rows=%d candidates=%d %s", rule.Rule, rule.Fired, rule.Rows, rule.Candidates, rule.Reason)
		}
	}
	t.Logf("%d actions, %d under the materiality floor", len(recs.Actions), recs.Below)
	for _, f := range report.Findings {
		if f.Kind == toolfriction.KindDegraded && (strings.Contains(f.ErrorLine, "park") || f.Owner == "search-hub") {
			listed := "under the materiality floor"
			for _, a := range recs.Actions {
				if slices.Contains(a.Fingerprints, f.Fingerprint) {
					listed = fmt.Sprintf("action #%d", a.Rank)
				}
			}
			t.Logf("M4 recommendations: %s %.60q → %s", f.Owner, f.ErrorLine, listed)
		}
	}
	for _, a := range recs.Actions {
		switch {
		case a.Kind == usageattribution.RuleSkill:
			t.Logf("M2 skill #%d %s | %s", a.Rank, a.Title, a.Rationale)
		case a.Kind == toolfriction.KindPolling:
			t.Logf("M6 action #%d %s | owner %s %s | %s", a.Rank, a.Title, a.OwnerKind, a.Owner, a.Text)
		case a.RootCause == "identity":
			t.Logf("M3 action #%d %s | %s", a.Rank, a.Title, a.ScoreBasis)
		case a.Kind == toolfriction.KindDegraded:
			t.Logf("M4 action #%d %s | %s", a.Rank, a.Title, a.ScoreBasis)
		case a.Kind == toolfriction.KindShapeGuess && strings.Contains(a.Owner, "journal note"):
			t.Logf("help action #%d %s | %s", a.Rank, a.Title, a.Text)
		case a.Kind == opportunities.KindSequence:
			t.Logf("M5 action #%d %s | effect %.0f %s | %s", a.Rank, a.Title, a.Effect.Value, a.Effect.Unit, a.ScoreBasis)
		}
	}
	for i, a := range recs.Actions {
		if i >= 15 {
			break
		}
		t.Logf("top #%d %s [%s %s] %s | owner %s %s | effect %.0f %s", a.Rank, a.ScoreBasis, a.Source, a.Kind, a.Title, a.OwnerKind, a.Owner, a.Effect.Value, a.Effect.Unit)
	}
	reviewRound3(t, ctx, service, report, recs, now)
	reviewRound5(t, report, found, recs)
	reviewRound6(t, report, found, recs)
}

// reviewRound5 prints what review round 5 found: bypasses by runs developing
// the scenario they reach (2), skill reloads (3), polling of finished results
// (4), the ranking's scales (6) and the search-hub and denominator wording (7).
func reviewRound5(t *testing.T, report toolfriction.Report, found opportunities.Report, recs usageattribution.Report) {
	for i, f := range report.Findings {
		switch {
		case f.Kind == toolfriction.KindBypass && f.Owner == "agent-manager":
			t.Logf("R5-2 bypass finding #%d: %d calls in %d runs | %s", i+1, f.Calls, f.Runs, f.Description)
		case f.Kind == toolfriction.KindPolling && (strings.HasPrefix(f.CommandPath, "test-genie runs") || strings.HasSuffix(f.CommandPath, " status") || strings.HasSuffix(f.CommandPath, " get")):
			t.Logf("R5-4 polling #%d %s: %d calls in %d runs, score %.1f%% | %s", i+1, f.CommandPath, f.Calls, f.Runs, 100*f.Score, f.Description)
		case f.Kind == toolfriction.KindDegraded && f.Owner == "search-hub":
			t.Logf("R5-7 search-hub #%d: %s", i+1, f.Description)
		}
	}
	t.Logf("R5-7 totals: %d command calls, %d Vrooli CLI calls, %d decided, %d failed; friction coverage %d commands vs opportunities %d calls", report.Totals.Calls, report.Totals.VrooliCalls, report.Totals.VrooliDecided, report.Totals.VrooliFailed, report.Coverage.Commands, found.Coverage.Calls)
	t.Logf("R5-7 friction coverage: %s", report.Coverage.Sentence())
	t.Logf("R5-7 opportunities coverage: %s", found.Coverage.Sentence())
	for _, o := range found.Opportunities {
		switch o.Kind {
		case opportunities.KindBypass:
			if o.Owner == "agent-manager" {
				t.Logf("R5-2 bypass opportunity %s: %d occurrences in %d sessions, %d calls saved | %s", o.Subject, o.Occurrences, o.Sessions, o.CallsSaved, o.Action)
			}
		case opportunities.KindSkillReload:
			t.Logf("R5-3 skill reload %s: %d occurrences in %d sessions, %d calls saved, %d tokens | %s", o.Subject, o.Occurrences, o.Sessions, o.CallsSaved, o.TokensSaved, o.Action)
		case opportunities.KindSequence:
			t.Logf("R5-5 sequence %s [%s]: %d occurrences in %d sessions, runs %v", o.Subject, o.Verdict, o.Occurrences, o.Sessions, o.RunIDs)
		}
	}
	for i, a := range recs.Actions {
		if i >= 20 {
			break
		}
		t.Logf("R5-6 #%d [%s %s] %s | %s | effect %.0f %s", a.Rank, a.Source, a.Kind, a.Title, a.ScoreBasis, a.Effect.Value, a.Effect.Unit)
	}
}

// reviewRound3 prints what review round 3 found: the park cluster (W2), test
// verdicts counted as failures (W3), the friction list's unlisted findings
// (W13), the search-hub degraded finding (W15) and the journal note counts
// (W16).
func reviewRound3(t *testing.T, ctx context.Context, service usageattribution.Service, report toolfriction.Report, recs usageattribution.Report, now time.Time) {
	listed := func(fingerprint string) string {
		for _, a := range recs.Actions {
			if slices.Contains(a.Fingerprints, fingerprint) {
				return fmt.Sprintf("action #%d (%s)", a.Rank, a.ScoreBasis)
			}
		}
		return "not an action (under the materiality floor)"
	}
	for i, f := range report.Findings {
		switch {
		case strings.Contains(f.ErrorLine, "runner type is unavailable") && strings.Contains(f.ErrorLine, "running state"):
			t.Logf("W2 park #%d: owner %s, %d calls in %d runs, score %.1f%%, cost %d %s; %s → %s | %s", i+1, f.Owner, f.Calls, f.Runs, 100*f.Score, f.Cost, f.Unit, shareList(f.Shares), listed(f.Fingerprint), f.Description)
		case f.Kind == toolfriction.KindFailure && (strings.Contains(f.ErrorLine, "exit code") || strings.Contains(f.ErrorLine, "suite execution")) && (f.Material() || f.CommandPath == "" || strings.Contains(f.CommandPath, "test")):
			t.Logf("W3 failure #%d: owner %s path %q, %d calls in %d runs, score %.1f%%, %s → %s | %s", i+1, f.Owner, f.CommandPath, f.Calls, f.Runs, 100*f.Score, shareList(f.Shares), listed(f.Fingerprint), f.Description)
		case f.Kind == toolfriction.KindDegraded && f.Owner == "search-hub":
			t.Logf("W15 search-hub #%d: %d calls in %d runs, cost %d %s | %s", i+1, f.Calls, f.Runs, f.Cost, f.Unit, f.Description)
			t.Logf("W15 search-hub #%d line: %s", i+1, f.ErrorLine)
		case f.CommandPath == "vrooli-memory journal note":
			t.Logf("W16 journal finding #%d [%s]: %d calls in %d runs | %s", i+1, f.Kind, f.Calls, f.Runs, f.Description)
		}
	}
	for _, p := range report.Profiles {
		if p.CommandPath == "vrooli-memory journal note" {
			t.Logf("W16 journal profile: %d calls, %d help lookups, %d runs; window stale: %s", p.Calls, p.HelpLookups, p.Runs, report.Coverage.Stale.Sentence())
		}
	}
	friction, err := service.Friction(ctx, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	open, material := 0, 0
	for _, f := range friction.ToolFindings {
		if f.Status == usageattribution.StatusOpen {
			open++
		}
		if f.Material() {
			material++
		}
	}
	t.Logf("W13 friction: %d findings (%d open); %d at or over the floor (score %.0f%%), %d counted under it (below_floor); the CLI lists the first 20 of the material ones and says how many more", len(friction.ToolFindings), open, material, 100*toolfriction.MaterialScore, len(friction.ToolFindings)-material)
}

func shareList(shares []toolfriction.Share) string {
	parts := make([]string, len(shares))
	for i, s := range shares {
		parts[i] = s.Sentence()
	}
	return strings.Join(parts, "; ")
}

// reextract refreshes, in the scratch copy only, the stored semantics and tool
// features of the facts whose reading the current rules change (test commands
// a manifest now declares verify, and provider-degraded summaries) from the
// live events, the way projection extracts them. Other facts keep their
// stored features.
func reextract(t *testing.T, ctx context.Context, live, scratch *sqlx.DB, log *logrus.Logger) {
	var rows []struct {
		RunID       string        `db:"run_id"`
		CallEventID string        `db:"call_event_id"`
		ResultID    string        `db:"result_event_id"`
		Segment     int           `db:"segment_index"`
		Capability  string        `db:"capability"`
		Executable  string        `db:"executable"`
		CommandPath string        `db:"command_path"`
		Ownership   string        `db:"ownership"`
		Outcome     string        `db:"outcome"`
		ExitCode    sql.NullInt64 `db:"exit_code"`
		OutputBytes int64         `db:"output_bytes"`
	}
	if err := scratch.SelectContext(ctx, &rows, `SELECT run_id, call_event_id, COALESCE(result_event_id, '') AS result_event_id, segment_index, capability, COALESCE(executable, '') AS executable, command_path, ownership, outcome, exit_code, output_bytes
		FROM invocation_read_model_facts
		WHERE tool_profiled = 1 AND (command_path IN ('vrooli scenario test', 'vrooli package test', 'test-genie runs wait', 'test-genie runs wait-all', 'test-genie execute') OR tool_notice LIKE '%providers degraded%')
		ORDER BY run_id, occurred_at, call_event_id, segment_index`); err != nil {
		t.Fatal(err)
	}
	store := event.NewSQLiteStore(live, log)
	byRun := map[string][]int{}
	for i, row := range rows {
		byRun[row.RunID] = append(byRun[row.RunID], i)
	}
	updated := 0
	for runID, indexes := range byRun {
		id, err := uuid.Parse(runID)
		if err != nil {
			continue
		}
		events, err := store.Get(ctx, id, event.GetOptions{AfterSequence: -1})
		if err != nil {
			t.Fatal(err)
		}
		byID := map[string]*domain.RunEvent{}
		for _, e := range events {
			byID[e.ID.String()] = e
		}
		var sources []toolfriction.Source
		for _, i := range indexes {
			row := rows[i]
			fact := runsignal.InvocationFact{CallEventID: row.CallEventID, SegmentIndex: row.Segment, Capability: row.Capability, Executable: row.Executable, CommandPath: row.CommandPath, Ownership: row.Ownership, Outcome: row.Outcome, OutputBytes: row.OutputBytes, SemanticsKind: runsignal.ResolveCatalog(row.CommandPath).SemanticsKind}
			if row.ExitCode.Valid {
				code := int(row.ExitCode.Int64)
				fact.ExitCode = &code
			}
			source := toolfriction.Source{Fact: toolfriction.Fact{InvocationFact: fact, RunID: runID}}
			if e := byID[row.CallEventID]; e != nil {
				source.Call, _ = e.Data.(*domain.ToolCallEventData)
			}
			if e := byID[row.ResultID]; e != nil {
				source.Result, _ = e.Data.(*domain.ToolResultEventData)
			}
			sources = append(sources, source)
		}
		for j, f := range toolfriction.Extract(sources, nil) {
			s := sources[j].Fact
			if _, err := scratch.ExecContext(ctx, `UPDATE invocation_read_model_facts SET semantics_kind = ?, tool_signature = ?, tool_error_line = ?, tool_shape_guess = ?, tool_notice = ?, tool_notice_line = ? WHERE run_id = ? AND call_event_id = ? AND segment_index = ?`,
				s.SemanticsKind, f.Signature, f.ErrorLine, f.ShapeGuess, f.Notice, f.NoticeLine, runID, s.CallEventID, s.SegmentIndex); err != nil {
				t.Fatal(err)
			}
			updated++
		}
	}
	t.Logf("re-extracted %d facts of %d runs from the live events (test commands, degraded-provider summaries)", updated, len(byRun))
}
