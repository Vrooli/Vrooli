package outcomeledger

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func near(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.4f, want %.4f ±%.4f", name, got, want, tolerance)
	}
}

// synthetic builds n accepted units of one model: each consumes consumption
// units at price points per consumption unit, with a small spread so the range
// is not degenerate.
func synthetic(model string, n int, consumption, price float64) ([]Unit, []Run) {
	var units []Unit
	var runs []Run
	for i := 0; i < n; i++ {
		spread := 1 + 0.02*float64(i%5-2)
		id := fmt.Sprintf("%s-%d", model, i)
		c := consumption * spread
		units = append(units, Unit{ID: id, Key: model, Runner: "codex", Outcome: OutcomeAccepted, Source: SourceValidation, Path: PathHeadless, Attempts: 1, Points: c * price, Consumption: c, LinkConfidence: 1})
		runs = append(runs, Run{ID: id, UnitID: id, Path: PathHeadless, Key: model, Runner: "codex", Points: c * price, Consumption: c, ActiveMS: int64(consumption), WallMS: int64(consumption)})
	}
	return units, runs
}

func row(t *testing.T, report Report, key string) Row {
	t.Helper()
	for _, r := range report.Rows {
		if r.Key == key {
			return r
		}
	}
	t.Fatalf("no row %q", key)
	return Row{}
}

// A model 5x cheaper per consumption that uses 3x the consumption per outcome
// is cheaper by 5/3; at 10x it is more expensive.
func TestKnownAnswerCheaperModelWithMoreConsumption(t *testing.T) {
	baseUnits, baseRuns := synthetic("terra", 40, 1000, 0.005)
	for _, tc := range []struct {
		name    string
		factor  float64
		cheaper float64
		verdict string
	}{
		{"3x consumption is cheaper by 1.67x", 3, 5.0 / 3, VerdictCheaper},
		{"10x consumption is more expensive", 10, 5.0 / 10, VerdictMoreExpensive},
		{"6x consumption costs 1.2x, inside the margin", 6, 5.0 / 6, VerdictAboutSame},
	} {
		t.Run(tc.name, func(t *testing.T) {
			units, runs := synthetic("luna", 40, 1000*tc.factor, 0.001)
			report := Analyze(append(units, baseUnits...), append(runs, baseRuns...), Config{})
			luna := row(t, report, "luna")
			near(t, "cheaper by", luna.CheaperBy.Value, tc.cheaper, 0.01)
			near(t, "break-even", luna.BreakEvenRatio, 5, 0.01)
			near(t, "consumption ratio", luna.ConsumptionRatio.Value, tc.factor, 0.01)
			near(t, "run length ratio", luna.RunLengthRatio.Value, tc.factor, 0.1)
			if luna.Verdict != tc.verdict || luna.Baseline != "terra" {
				t.Fatalf("verdict %q baseline %q, want %q terra", luna.Verdict, luna.Baseline, tc.verdict)
			}
			if !(luna.CheaperBy.Low <= luna.CheaperBy.Value && luna.CheaperBy.Value <= luna.CheaperBy.High) {
				t.Fatalf("range does not hold the value: %+v", luna.CheaperBy)
			}
			if got := row(t, report, "terra").Verdict; got != VerdictBaseline {
				t.Fatalf("baseline verdict %q", got)
			}
		})
	}
}

func TestAnalyzeExcludesUnknownAndReportsIt(t *testing.T) {
	units, runs := synthetic("terra", 10, 1000, 0.005)
	for i := 0; i < 30; i++ {
		// A large unknown unit must move neither points per accepted nor the verdict.
		units = append(units, Unit{ID: fmt.Sprintf("u%d", i), Key: "terra", Runner: "codex", Outcome: OutcomeUnknown, Source: SourceNoSignal, Path: PathHeadless, Attempts: 1, Points: 1000, Consumption: 1e6})
	}
	report := Analyze(units, runs, Config{})
	terra := row(t, report, "terra")
	near(t, "points per accepted", terra.PointsPerAccepted.Value, 5, 0.01)
	if terra.Unknown != 30 || report.Unknown != 30 {
		t.Fatalf("unknown counts: row %d report %d", terra.Unknown, report.Unknown)
	}
	near(t, "unknown rate", terra.UnknownRate, 0.75, 0.001)
	if terra.PointsTotal <= terra.PointsKnown {
		t.Fatalf("unknown points should be reported apart: total %f known %f", terra.PointsTotal, terra.PointsKnown)
	}
}

func TestFailedAttemptsAndReworkCountInTheNumerator(t *testing.T) {
	units, runs := synthetic("terra", 10, 1000, 0.005)
	for i := 0; i < 10; i++ {
		units = append(units, Unit{ID: fmt.Sprintf("f%d", i), Key: "terra", Runner: "codex", Outcome: OutcomeFailed, Source: SourceRunStatus, Path: PathHeadless, Attempts: 2, Points: 5, Consumption: 1000})
	}
	terra := row(t, Analyze(units, runs, Config{}), "terra")
	near(t, "points per accepted", terra.PointsPerAccepted.Value, 10, 0.01)
	near(t, "rework", terra.ReworkRate.Value, 0.5, 0.001)
}

func TestLowDataReadsNotEnoughData(t *testing.T) {
	baseUnits, baseRuns := synthetic("terra", 40, 1000, 0.005)
	units, runs := synthetic("luna", 4, 3000, 0.001)
	luna := row(t, Analyze(append(units, baseUnits...), append(runs, baseRuns...), Config{}), "luna")
	if luna.Status != StatusNotEnough || luna.Verdict != StatusNotEnough || luna.Reason == "" {
		t.Fatalf("low data row: %+v", luna)
	}
	if luna.PointsPerAccepted.Available || luna.CheaperBy.Available || luna.PointsPerRun.Available || luna.BreakEvenRatio != 0 {
		t.Fatalf("low data row carries numbers: %+v", luna)
	}
	if luna.Units != 4 || luna.Accepted != 4 {
		t.Fatalf("low data row lost its counts: %+v", luna)
	}
}

func TestProvidersAreNeverCompared(t *testing.T) {
	codexUnits, codexRuns := synthetic("terra", 20, 1000, 0.005)
	claudeUnits, claudeRuns := synthetic("opus", 20, 1000, 0.05)
	for i := range claudeUnits {
		claudeUnits[i].Runner, claudeRuns[i].Runner = "claude-code", "claude-code"
	}
	report := Analyze(append(codexUnits, claudeUnits...), append(codexRuns, claudeRuns...), Config{})
	if row(t, report, "terra").Verdict != VerdictBaseline || row(t, report, "opus").Verdict != VerdictBaseline {
		t.Fatalf("a row was compared across providers: %+v", report.Rows)
	}
}

func TestLongParkedRunsAreExplained(t *testing.T) {
	var runs []Run
	for i := 0; i < 20; i++ {
		wall := int64(10 * 60 * 1000)
		if i < 2 {
			wall = 100 * 3600 * 1000
		}
		runs = append(runs, Run{ID: fmt.Sprint(i), Key: "luna", Runner: "codex", ActiveMS: 600000, WallMS: wall})
	}
	report := Analyze(nil, runs, Config{})
	luna := row(t, report, "luna")
	near(t, "mean", luna.WallHoursMean, 10.15, 0.01)
	near(t, "median", luna.WallHoursMedian, 1.0/6, 0.01)
	near(t, "active median", luna.ActiveMinutesPerRun.Value, 10, 0.01)
	if luna.RunsOver24h != 2 || len(report.Notes) != 1 {
		t.Fatalf("expected a note about 2 long runs: %+v %v", luna, report.Notes)
	}
}

func TestAnalyzeIsDeterministic(t *testing.T) {
	baseUnits, baseRuns := synthetic("terra", 40, 1000, 0.005)
	units, runs := synthetic("luna", 40, 3000, 0.001)
	a := row(t, Analyze(append(append([]Unit{}, units...), baseUnits...), append(append([]Run{}, runs...), baseRuns...), Config{}), "luna")
	b := row(t, Analyze(append(append([]Unit{}, baseUnits...), units...), append(append([]Run{}, baseRuns...), runs...), Config{}), "luna")
	if a.CheaperBy != b.CheaperBy || a.ReworkRate != b.ReworkRate {
		t.Fatalf("not deterministic: %+v vs %+v", a.CheaperBy, b.CheaperBy)
	}
}

func TestEstimateRules(t *testing.T) {
	imported := RunFacts{RunID: "i", Path: PathImported, Status: "complete", SessionStart: t0, Requests: 5}
	headless := RunFacts{RunID: "h", Path: PathHeadless, Status: "complete", SessionStart: t0, Requests: 5}
	interactive := RunFacts{RunID: "a", Path: PathInteractive, Status: "complete", SessionStart: t0, Requests: 5}
	with := func(run RunFacts, change func(*RunFacts)) RunFacts { change(&run); return run }
	failedFirst := with(headless, func(r *RunFacts) { r.RunID, r.Status, r.SessionStart = "h0", "failed", t0.Add(-time.Hour) })
	for _, tc := range []struct {
		name        string
		runs        []RunFacts
		corrections []Correction
		want, from  string
	}{
		{"imported complete is never accepted", []RunFacts{imported}, nil, OutcomeUnknown, SourceNoSignal},
		{"imported web console complete is never accepted", []RunFacts{with(imported, func(r *RunFacts) { r.Path = PathImportedWebConsole })}, nil, OutcomeUnknown, SourceNoSignal},
		{"live is unknown even with validation", []RunFacts{with(imported, func(r *RunFacts) { r.Live, r.LastValidation = true, "success" })}, nil, OutcomeUnknown, SourceInProgress},
		{"approved", []RunFacts{with(headless, func(r *RunFacts) { r.ApprovalState = "approved" })}, nil, OutcomeAccepted, SourceOperatorApproval},
		{"rejected", []RunFacts{with(headless, func(r *RunFacts) { r.ApprovalState = "rejected" })}, nil, OutcomeFailed, SourceOperatorRejection},
		{"operator correction outranks status", []RunFacts{headless}, []Correction{{Category: "quality_miss", Probability: 0.9}}, OutcomeCorrected, SourceOperatorCorrection},
		{"approval message is no evidence", []RunFacts{imported}, []Correction{{Category: "approval_or_info", Probability: 0.99}}, OutcomeUnknown, SourceNoSignal},
		{"unlikely correction is no evidence", []RunFacts{imported}, []Correction{{Category: "wrong_claim", Probability: 0.3}}, OutcomeUnknown, SourceNoSignal},
		{"final reported failure ends the unit failed", []RunFacts{imported}, []Correction{{RunID: "i", Signal: SignalReportedFailure, Probability: 0.9, Final: true, Precision: 0.85}}, OutcomeFailed, SourceOperatorReportedFailure},
		{"earlier reported failure means corrected", []RunFacts{imported}, []Correction{{RunID: "i", Signal: SignalReportedFailure, Probability: 0.9, Precision: 0.85}}, OutcomeCorrected, SourceOperatorReportedFailure},
		{"final failure in an earlier attempt is only a correction", []RunFacts{failedFirst, imported}, []Correction{{RunID: "h0", Signal: SignalReportedFailure, Probability: 0.9, Final: true, Precision: 0.85}}, OutcomeCorrected, SourceOperatorReportedFailure},
		{"reported failure outranks a category correction", []RunFacts{imported}, []Correction{{Category: "quality_miss", Probability: 0.9}, {RunID: "i", Signal: SignalReportedFailure, Probability: 0.9, Precision: 0.85}}, OutcomeCorrected, SourceOperatorReportedFailure},
		{"signal correction", []RunFacts{imported}, []Correction{{RunID: "i", Signal: SignalCorrection, Probability: 0.8, Precision: 0.8}}, OutcomeCorrected, SourceOperatorSignalCorrection},
		{"unmeasured signal is no evidence", []RunFacts{imported}, []Correction{{RunID: "i", Signal: SignalReportedFailure, Probability: 0.99, Final: true}}, OutcomeUnknown, SourceNoSignal},
		{"signal measured below even odds is no evidence", []RunFacts{imported}, []Correction{{RunID: "i", Signal: SignalReportedFailure, Probability: 0.9, Precision: 0.4}}, OutcomeUnknown, SourceNoSignal},
		{"a signal read at its measured operating point counts below even odds", []RunFacts{imported}, []Correction{{RunID: "i", Signal: SignalCorrection, Probability: 0.2, Precision: 0.64}}, OutcomeCorrected, SourceOperatorSignalCorrection},
		{"approval outranks a reported failure", []RunFacts{with(headless, func(r *RunFacts) { r.ApprovalState = "approved" })}, []Correction{{RunID: "h", Signal: SignalReportedFailure, Probability: 0.9, Final: true, Precision: 0.9}}, OutcomeAccepted, SourceOperatorApproval},
		{"failed managed", []RunFacts{with(headless, func(r *RunFacts) { r.Status = "failed" })}, nil, OutcomeFailed, SourceRunStatus},
		{"cancelled", []RunFacts{with(headless, func(r *RunFacts) { r.Status = "cancelled" })}, nil, OutcomeAbandoned, SourceRunStatus},
		{"crashed import", []RunFacts{with(imported, func(r *RunFacts) { r.TerminalClass, r.StopReason = "interruption", "crash" })}, nil, OutcomeAbandoned, SourceRunStatus},
		{"retry after failure", []RunFacts{failedFirst, headless}, nil, OutcomeCorrected, SourceRetry},
		{"blocked verdict", []RunFacts{with(interactive, func(r *RunFacts) { r.TerminalClass, r.StopReason = "verdict", "blocked" })}, nil, OutcomeFailed, SourceAgentVerdict},
		{"complete verdict", []RunFacts{with(interactive, func(r *RunFacts) { r.TerminalClass, r.StopReason = "verdict", "complete" })}, nil, OutcomeAccepted, SourceAgentVerdict},
		{"interactive complete alone is unknown", []RunFacts{interactive}, nil, OutcomeUnknown, SourceNoSignal},
		{"last validation failed", []RunFacts{with(imported, func(r *RunFacts) { r.LastValidation = "failure" })}, nil, OutcomeFailed, SourceValidation},
		{"headless complete alone is completed, not accepted", []RunFacts{headless}, nil, OutcomeCompleted, SourceHeadlessStatus},
		{"sandbox auto-approval at run end is no acceptance", []RunFacts{with(headless, func(r *RunFacts) { r.ApprovalState = ApprovalAutomatic })}, nil, OutcomeCompleted, SourceHeadlessStatus},
		{"a failure after the last pass is failed", []RunFacts{with(headless, func(r *RunFacts) { r.LastValidation, r.LastValidationAt = "success", t0 }), with(headless, func(r *RunFacts) {
			r.RunID, r.SessionStart, r.LastValidation, r.LastValidationAt = "h2", t0.Add(time.Hour), "failure", t0.Add(time.Hour)
		})}, nil, OutcomeFailed, SourceValidation},
		{"a pass after an earlier failure is accepted", []RunFacts{with(headless, func(r *RunFacts) { r.LastValidation, r.LastValidationAt = "failure", t0 }), with(headless, func(r *RunFacts) {
			r.RunID, r.SessionStart, r.LastValidation, r.LastValidationAt = "h2", t0.Add(time.Hour), "success", t0.Add(time.Hour)
		})}, nil, OutcomeAccepted, SourceValidation},
		{"a passing validation makes headless work accepted", []RunFacts{with(headless, func(r *RunFacts) { r.LastValidation = "success" })}, nil, OutcomeAccepted, SourceValidation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateUnit(tc.runs, tc.corrections)
			if got.Outcome != tc.want || got.Source != tc.from {
				t.Fatalf("got %+v, want %s from %s", got, tc.want, tc.from)
			}
		})
	}
	// The same transcript facts get the same estimate on any path.
	validated := func(path string) Estimate {
		return EstimateUnit([]RunFacts{{RunID: "x", Path: path, Status: "complete", SessionStart: t0, LastValidation: "success", LastValidationAt: t0}}, nil)
	}
	if validated(PathImported) != validated(PathInteractive) || validated(PathImported).Outcome != OutcomeAccepted {
		t.Fatalf("paths disagree: %+v vs %+v", validated(PathImported), validated(PathInteractive))
	}
}

func TestLinkWorkUnits(t *testing.T) {
	at := func(minutes int) time.Time { return t0.Add(time.Duration(minutes) * time.Minute) }
	runs := []RunFacts{
		{RunID: "parent", SessionStart: at(0), Path: PathImported},
		{RunID: "child", ParentRunID: "parent", SessionStart: at(1), Path: PathImported},
		{RunID: "resume-a", SessionKey: "codex:s1", SessionStart: at(2), Path: PathImported},
		{RunID: "resume-b", SessionKey: "codex:s1", SessionStart: at(3), Path: PathImported},
		{RunID: "retry-a", WorkloadKey: "adhoc:k", Status: "failed", SessionStart: at(4), Path: PathHeadless},
		{RunID: "retry-b", WorkloadKey: "adhoc:k", Status: "complete", SessionStart: at(5), Path: PathHeadless},
		{RunID: "same-task-a", TaskID: "t", SessionStart: at(6), Path: PathHeadless},
		{RunID: "same-task-b", TaskID: "t", SessionStart: at(7), Path: PathHeadless},
		{RunID: "later-a", WorkloadKey: "adhoc:far", Status: "failed", SessionStart: at(8), Path: PathHeadless},
		{RunID: "later-b", WorkloadKey: "adhoc:far", SessionStart: at(8 + 3*24*60), Path: PathHeadless},
		{RunID: "source", SessionStart: at(9), Path: PathHeadless},
		{RunID: "successor", SourceRunIDs: []string{"source"}, SessionStart: at(10), Path: PathHeadless},
	}
	for i := 0; i < 20; i++ {
		runs = append(runs, RunFacts{RunID: fmt.Sprintf("draft-%d", i), WorkloadKey: "workflow_node:draft", Status: "complete", SessionStart: at(20 + i), Path: PathHeadless})
	}
	got := LinkWorkUnits(runs)
	same := func(a, b, kind string, confidence float64) {
		t.Helper()
		if got[a].WorkUnitID != got[b].WorkUnitID {
			t.Fatalf("%s and %s not linked: %+v %+v", a, b, got[a], got[b])
		}
		if got[b].LinkKind != kind || got[b].LinkConfidence != confidence {
			t.Fatalf("%s link: %+v, want %s %.2f", b, got[b], kind, confidence)
		}
	}
	same("parent", "child", LinkChild, 1)
	same("resume-a", "resume-b", LinkResume, 0.95)
	same("retry-a", "retry-b", LinkWorkloadRetry, 0.8)
	same("same-task-a", "same-task-b", LinkTask, 0.9)
	same("source", "successor", LinkSuccessor, 0.95)
	if got["later-a"].WorkUnitID == got["later-b"].WorkUnitID {
		t.Fatal("a workload key linked runs three days apart")
	}
	if got["draft-0"].WorkUnitID == got["draft-1"].WorkUnitID {
		t.Fatal("a key shared by many runs names a kind of work, not one task")
	}
	if got["parent"].LinkKind != LinkRoot || got["parent"].LinkConfidence != 1 {
		t.Fatalf("root: %+v", got["parent"])
	}
}

func TestClassifyAttached(t *testing.T) {
	imported := []RunFacts{
		{RunID: "wc", Harness: "codex", Path: PathImportedWebConsole, WebConsoleSession: "w1", SessionStart: t0.Add(time.Hour)},
		{RunID: "term", Harness: "codex", Path: PathImported, SessionStart: t0.Add(20 * time.Second)},
		{RunID: "sub", Harness: "codex", Path: PathImported, ParentRunID: "term", SessionStart: t0.Add(25 * time.Second)},
	}
	attached := []RunFacts{
		{RunID: "a-wc", Harness: "codex", Path: PathAttached, WebConsoleSession: "w1", SessionStart: t0.Add(-time.Hour), LastActivity: t0.Add(-time.Hour)},
		{RunID: "a-time", Harness: "codex", Path: PathAttached, SessionStart: t0, LastActivity: t0.Add(30 * time.Second)},
		{RunID: "a-helper", Harness: "codex", Path: PathAttached, SessionStart: t0.Add(3 * time.Hour), LastActivity: t0.Add(3*time.Hour + time.Minute)},
		{RunID: "a-lost", Harness: "codex", Path: PathAttached, SessionStart: t0.Add(5 * time.Hour), LastActivity: t0.Add(6 * time.Hour)},
		{RunID: "a-grok", Harness: "grok", Path: PathAttached, SessionStart: t0.Add(5 * time.Hour), LastActivity: t0.Add(6 * time.Hour)},
		{RunID: "a-grok-short", Harness: "grok", Path: PathAttached, SessionStart: t0.Add(5 * time.Hour), LastActivity: t0.Add(5*time.Hour + time.Minute)},
		{RunID: "a-long", Harness: "claude-code", Path: PathAttached, SessionStart: t0, LastActivity: t0.Add(2 * time.Hour)},
	}
	imported = append(imported, RunFacts{RunID: "late", Harness: "claude-code", Path: PathImported, SessionStart: t0.Add(time.Hour)})
	got := ClassifyAttached(attached, imported)
	for id, want := range map[string]string{"a-wc": AttachedLinked, "a-time": AttachedLinked, "a-helper": AttachedHelper, "a-lost": AttachedUnlinked, "a-grok": AttachedNoReader, "a-grok-short": AttachedHelper, "a-long": AttachedLinked} {
		if got[id].AttachedClass != want {
			t.Fatalf("%s: %+v, want %s", id, got[id], want)
		}
	}
	if got["a-long"].LinkedRunID != "late" || got["a-long"].LinkConfidence != 0.4 {
		t.Fatalf("session started during the attached row: %+v", got["a-long"])
	}
	if got["a-wc"].LinkedRunID != "wc" || got["a-time"].LinkedRunID != "term" || got["a-wc"].LinkConfidence != 0.95 || got["a-time"].LinkConfidence != 0.6 {
		t.Fatalf("links: %+v %+v", got["a-wc"], got["a-time"])
	}
}

func TestParityDistinguishesNoCommandsFromMissing(t *testing.T) {
	base := RunFacts{Path: PathImported, Harness: "codex", Requests: 5, ActualModel: "gpt", ModeRelation: ModeNone, FileFullyRead: true, FileStatus: "indexed", OperatorMessages: 1, UnitOutcome: OutcomeUnknown, UnitSource: SourceNoSignal}
	rows := []RunFacts{
		func() RunFacts {
			r := base
			r.RunID, r.TranscriptToolCalls, r.FactRows, r.Commands, r.KnownCommands = "has", 5, 5, 4, 3
			return r
		}(),
		func() RunFacts { r := base; r.RunID, r.TranscriptToolCalls, r.FactRows = "no-commands", 0, 0; return r }(),
		func() RunFacts {
			r := base
			r.RunID, r.TranscriptToolCalls, r.RecallEvicted = "missing", 3, true
			return r
		}(),
		func() RunFacts { r := base; r.RunID, r.Requests = "empty", 0; return r }(),
		func() RunFacts {
			r := base
			r.RunID, r.Requests, r.FileFullyRead, r.FileStatus = "stuck", 0, false, "unreadable"
			return r
		}(),
		func() RunFacts {
			r := base
			r.RunID, r.OlderParser, r.Commands, r.KnownCommands, r.TranscriptToolCalls, r.FactRows = "old", true, 10, 0, 10, 10
			return r
		}(),
		{RunID: "h", Path: PathAttached, Harness: "codex", AttachedClass: AttachedHelper},
		{RunID: "l", Path: PathAttached, Harness: "codex", AttachedClass: AttachedLinked},
		{RunID: "g", Path: PathAttached, Harness: "grok", AttachedClass: AttachedNoReader},
	}
	report := BuildParity(rows)
	get := func(path, harness, name string) ParityMeasure {
		for _, row := range report {
			if row.Path == path && row.Harness == harness {
				for _, m := range row.Measures {
					if m.Name == name {
						return m
					}
				}
			}
		}
		t.Fatalf("no %s %s %s", path, harness, name)
		return ParityMeasure{}
	}
	tools := get(PathImported, "codex", MeasureToolFacts)
	if tools.Numerator != 2 || tools.Denominator != 3 || tools.NotApplicable != 1 || tools.Reason == "" {
		t.Fatalf("tool facts: %+v", tools)
	}
	usage := get(PathImported, "codex", MeasureUsage)
	if usage.Numerator != 4 || usage.Denominator != 5 || usage.NotApplicable != 1 {
		t.Fatalf("an unread transcript must count as missing and an empty one as not applicable: %+v", usage)
	}
	outcomes := get(PathImported, "codex", MeasureOutcomes)
	if outcomes.Numerator != 3 || outcomes.Denominator != 4 || outcomes.NotApplicable != 10 || outcomes.Reason == "" {
		t.Fatalf("older-parser commands stay out of the share and in the reason: %+v", outcomes)
	}
	if estimate := get(PathImported, "codex", MeasureEstimate); estimate.Numerator != 0 || estimate.Reason == "" {
		t.Fatalf("estimate below 90%% needs a reason: %+v", estimate)
	}
	if attached := get(PathAttached, "codex", MeasureUsage); attached.Denominator != 0 {
		t.Fatalf("helpers and duplicates entered denominators: %+v", attached)
	}
	if grok := get(PathAttached, "grok", MeasureUsage); grok.Denominator != 1 || grok.Reason == "" {
		t.Fatalf("an unreadable harness is a stated gap: %+v", grok)
	}
}

func TestResolveMode(t *testing.T) {
	period := ModePeriod{ID: "um", Runner: "codex", Targets: []string{"gpt-6-luna"}, StartedAt: t0, ExpiresAt: t0.Add(2 * time.Hour)}
	run := func(start time.Time, model string) RunFacts {
		return RunFacts{Harness: "codex", ActualModel: model, FirstActivity: start, LastActivity: start.Add(time.Minute)}
	}
	for _, tc := range []struct {
		name string
		run  RunFacts
		want string
		conf float64
	}{
		{"exact from selection reason", RunFacts{SelectionReason: "role;usage_mode:um-1", Harness: "codex"}, ModeApplied, 1},
		{"skipped is exact", RunFacts{SelectionReason: "usage_mode_skipped:explicit_model", Harness: "codex"}, ModeSkipped, 1},
		{"in period on target", run(t0.Add(time.Hour), "gpt-6-luna"), ModeOnTarget, inferredConfidence},
		{"in period not changed", run(t0.Add(time.Hour), "gpt-6.1-terra"), ModeNotChanged, inferredConfidence},
		{"started before and still running", RunFacts{Harness: "codex", ActualModel: "gpt-6.1-terra", FirstActivity: t0.Add(-time.Hour), LastActivity: t0.Add(time.Hour)}, ModePredates, 1},
		{"before the period", run(t0.Add(-time.Hour), "gpt-6-luna"), ModeNone, 1},
		{"after the period", run(t0.Add(3*time.Hour), "gpt-6-luna"), ModeNone, 1},
		{"other runner", RunFacts{Harness: "claude-code", ActualModel: "gpt-6-luna", FirstActivity: t0.Add(time.Hour), LastActivity: t0.Add(time.Hour)}, ModeNone, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveMode(tc.run, []ModePeriod{period})
			if got.Relation != tc.want || got.Confidence != tc.conf {
				t.Fatalf("got %+v, want %s %.2f", got, tc.want, tc.conf)
			}
		})
	}
}

func TestActiveTimeExcludesLongGaps(t *testing.T) {
	reports := []time.Time{t0, t0.Add(time.Minute), t0.Add(2 * time.Minute), t0.Add(10 * time.Hour), t0.Add(10*time.Hour + 30*time.Second)}
	wall, active := ActiveTime(reports)
	if wall != 10*time.Hour+30*time.Second || active != 2*time.Minute+30*time.Second {
		t.Fatalf("wall %s active %s", wall, active)
	}
	if wall, active := ActiveTime([]time.Time{t0}); wall != 0 || active != 0 {
		t.Fatalf("single report: %s %s", wall, active)
	}
}

func TestSummarizeCalls(t *testing.T) {
	summary := SummarizeCalls([]CallFact{
		{CallID: "1", Tool: "exec", Command: "go test ./...", Outcome: "failure", At: t0},
		{CallID: "1", Tool: "exec", Command: "tail", Outcome: "unknown", At: t0},
		{CallID: "2", Tool: "Bash", Command: "go test ./...", Outcome: "success", At: t0.Add(time.Minute)},
		{CallID: "3", Tool: "exec", Command: "ls", Outcome: "unknown", At: t0},
		{CallID: "4", Tool: "exec", Command: "go test --help", Outcome: "failure", At: t0.Add(time.Hour)},
		{CallID: "5", Tool: "send_message", Outcome: "unknown", At: t0},
		{CallID: "6", Tool: "exec", Command: "vrooli scenario test x", Help: true, Outcome: "failure", At: t0.Add(2 * time.Hour)},
		{CallID: "7", Tool: "exec", Basis: "call-id", Outcome: "unknown", At: t0},
	}, true)
	if summary.Rows != 8 || summary.Commands != 5 || summary.Known != 4 {
		t.Fatalf("counts: %+v", summary)
	}
	// An older parser reports no nested commands, so its exec is the only record.
	if older := SummarizeCalls([]CallFact{{CallID: "7", Tool: "exec", Basis: "call-id", Outcome: "unknown", At: t0}}, false); older.Commands != 1 || older.Known != 0 {
		t.Fatalf("older parser: %+v", older)
	}
	if summary.ValidationsPassed != 1 || summary.ValidationsFailed != 1 || summary.LastValidation != "success" {
		t.Fatalf("validation: %+v", summary)
	}
}

func TestAssembleKeepsUnitTotalsAndRunKeys(t *testing.T) {
	runs := []RunFacts{
		{RunID: "root", WorkUnitID: "root", Harness: "claude-code", Path: PathImported, Requests: 3, ActualModel: "opus", SessionStart: t0, UnitOutcome: OutcomeAccepted, UnitSource: SourceValidation, LinkConfidence: 1},
		{RunID: "sub", WorkUnitID: "root", ParentRunID: "root", Harness: "claude-code", Path: PathImported, Requests: 2, ActualModel: "sonnet", SessionStart: t0, UnitOutcome: OutcomeAccepted, UnitSource: SourceValidation, LinkConfidence: 1},
		{RunID: "attached", WorkUnitID: "", Harness: "codex", Path: PathAttached},
		{RunID: "empty", WorkUnitID: "empty", Harness: "codex", Path: PathImported},
	}
	use := map[string]map[string]Use{
		"root": {"opus": {Points: 6, Consumption: 100}},
		"sub":  {"sonnet": {Points: 2, Consumption: 300}},
	}
	units, samples := Assemble(runs, use, ByModel)
	if len(units) != 1 || units[0].Key != "opus" || units[0].Points != 8 || units[0].Consumption != 400 || units[0].Attempts != 1 {
		t.Fatalf("units: %+v", units)
	}
	if len(samples) != 2 {
		t.Fatalf("samples: %+v", samples)
	}
	keys := map[string]string{}
	for _, s := range samples {
		keys[s.ID] = s.Key
	}
	if keys["root"] != "opus" || keys["sub"] != "sonnet" {
		t.Fatalf("a sub-agent counts under its own model: %v", keys)
	}
	if units, _ := Assemble(runs, use, ByRole); units[0].Key != "operator_session" {
		t.Fatalf("role: %+v", units)
	}
}

func TestDeriveLinksEstimatesAndReportsOnlyChanges(t *testing.T) {
	rows := []RunFacts{
		{RunID: "a", Path: PathHeadless, Harness: "codex", Status: "failed", WorkloadKey: "adhoc:k", Requests: 1, SessionStart: t0},
		{RunID: "b", Path: PathHeadless, Harness: "codex", Status: "complete", WorkloadKey: "adhoc:k", Requests: 1, SessionStart: t0.Add(time.Hour)},
		{RunID: "i", Path: PathImported, Harness: "codex", Status: "complete", Requests: 1, SessionStart: t0},
	}
	changed := Derive(rows, nil, nil)
	if len(changed) != 3 || rows[0].WorkUnitID != rows[1].WorkUnitID || rows[1].UnitOutcome != OutcomeCorrected || rows[1].UnitSource != SourceRetry || rows[2].UnitOutcome != OutcomeUnknown {
		t.Fatalf("derive: %+v", rows)
	}
	if again := Derive(rows, nil, nil); len(again) != 0 {
		t.Fatalf("a second pass changed %d rows", len(again))
	}
}

// Completed work has no evidence of acceptance. However much of it there is, it
// passes no acceptance gate and reads only as its own weaker unit.
func TestCompletedNeverPassesAcceptanceGates(t *testing.T) {
	units, runs := synthetic("terra", 40, 1000, 0.005)
	for i := range units {
		units[i].Outcome, units[i].Source = OutcomeCompleted, SourceHeadlessStatus
	}
	report := Analyze(units, runs, Config{})
	terra := row(t, report, "terra")
	if terra.Accepted != 0 || terra.Completed != 40 || report.Completed != 40 || terra.Status != StatusNotEnough || terra.Verdict != StatusNotEnough || terra.PointsPerAccepted.Available {
		t.Fatalf("completed passed an acceptance gate: %+v", terra)
	}
	near(t, "points per completed", terra.PointsPerCompleted.Value, 5, 0.01)
	// A completed unit's spend is still spend: it stays in points per accepted.
	for i := 0; i < 5; i++ {
		units[i].Outcome, units[i].Source = OutcomeAccepted, SourceValidation
	}
	mixed := row(t, Analyze(units, runs, Config{}), "terra")
	near(t, "all spend over 5 accepted", mixed.PointsPerAccepted.Value, 40.0*5/5, 0.3)
	near(t, "all spend over 40 finished", mixed.PointsPerCompleted.Value, 5, 0.01)
}

// A comparison never mixes paths: an operator can only correct work they watch,
// so the same model reads very differently by path for reasons unrelated to it.
func TestModelRowsAreStratifiedByPathAndComparedOnlyWithinIt(t *testing.T) {
	headless, headlessRuns := synthetic("sol", 20, 1000, 0.005)
	operator, operatorRuns := synthetic("sol", 15, 1000, 0.005)
	for i := range operator {
		operator[i].ID, operator[i].Path = fmt.Sprintf("op-%d", i), PathImported
		operatorRuns[i].ID, operatorRuns[i].UnitID, operatorRuns[i].Path = operator[i].ID, operator[i].ID, PathImported
		if i < 12 {
			operator[i].Outcome = OutcomeCorrected
		}
	}
	other, otherRuns := synthetic("luna", 20, 3000, 0.001)
	report := Analyze(append(append(headless, operator...), other...), append(append(headlessRuns, operatorRuns...), otherRuns...), Config{})
	var solHeadless, solOperator Row
	for _, r := range report.Rows {
		switch {
		case r.Key == "sol" && r.Stratum == StratumHeadless:
			solHeadless = r
		case r.Key == "sol" && r.Stratum == StratumOperator:
			solOperator = r
		}
	}
	if solHeadless.Corrected != 0 || solHeadless.Units != 20 || solOperator.Corrected != 12 || solOperator.Units != 15 {
		t.Fatalf("rows pooled paths: %+v / %+v", solHeadless, solOperator)
	}
	if solOperator.Baseline != "" && solOperator.Baseline != "sol" {
		t.Fatalf("the operator row was compared across paths: %+v", solOperator)
	}
	if solHeadless.Baseline == "" || row(t, report, "luna").Baseline != solHeadless.Key {
		t.Fatalf("headless rows compare with each other: %+v", report.Rows)
	}
}

func TestAboutTheSameNeedsTheRangeInsideTheMargin(t *testing.T) {
	for _, tc := range []struct {
		name             string
		value, low, high float64
		want             string
	}{
		{"the reported case: 0.97x, 0.13x to 49.50x, says nothing", 0.97, 0.13, 49.5, VerdictInconclusive},
		{"inside plus or minus 30 percent", 1.02, 0.85, 1.2, VerdictAboutSame},
		{"crossing 1 but past the margin on one side", 1.1, 0.9, 1.6, VerdictInconclusive},
		{"clearly cheaper", 3, 2, 4, VerdictCheaper},
		{"clearly dearer", 0.4, 0.3, 0.55, VerdictMoreExpensive},
		{"excludes 1 but inside the margin is about the same", 1.15, 1.05, 1.25, VerdictAboutSame},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readRange(Range{Available: true, Value: tc.value, Low: tc.low, High: tc.high}); got != tc.want {
				t.Fatalf("%s, want %s", got, tc.want)
			}
		})
	}
	// End to end: a heavy-tailed pair with no true difference.
	rng := rand.New(rand.NewSource(3))
	build := func(model string) []Unit {
		var units []Unit
		for i := 0; i < 8; i++ {
			c := 1000 * math.Exp(2*rng.NormFloat64())
			units = append(units, Unit{ID: fmt.Sprintf("%s-%d", model, i), Key: model, Runner: "codex", Outcome: OutcomeAccepted, Source: SourceValidation, Path: PathHeadless, Attempts: 1, Points: c * 0.005, Consumption: c})
		}
		return units
	}
	report := Analyze(append(build("a"), build("b")...), nil, Config{})
	for _, r := range report.Rows {
		if r.Verdict == VerdictAboutSame {
			t.Fatalf("a wide range read about the same: %+v", r.CheaperBy)
		}
	}
}

// The price advantage and the extra use are read on one unit set with one set of
// weights, so they multiply back to the cost ratio exactly.
func TestBreakEvenKnownAnswerOnTheSameUnitsAndWeights(t *testing.T) {
	// Terra: 10 accepted, 1,000 tokens each at 0.005 a token. Luna: 10 accepted, 3,000
	// tokens at 0.001, and 10 unknown units at 0.05 a token that must move nothing.
	unit := func(id, key, outcome string, c, price float64) Unit {
		return Unit{ID: id, Key: key, Runner: "codex", Outcome: outcome, Source: SourceValidation, Path: PathHeadless, Attempts: 1, Points: c * price, Consumption: c}
	}
	var units []Unit
	for i := 0; i < 10; i++ {
		units = append(units, unit(fmt.Sprintf("t%d", i), "terra", OutcomeAccepted, 1000, 0.005), unit(fmt.Sprintf("l%d", i), "luna", OutcomeAccepted, 3000, 0.001), unit(fmt.Sprintf("u%d", i), "luna", OutcomeUnknown, 500, 0.05))
	}
	report := Analyze(units, nil, Config{
		PriceSources: map[string]string{"terra": "price_book", "luna": "tier_prior"},
		ModelPrices: map[string]ModelPrice{
			"terra": {Input: 10, CacheRead: 1, CacheCreation: 12.5, Output: 40, Source: "price_book"},
			"luna":  {Input: 2, CacheRead: 0.2, CacheCreation: 2.5, Output: 8, Source: "tier_prior"},
		},
		ReferenceMix: map[string]TokenMix{"codex": {Input: 100, CacheRead: 1000, Output: 50}},
	})
	luna := row(t, report, "luna")
	near(t, "realized price ratio", luna.BreakEvenRatio, 5, 1e-9)
	near(t, "extra use", luna.ConsumptionRatio.Value, 3, 1e-9)
	near(t, "cheaper by", luna.CheaperBy.Value, 5.0/3, 1e-9)
	near(t, "identity", luna.CheaperBy.Value, luna.BreakEvenRatio/luna.ConsumptionRatio.Value, 1e-9)
	// Model price is the price book at one mix, apart from what either model realized.
	near(t, "model price ratio", luna.ModelPriceRatio, 5, 1e-9)
	if luna.ModelPriceSource != "mixed" || luna.PriceSource != "mixed" {
		t.Fatalf("a prior on either side must show: %q %q", luna.ModelPriceSource, luna.PriceSource)
	}
	// A realized mix differs from the model price and both are reported.
	for i := range units {
		if units[i].Key == "luna" && units[i].Outcome == OutcomeAccepted {
			units[i].Points *= 2 // heavier output than the reference mix
		}
	}
	shifted := row(t, Analyze(units, nil, Config{ModelPrices: map[string]ModelPrice{
		"terra": {Input: 10, Source: "price_book"}, "luna": {Input: 2, Source: "price_book"},
	}, ReferenceMix: map[string]TokenMix{"codex": {Input: 100}}}), "luna")
	near(t, "realized moved", shifted.BreakEvenRatio, 2.5, 1e-9)
	near(t, "price stayed", shifted.ModelPriceRatio, 5, 1e-9)
	near(t, "identity after the shift", shifted.CheaperBy.Value, shifted.BreakEvenRatio/shifted.ConsumptionRatio.Value, 1e-9)
}

func TestLowConfidenceLinksDoNotChainOrCrossTasks(t *testing.T) {
	hour := func(n float64) time.Time { return t0.Add(time.Duration(n * float64(time.Hour))) }
	console := func(id string, at float64, goal string) RunFacts {
		return RunFacts{RunID: id, WebConsoleSession: "tab", GoalID: goal, SessionStart: hour(at), Path: PathImportedWebConsole}
	}
	// A tab open all day, a run every 6 hours: each is inside 12 hours of the last,
	// and the old sliding window chained them all.
	got := LinkWorkUnits([]RunFacts{console("a", 0, ""), console("b", 6, ""), console("c", 12, ""), console("d", 18, ""), console("e", 24, ""), console("f", 30, "")})
	if got["b"].WorkUnitID != got["a"].WorkUnitID || got["c"].WorkUnitID != got["a"].WorkUnitID {
		t.Fatalf("runs inside the window from the unit's start link: %+v", got)
	}
	if got["d"].WorkUnitID == got["a"].WorkUnitID {
		t.Fatalf("18 hours from the unit's start is past the window: %+v", got["d"])
	}
	if got["e"].WorkUnitID != got["d"].WorkUnitID {
		t.Fatalf("a new unit starts its own window: %+v", got["e"])
	}
	// Different goals in one tab are different tasks: their costs stay apart.
	apart := LinkWorkUnits([]RunFacts{console("g1", 0, "goal-1"), console("g2", 1, "goal-2"), console("g3", 2, "goal-1")})
	if apart["g2"].WorkUnitID == apart["g1"].WorkUnitID {
		t.Fatalf("a low-confidence link merged two goals: %+v", apart)
	}
	if apart["g3"].WorkUnitID != apart["g1"].WorkUnitID || apart["g3"].LinkKind != LinkGoal {
		t.Fatalf("the same goal still links by goal: %+v", apart["g3"])
	}
	// A burst inside the window takes at most the cap of low-confidence links.
	var burst []RunFacts
	for i := 0; i < 8; i++ {
		burst = append(burst, console(fmt.Sprintf("b%d", i), float64(i)/10, ""))
	}
	sized := LinkWorkUnits(burst)
	joined := 0
	for _, a := range sized {
		if a.WorkUnitID == sized["b0"].WorkUnitID {
			joined++
		}
	}
	if joined != maxLowConfidenceLinks+1 {
		t.Fatalf("unit holds %d runs, want the root and %d links", joined, maxLowConfidenceLinks)
	}
	// A workload successor is bounded the same way.
	var workload []RunFacts
	for i := 0; i < 6; i++ {
		workload = append(workload, RunFacts{RunID: fmt.Sprintf("w%d", i), WorkloadKey: "k", Status: "complete", SessionStart: hour(float64(i) * 20), Path: PathHeadless})
	}
	wl := LinkWorkUnits(workload)
	if wl["w2"].WorkUnitID != wl["w0"].WorkUnitID || wl["w3"].WorkUnitID == wl["w0"].WorkUnitID {
		t.Fatalf("workload successors chained past 48 hours from the unit's start: %+v", wl)
	}
}

// One outcome cannot speak for an orchestrator's 93 runs, even over links that
// are certain: units stop at the cap and the later children continue together in
// the next unit, so the total cost is still there and no unit exceeds the bound.
func TestUnitSizeIsBoundedAtTheCap(t *testing.T) {
	runs := []RunFacts{{RunID: "orchestrator", SessionStart: t0, Path: PathImported}}
	for i := 0; i < 92; i++ {
		runs = append(runs, RunFacts{RunID: fmt.Sprintf("child-%02d", i), ParentRunID: "orchestrator", SessionStart: t0.Add(time.Duration(i+1) * time.Minute), Path: PathImported})
	}
	got := LinkWorkUnits(runs)
	sizes := map[string]int{}
	overCap := 0
	for _, a := range got {
		sizes[a.WorkUnitID]++
		if a.LinkKind == LinkOverCap {
			overCap++
		}
	}
	if len(sizes) != 3 || overCap != 2 {
		t.Fatalf("units %v, %d over-cap roots", sizes, overCap)
	}
	for id, size := range sizes {
		if size > MaxUnitRuns {
			t.Fatalf("unit %s holds %d runs", id, size)
		}
	}
	if sizes["orchestrator"] != MaxUnitRuns {
		t.Fatalf("the first unit fills to the cap: %v", sizes)
	}
}
