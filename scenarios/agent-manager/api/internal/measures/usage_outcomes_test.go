package measures

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/outcomeledger"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

type ledgerFixture struct {
	runs              []outcomeledger.RunFacts
	backfill, pending int64
	periods           []outcomeledger.ModePeriod
}

func (f ledgerFixture) FactsInWindow(context.Context, time.Time, time.Time) ([]outcomeledger.RunFacts, error) {
	return f.runs, nil
}
func (f ledgerFixture) BackfillCount(context.Context, time.Time, time.Time) (int64, error) {
	return f.backfill, nil
}
func (f ledgerFixture) PendingCount(context.Context, time.Time) (int64, error) { return f.pending, nil }
func (f ledgerFixture) ModePeriods(context.Context) ([]outcomeledger.ModePeriod, error) {
	return f.periods, nil
}

// importedCompleted reads every run as an imported session whose transcript
// ended: complete, with no outcome estimate.
type importedCompleted struct{ usageFacts }

func (importedCompleted) UsageRuns(_ context.Context, ids []string) (map[string]usageattribution.Run, error) {
	out := map[string]usageattribution.Run{}
	for _, id := range ids {
		out[id] = usageattribution.Run{ID: id, Runner: "codex", Source: usageattribution.SourceIndexed, Status: "complete", Completed: true, OutcomeUnknown: true, WorkUnit: id}
	}
	return out, nil
}

func TestUsageCompareNeverCountsImportedCompleteAsAccepted(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := NewHandler(&fakeStore{}, func() time.Time { return now })
	h.SetUsageForecast(&quotaReadings{spans: codexReadings(now)})
	h.SetUsageAttribution(importedCompleted{usageFacts{now: now}}, nil, &staticCoverage{}, &memoryStatuses{})
	ids := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000005"}
	response, err := h.UsageCompare(context.Background(), connect.NewRequest(&measurepb.UsageCompareRequest{
		Left: &measurepb.CohortSelector{RunIds: ids}, Right: &measurepb.CohortSelector{RunIds: ids},
	}))
	if err != nil {
		t.Fatal(err)
	}
	left := response.Msg.GetLeft()
	if left.GetCompletedRuns() != 5 || left.GetAcceptedUnits() != 0 || left.GetUnknownUnits() != 5 || response.Msg.GetVerdict() != "insufficient" {
		t.Fatalf("imported complete counted as accepted: %v", response.Msg)
	}
}

func TestUsageParityReportsPathsAndGaps(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	h.SetOutcomeLedger(ledgerFixture{backfill: 7, pending: 2, runs: []outcomeledger.RunFacts{
		{RunID: "a", Path: outcomeledger.PathImported, Harness: "codex", Requests: 3, ActualModel: "gpt", ModeRelation: outcomeledger.ModeNone, FileStatus: "indexed", FileFullyRead: true, UnitOutcome: outcomeledger.OutcomeUnknown, UnitSource: outcomeledger.SourceNoSignal},
		{RunID: "h", Path: outcomeledger.PathAttached, Harness: "grok", AttachedClass: outcomeledger.AttachedHelper},
		{RunID: "g", Path: outcomeledger.PathAttached, Harness: "grok", AttachedClass: outcomeledger.AttachedNoReader},
	}})
	response, err := h.UsageParity(context.Background(), connect.NewRequest(&measurepb.UsageParityRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetRows()) != 2 || response.Msg.GetBackfillRuns() != 7 || response.Msg.GetLedgerPending() != 2 {
		t.Fatalf("parity = %v", response.Msg)
	}
	var grok bool
	for _, gap := range response.Msg.GetGaps() {
		grok = grok || (strings.HasPrefix(gap.GetSubject(), "grok") && gap.GetCount() == 1 && gap.GetReason() != "")
	}
	if !grok {
		t.Fatalf("no grok gap: %v", response.Msg.GetGaps())
	}
}

func TestUsageOutcomesReadsTheLedgerAndRejectsUnknownDimensions(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	h.SetOutcomeLedger(ledgerFixture{runs: []outcomeledger.RunFacts{
		{RunID: "orchestrator", WorkUnitID: "orchestrator", Path: outcomeledger.PathImported, Harness: "codex", Requests: 3, ActualModel: "gpt-6.1-terra", UnitOutcome: outcomeledger.OutcomeUnknown, UnitSource: outcomeledger.SourceNoSignal},
		{RunID: "helper", WorkUnitID: "helper", Path: outcomeledger.PathHeadless, Harness: "codex", Requests: 3, ActualModel: "gpt-6-luna", UnitOutcome: outcomeledger.OutcomeCompleted, UnitSource: outcomeledger.SourceHeadlessStatus},
	}})
	response, err := h.UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{By: "model"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetUnits() != 2 || response.Msg.GetUnknownUnits() != 1 || response.Msg.GetCompletedUnits() != 1 || len(response.Msg.GetRows()) != 2 {
		t.Fatalf("outcomes = %v", response.Msg)
	}
	for _, row := range response.Msg.GetRows() {
		if row.GetVerdict() != outcomeledger.StatusNotEnough || row.GetStatusReason() == "" || row.GetPointsPerAccepted().GetAvailable() {
			t.Fatalf("low data must read not enough data: %v", row)
		}
		if row.GetPointsTotal() <= 0 {
			t.Fatalf("points were not joined from attribution: %v", row)
		}
	}
	if len(response.Msg.GetRefit()) == 0 {
		t.Fatal("refit missing")
	}
	if _, err := h.UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{By: "tone"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid dimension: %v", err)
	}
}

// countingTurns counts how many times the slow usage scan runs.
type countingTurns struct {
	usageFacts
	scans *atomic.Int64
}

func (c countingTurns) UsageTurns(ctx context.Context, from, to time.Time) ([]usageattribution.Turn, error) {
	c.scans.Add(1)
	return c.usageFacts.UsageTurns(ctx, from, to)
}

// The usage scan costs seconds on a real database. A view reads several windows
// and dimensions; only the first read of a window pays, and a stale one still
// answers at once while a single refresh runs behind it.
func TestOutcomesReuseTheWindowScanAndRefreshInTheBackground(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	clock := now
	var scans atomic.Int64
	h := NewHandler(&fakeStore{}, func() time.Time { return clock })
	h.SetUsageForecast(&quotaReadings{spans: codexReadings(now)})
	h.SetUsageAttribution(countingTurns{usageFacts: usageFacts{now: now}, scans: &scans}, nil, &staticCoverage{}, &memoryStatuses{})
	h.SetOutcomeLedger(ledgerFixture{runs: []outcomeledger.RunFacts{{RunID: "helper", WorkUnitID: "helper", Path: outcomeledger.PathHeadless, Harness: "codex", Requests: 3, ActualModel: "gpt-6-luna", UnitOutcome: outcomeledger.OutcomeCompleted}}})
	read := func(by string) {
		t.Helper()
		if _, err := h.UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{By: by})); err != nil {
			t.Fatal(err)
		}
	}
	read("model")
	read("mode")
	clock = now.Add(time.Minute) // the rolling window's edges moved a little
	read("role")
	if got := scans.Load(); got != 1 {
		t.Fatalf("three reads of one window scanned %d times", got)
	}
	clock = now.Add(5 * time.Minute) // stale: answered from memory, refreshed behind it
	read("model")
	deadline := time.Now().Add(5 * time.Second)
	for scans.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := scans.Load(); got != 2 {
		t.Fatalf("a stale read must start one refresh: %d scans", got)
	}
	clock = now.Add(time.Hour) // too old to serve
	read("model")
	if got := scans.Load(); got != 3 {
		t.Fatalf("an expired entry must be read again: %d scans", got)
	}
}

func TestPeriodUseCountsAnOverlappingPoolOnce(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	service, err := h.usageService()
	if err != nil {
		t.Fatal(err)
	}
	period, err := service.PeriodUse(context.Background(), "", now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	var tokens domain.UsageCounters
	for _, use := range period.Runs["helper"] {
		tokens.InputTokens += use.Tokens.InputTokens
	}
	if tokens.InputTokens != 2000*144 {
		t.Fatalf("helper tokens = %d", tokens.InputTokens)
	}
}

func TestModelSeriesSumsPointsPerModelAndDay(t *testing.T) {
	day := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	runs := []outcomeledger.RunFacts{
		{RunID: "a", Harness: "codex", FirstActivity: day},
		{RunID: "b", Harness: "codex", FirstActivity: day.Add(2 * time.Hour)},
		{RunID: "c", Harness: "codex", FirstActivity: day.Add(30 * time.Hour)},
		{RunID: "outside", Harness: "codex", FirstActivity: day.Add(-90 * 24 * time.Hour)},
	}
	use := map[string]map[string]outcomeledger.Use{
		"a": {"m1": {Points: 1.5}}, "b": {"m1": {Points: 0.5}, "m2": {Points: 2}}, "c": {"m1": {Points: 4}}, "outside": {"m1": {Points: 99}},
	}
	got := modelSeries(runs, use, day.Add(-24*time.Hour), day.Add(72*time.Hour))
	if len(got) != 2 || got[0].Model != "m1" || got[1].Model != "m2" {
		t.Fatalf("models = %v", got)
	}
	first := got[0].Buckets
	if len(first) != 2 || first[0].Points != 2 || first[0].Runs != 2 || first[1].Points != 4 {
		t.Fatalf("m1 buckets = %v", first)
	}
}

func TestModePeriodsKeepOnlyOverlappingPeriodsWithLabels(t *testing.T) {
	from := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	to := from.Add(7 * 24 * time.Hour)
	periods := []outcomeledger.ModePeriod{
		{ID: "old", StartedAt: from.Add(-48 * time.Hour), EndedAt: from.Add(-24 * time.Hour)},
		{ID: "in", Runner: "codex", StartedAt: from.Add(24 * time.Hour), EndedAt: from.Add(30 * time.Hour)},
		{ID: "open", Runner: "claude", StartedAt: from.Add(48 * time.Hour)},
	}
	got := protoModePeriods(periods, map[string]string{"open": "Lower tier"}, from, to)
	if len(got) != 2 || got[0].Id != "in" || got[1].Id != "open" || got[1].Label != "Lower tier" || got[1].EndedAt != "" {
		t.Fatalf("periods = %v", got)
	}
}
