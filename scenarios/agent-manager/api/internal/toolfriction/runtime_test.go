package toolfriction

import (
	"testing"
	"time"

	"agent-manager/internal/runsignal"
)

func TestAnalyzeRuntimeConservesTimeAndGroupsBlockedChecksByStableID(t *testing.T) {
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
			{At: start.Add(time.Minute), Kind: "invocation", Check: "test-genie runs wait", CheckID: "test-genie runs wait", CommandPath: "test-genie runs wait", Outcome: runsignal.OutcomeSuccess, SemanticsKind: "verify", DurationMS: 60_000, DurationKnown: true},
			{At: start.Add(2 * time.Minute), Kind: "invocation", Check: "agent-manager run list", CommandPath: "agent-manager run list", Owner: "agent-manager", Outcome: runsignal.OutcomeFailure, DurationMS: 30_000, DurationKnown: true},
			{At: start.Add(3 * time.Minute), Kind: "invocation", Check: "go test", CheckID: "go test ./...", CommandPath: "go test", Outcome: runsignal.OutcomeFailure, SemanticsKind: "verify"},
			{At: start.Add(4 * time.Minute), Kind: "invocation", Check: "go test", CheckID: "go test ./...", CommandPath: "go test", Outcome: runsignal.OutcomeSuccess, SemanticsKind: "verify"},
			{At: start.Add(5 * time.Minute), Kind: "invocation", Retry: true, Outcome: runsignal.OutcomeFailure, DurationMS: 90_000, DurationKnown: true},
			{At: start.Add(6 * time.Minute), Kind: RuntimeSandboxEnvironment, DurationMS: 15_000, DurationKnown: true, DurationBasis: "unattributed"},
		},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start.Add(-time.Hour), Until: end.Add(time.Hour)})
	if report.Coverage.State != "partial" || report.Coverage.ImportedRuns != 1 || !report.Coverage.ImportedRunsIncluded {
		t.Fatalf("coverage = %+v", report.Coverage)
	}
	if report.Totals.TotalDurationMS != 10*60_000 || report.Totals.KnownDurationMS != 7*60_000+15_000 || report.Totals.UnknownDurationMS != 3*60_000-15_000 {
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
	if got := byCategory[RuntimeBuilding]; got.DurationMS != 4*60_000 {
		t.Fatalf("building = %+v", got)
	}
	if len(report.BlockedChecks) != 1 {
		t.Fatalf("blocked checks = %+v", report.BlockedChecks)
	}
	check := report.BlockedChecks[0]
	if check.CheckID != "go test ./..." || check.Check != "go test" || check.Runs != 1 || check.Responses.Succeeded != 1 {
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

func TestAnalyzeRuntimeReportsUnknownDurationWithoutManufacturingBuilding(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	run := RuntimeRun{
		ID: "run-without-events", StartedAt: &start, EndedAt: &end, HasTime: true,
		Time: runsignal.TimeAccounting{ModelGeneratingMS: 30_000, ToolExecutingMS: 30_000},
	}
	report := AnalyzeRuntime([]RuntimeRun{run}, RuntimeWindow{Since: start, Until: end})
	if report.Totals.KnownDurationMS != 30_000 || report.Totals.UnknownDurationMS != 30_000 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	for _, category := range report.Categories {
		if category.Category == RuntimeBuilding && category.DurationMS != 30_000 {
			t.Fatalf("building = %+v", category)
		}
	}
}
