package toolfriction

import (
	"testing"
	"time"

	"agent-manager/internal/runsignal"
)

func TestAnalyzeRuntimeConservesTimeAndKeepsCommandEvidenceSeparateFromCheckIdentity(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	run := RuntimeRun{
		ID: "imported-run", Imported: true, Status: "complete", StartedAt: &start, EndedAt: &end,
		Time: runsignal.TimeAccounting{
			ModelGeneratingMS: 3 * 60_000,
			ToolExecutingMS:   4 * 60_000,
			IdleWaitingMS:     60_000,
			AwaitingHumanMS:   60_000,
			UnattributableMS:  60_000,
		},
		HasTime: true,
		Events: []RuntimeSignal{
			{At: start.Add(time.Minute), Kind: "invocation", Check: "test-genie runs wait", CommandPath: "test-genie runs wait", Outcome: runsignal.OutcomeSuccess, SemanticsKind: "verify", DurationMS: 60_000, DurationKnown: true},
			{At: start.Add(2 * time.Minute), Kind: "invocation", Check: "agent-manager run list", CommandPath: "agent-manager run list", Owner: "agent-manager", Outcome: runsignal.OutcomeFailure, DurationMS: 30_000, DurationKnown: true},
			{At: start.Add(3 * time.Minute), Kind: "invocation", Check: "go test", CommandPath: "go test", Outcome: runsignal.OutcomeFailure, SemanticsKind: "verify"},
			{At: start.Add(4 * time.Minute), Kind: "invocation", Check: "go test", CommandPath: "go test", Outcome: runsignal.OutcomeSuccess, SemanticsKind: "verify"},
			{At: start.Add(5 * time.Minute), Kind: "invocation", Retry: true, Outcome: runsignal.OutcomeFailure, DurationMS: 90_000, DurationKnown: true},
			{At: start.Add(6 * time.Minute), Kind: RuntimeSandboxEnvironment, DurationMS: 15_000, DurationKnown: true, DurationBasis: "unattributed"},
		},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start.Add(-time.Hour), Until: end.Add(time.Hour)})
	if report.Coverage.State != "partial" || report.Coverage.ImportedRuns != 1 || !report.Coverage.ImportedRunsIncluded {
		t.Fatalf("coverage = %+v", report.Coverage)
	}
	if report.Totals.TotalDurationMS != 10*60_000 || report.Totals.KnownDurationMS != 3*60_000+15_000 || report.Totals.UnknownDurationMS != 6*60_000+45_000 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	byCategory := map[string]RuntimeLoss{}
	for _, category := range report.Categories {
		byCategory[category.Category] = category
	}
	if got := byCategory[RuntimeWaitingOnValidation]; got.DurationMS != 60_000 || got.EvidenceCount != 1 || got.Runs != 1 {
		t.Fatalf("validation waiting = %+v", got)
	}
	if got := byCategory[RuntimeAgentManagerFailure]; got.DurationMS != 30_000 || got.EvidenceCount != 1 {
		t.Fatalf("Agent Manager failure = %+v", got)
	}
	if got := byCategory[RuntimeRework]; got.DurationMS != 90_000 || got.EvidenceCount != 1 {
		t.Fatalf("rework = %+v", got)
	}
	if got := byCategory[RuntimeSandboxEnvironment]; got.DurationMS != 15_000 || got.EvidenceCount != 1 {
		t.Fatalf("sandbox environment = %+v", got)
	}
	if got := byCategory[RuntimeBuilding]; got.DurationMS != 0 {
		t.Fatalf("unsubstantiated time was called building: %+v", got)
	}
	if len(report.BlockedChecks) != 1 {
		t.Fatalf("blocked checks = %+v", report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "" || check.Check != "go test" || check.Runs != 1 || check.Responses.Succeeded != 1 || report.Coverage.UnclassifiedBlockedRuns != 1 {
		t.Fatalf("blocked check = %+v", check)
	}
}

func TestAnalyzeRuntimeLeavesUnknownCheckIdentityAndResponseUnclassified(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run := RuntimeRun{
		ID: "run-unknown-check", Status: "complete", StartedAt: &start, EndedAt: &end,
		Events:  []RuntimeSignal{{At: start.Add(10 * time.Second), Kind: "invocation", SemanticsKind: "verify", Outcome: runsignal.OutcomeFailure}},
		HasTime: true, Time: runsignal.TimeAccounting{UnattributableMS: end.Sub(start).Milliseconds()},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start, Until: end})
	if report.Coverage.UnclassifiedBlockedRuns != 1 || len(report.BlockedChecks) != 1 {
		t.Fatalf("unclassified blocked check coverage=%+v checks=%+v", report.Coverage, report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "" || check.Responses.Unknown != 1 || check.Responses.Workaround+check.Responses.Asked+check.Responses.Succeeded+check.Responses.Stopped != 0 {
		t.Fatalf("unknown evidence was guessed: %+v", check)
	}
}

func TestAnalyzeRuntimeReportsStructuredGuardIdentityOutcomeResponseAndDurationCoverage(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run := RuntimeRun{
		ID: "guard-run", Imported: true, Status: "complete", StartedAt: &start, EndedAt: &end,
		Events: []RuntimeSignal{
			{At: start.Add(10 * time.Second), Kind: RuntimeGuardCheck, CheckID: "git_verb_blocked", Outcome: RuntimeOutcomeBlocked, DurationMS: 23, DurationKnown: true, DurationBasis: "unattributed"},
			{At: start.Add(20 * time.Second), Kind: RuntimeGuardCheck, CheckID: "git_verb_blocked", Outcome: RuntimeOutcomeBlocked},
			{At: start.Add(30 * time.Second), Kind: "message", Text: "I will stop here and ask the operator. Could you advise?"},
		},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start, Until: end})
	if report.Coverage.ImportedRuns != 1 || len(report.BlockedChecks) != 1 {
		t.Fatalf("imported guard coverage/checks = %+v / %+v", report.Coverage, report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "git_verb_blocked" || check.Runs != 1 || check.Occurrences != 2 || check.Outcomes.Blocked != 2 {
		t.Fatalf("guard identity/outcomes = %+v", check)
	}
	if len(check.BlockedRunIDs) != 1 || check.BlockedRunIDs[0] != "guard-run" {
		t.Fatalf("distinct blocked runs = %v, want guard-run", check.BlockedRunIDs)
	}
	if check.EnforcementDurationMS != 23 || check.EnforcementDurationKnown != 1 || check.EnforcementDurationUnknown != 1 {
		t.Fatalf("guard duration coverage = %+v", check)
	}
	if check.Responses.Asked != 2 || check.Responses.Unknown != 0 {
		t.Fatalf("agent response = %+v", check.Responses)
	}
	if report.Coverage.UnclassifiedBlockedRuns != 0 {
		t.Fatalf("authoritative check was marked unclassified: %+v", report.Coverage)
	}
	var stoppedDuration int64
	for _, category := range report.Categories {
		if category.Category == RuntimeStoppedByCheck {
			stoppedDuration = category.DurationMS
		}
	}
	if got := stoppedDuration; got != 23 {
		t.Fatalf("measured guard duration category = %d, want 23ms", got)
	}
}

func TestAnalyzeRuntimeSeparatesBlockedRunIDsFromOtherGuardOutcomes(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	checkID := "scenario-dependency-analyzer:dependency-addition"
	runs := []RuntimeRun{
		{ID: "blocked-run", StartedAt: &start, EndedAt: &end, Events: []RuntimeSignal{{At: start, Kind: RuntimeGuardCheck, CheckID: checkID, Outcome: RuntimeOutcomeBlocked}}},
		{ID: "allowed-run", StartedAt: &start, EndedAt: &end, Events: []RuntimeSignal{{At: start, Kind: RuntimeGuardCheck, CheckID: checkID, Outcome: "allowed"}}},
	}
	report := AnalyzeRuntime(runs, RuntimeWindow{Since: start, Until: end})
	if len(report.BlockedChecks) != 1 {
		t.Fatalf("checks = %+v, want one grouped evaluator", report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.Runs != 2 || check.Outcomes.Blocked != 1 || check.Outcomes.Allowed != 1 {
		t.Fatalf("grouped outcomes = %+v", check)
	}
	if len(check.BlockedRunIDs) != 1 || check.BlockedRunIDs[0] != "blocked-run" {
		t.Fatalf("blocked run provenance = %v, want only blocked-run", check.BlockedRunIDs)
	}
}

func TestAnalyzeRuntimeDoesNotPromoteGuardDisplayTextToIdentity(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	report := AnalyzeRuntime([]RuntimeRun{{
		ID: "guard-unknown", StartedAt: &start, EndedAt: &end,
		Events: []RuntimeSignal{{At: start, Kind: RuntimeGuardCheck, Check: "git push", Outcome: RuntimeOutcomeBlocked}},
	}}, RuntimeWindow{Since: start, Until: end})
	if len(report.BlockedChecks) != 1 || report.BlockedChecks[0].CheckID != "" || report.Coverage.UnclassifiedBlockedRuns != 1 {
		t.Fatalf("missing guard identity was guessed: report=%+v", report)
	}
}

func TestAnalyzeRuntimeLeavesUnmatchedModelAndToolTimeUnknown(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run := RuntimeRun{
		ID: "run-without-events", StartedAt: &start, EndedAt: &end, HasTime: true,
		Time: runsignal.TimeAccounting{ModelGeneratingMS: 30_000, ToolExecutingMS: 30_000},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start, Until: end})
	if report.Totals.KnownDurationMS != 0 || report.Totals.UnknownDurationMS != 60_000 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	for _, category := range report.Categories {
		if category.Category == RuntimeBuilding && category.DurationMS != 0 {
			t.Fatalf("unmatched model time was called building: %+v", category)
		}
	}
}

func TestAnalyzeRuntimeUsesMeasuredSignalDurationWhenAccountingIsUnavailable(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run := RuntimeRun{
		ID: "run-with-measured-signal", StartedAt: &start, EndedAt: &end,
		Events: []RuntimeSignal{{Kind: "invocation", SemanticsKind: "verify", Outcome: runsignal.OutcomeFailure, DurationMS: 20_000, DurationKnown: true}},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start, Until: end})
	if report.Totals.TotalDurationMS != 60_000 || report.Totals.KnownDurationMS != 20_000 || report.Totals.UnknownDurationMS != 40_000 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	for _, category := range report.Categories {
		if category.Category == RuntimeStoppedByCheck && category.DurationMS != 20_000 {
			t.Fatalf("measured check duration = %+v", category)
		}
		if category.Category == RuntimeBuilding && category.DurationMS != 0 {
			t.Fatalf("unmatched duration was called building: %+v", category)
		}
	}
}

func TestAnalyzeRuntimeCapsInconsistentAccountingAtObservedRunDuration(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run := RuntimeRun{
		ID: "run-inconsistent-time-accounting", StartedAt: &start, EndedAt: &end, HasTime: true,
		Time: runsignal.TimeAccounting{ToolExecutingMS: end.Sub(start).Milliseconds(), UnattributableMS: end.Sub(start).Milliseconds()},
		Events: []RuntimeSignal{
			{Kind: RuntimeAgentManagerFailure, DurationMS: end.Sub(start).Milliseconds(), DurationKnown: true},
			{Kind: RuntimeSandboxEnvironment, DurationMS: end.Sub(start).Milliseconds(), DurationKnown: true, DurationBasis: "unattributed"},
		},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start, Until: end})
	if report.Totals.TotalDurationMS != end.Sub(start).Milliseconds() || report.Totals.KnownDurationMS != end.Sub(start).Milliseconds() || report.Totals.UnknownDurationMS != 0 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	var categoryTotal int64
	for _, category := range report.Categories {
		categoryTotal += category.DurationMS
	}
	if categoryTotal > report.Totals.TotalDurationMS {
		t.Fatalf("category duration %d exceeds observed run duration %d", categoryTotal, report.Totals.TotalDurationMS)
	}
}
