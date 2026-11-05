package skills

import (
	"context"
	"testing"
	"time"

	"prompt-manager/internal/store"
)

type outcomeResolverFunc func(context.Context, string) (string, error)

func (f outcomeResolverFunc) RunStatus(ctx context.Context, id string) (string, error) {
	return f(ctx, id)
}

func TestReportOutcomesSplitsAttributedRunsAndLeavesUnattributedUncovered(t *testing.T) {
	reads := store.NewSkillReadStore(t.TempDir())
	for _, read := range []store.SkillRead{
		{SkillID: "skill-a", AgentRunID: "run-ok", CallerKind: "agent-member"},
		{SkillID: "skill-a", AgentRunID: "run-failed", CallerKind: "agent-member"},
		{SkillID: "skill-a", CallerKind: "operator-direct"},
	} {
		if err := reads.Append(read); err != nil {
			t.Fatal(err)
		}
	}
	reporter := NewUsageReporter(reads, nil)
	reporter.SetOutcomeResolver(outcomeResolverFunc(func(_ context.Context, id string) (string, error) {
		if id == "run-failed" {
			return "RUN_STATUS_FAILED", nil
		}
		return "RUN_STATUS_COMPLETE", nil
	}))
	report, err := reporter.Report(24*time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%+v", report.Rows)
	}
	row := report.Rows[0]
	if row.ReadsWithRun != 2 || row.SucceededRuns != 1 || row.FailedRuns != 1 || row.OutcomeCoverage == nil || *row.OutcomeCoverage >= 1 {
		t.Fatalf("row=%+v", row)
	}
}

func TestReportDefaultDoesNotResolveRunOutcomes(t *testing.T) {
	reads := store.NewSkillReadStore(t.TempDir())
	if err := reads.Append(store.SkillRead{SkillID: "skill-a", AgentRunID: "run-ok"}); err != nil {
		t.Fatal(err)
	}
	reporter := NewUsageReporter(reads, nil)
	reporter.SetOutcomeResolver(outcomeResolverFunc(func(context.Context, string) (string, error) {
		t.Fatal("default report resolved a run")
		return "", nil
	}))
	if _, err := reporter.Report(24 * time.Hour); err != nil {
		t.Fatal(err)
	}
}
