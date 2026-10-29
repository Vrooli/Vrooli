package measures

import (
	"context"
	"math"
	"strconv"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// usageFacts serves a day of usage for two sessions: an orchestrator that
// re-reads a long context and a short helper run.
type usageFacts struct{ now time.Time }

func (u usageFacts) UsageTurns(_ context.Context, from, to time.Time) ([]usageattribution.Turn, error) {
	var turns []usageattribution.Turn
	for at := u.now.Add(-24 * time.Hour); at.Before(u.now); at = at.Add(10 * time.Minute) {
		if at.Before(from) || at.After(to) {
			continue
		}
		turns = append(turns,
			usageattribution.Turn{At: at, RunID: "orchestrator", Runner: "codex", Model: "gpt-6.1-terra", Reports: 3, Tokens: domain.UsageCounters{InputTokens: 6000, CacheReadTokens: 540_000, OutputTokens: 2400}, Facts: transcriptindex.TurnFacts{ToolCalls: 3, ToolKinds: map[string]int{"shell": 3}}},
			usageattribution.Turn{At: at, RunID: "helper", Runner: "codex", Model: "gpt-6-luna", Reports: 1, Tokens: domain.UsageCounters{InputTokens: 2000, CacheReadTokens: 20_000, OutputTokens: 400}},
		)
	}
	return turns, nil
}

func (usageFacts) UsageRuns(_ context.Context, ids []string) (map[string]usageattribution.Run, error) {
	out := map[string]usageattribution.Run{}
	for _, id := range ids {
		out[id] = usageattribution.Run{ID: id, Label: id, Session: id, SessionLabel: id, Runner: "codex", Source: usageattribution.SourceAgentManager, Succeeded: true}
	}
	return out, nil
}

func (usageFacts) UsageEpisodes(context.Context, time.Time, time.Time) ([]usageattribution.Episode, error) {
	return nil, nil
}

func (usageFacts) UsageOutcomes(context.Context, time.Time, time.Time) ([]usageattribution.Outcome, error) {
	return nil, nil
}

type staticCoverage struct{ since time.Time }

func (c *staticCoverage) Coverage(_ context.Context, since time.Time) (transcriptindex.Coverage, error) {
	c.since = since
	root := transcriptindex.RootCoverage{Kind: transcriptindex.RootCodexHome, Harness: transcriptindex.HarnessCodex, Files: 4, Indexed: 3, Pending: 1, Bytes: 1000, IndexedBytes: 970, Turns: 40, Issues: []transcriptindex.FileIssue{{Path: "/x", Reason: "not indexed yet"}}}
	return transcriptindex.Coverage{Since: since, Roots: []transcriptindex.RootCoverage{root}, Totals: root}, nil
}

func usageHandler(now time.Time) *Handler {
	h := NewHandler(&fakeStore{}, func() time.Time { return now })
	h.SetUsageForecast(&quotaReadings{spans: codexReadings(now)})
	h.SetUsageAttribution(usageFacts{now: now}, nil, &staticCoverage{})
	return h
}

// A5 at the procedure boundary: every group-by's burners, other points and
// the unattributed remainder sum to the observed increase.
func TestUsageBurnersSumToTheObservedIncreaseForEveryDimension(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	for _, by := range usageattribution.Dimensions {
		response, err := h.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{GroupBy: by, Limit: 1}))
		if err != nil {
			t.Fatalf("%s: %v", by, err)
		}
		windows := response.Msg.GetWindows()
		if len(windows) != 1 || windows[0].GetObservedIncreasePoints() != 36 || windows[0].GetCalibration().GetMethod() == "" {
			t.Fatalf("%s: windows = %v", by, windows)
		}
		w := windows[0]
		sum := w.GetOtherPoints() + w.GetUnattributedPoints()
		for _, burner := range w.GetBurners() {
			sum += burner.GetPoints()
		}
		if math.Abs(sum-w.GetObservedIncreasePoints()) > 1e-6 || len(w.GetBurners()) > 1 {
			t.Errorf("%s: burners + other + unattributed = %v of %v (%d rows)", by, sum, w.GetObservedIncreasePoints(), len(w.GetBurners()))
		}
	}
	session, err := h.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	top := session.Msg.GetWindows()[0].GetBurners()[0]
	if top.GetKey() != "orchestrator" || top.GetTokens().GetCachedInput() == 0 || len(top.GetRunIds()) != 1 || top.GetModel() != "gpt-6.1-terra" || top.GetModality().GetToolKinds()[0].GetKind() != "shell" {
		t.Fatalf("top burner = %v", top)
	}
	run, err := h.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{GroupBy: "run", RunId: "helper"}))
	if err != nil {
		t.Fatal(err)
	}
	if rows := run.Msg.GetWindows()[0].GetBurners(); len(rows) != 1 || rows[0].GetKey() != "helper" {
		t.Fatalf("run panel rows = %v", rows)
	}
	if _, err := h.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{GroupBy: "colour"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid group_by error = %v", err)
	}
}

func TestUsageRecommendationsReportEveryRuleStatus(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	response, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.GetRules()) != len(usageattribution.Rules) || response.Msg.GetDominantConsumer() != "orchestrator" {
		t.Fatalf("rules = %v dominant = %q", response.Msg.GetRules(), response.Msg.GetDominantConsumer())
	}
	top := response.Msg.GetRecommendations()[0]
	if top.GetRule() != usageattribution.RuleContext || top.GetTargetId() != "orchestrator" || top.GetRank() != 1 || top.GetEstWeeklyQuotaPointsSaved() <= 0 || len(top.GetEvidenceRefs()) == 0 {
		t.Fatalf("top recommendation = %v", top)
	}
	filtered, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{RunId: "nobody"}))
	if err != nil || len(filtered.Msg.GetRecommendations()) != 0 {
		t.Fatalf("run filter = %v, %v", filtered.Msg.GetRecommendations(), err)
	}
}

func TestUsageCoverageReportsCapturedShareAndIssues(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	response, err := h.UsageCoverage(context.Background(), connect.NewRequest(&measurepb.UsageCoverageRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if h.coverage.(*staticCoverage).since != now.Add(-7*24*time.Hour) {
		t.Fatalf("coverage since = %v, want 7 days ago", h.coverage.(*staticCoverage).since)
	}
	root := response.Msg.GetRoots()[0]
	if root.GetCapturedShare() != 0.97 || root.GetHarness() != "codex" || len(root.GetIssues()) != 1 || response.Msg.GetMethod() == "" {
		t.Fatalf("coverage = %v", response.Msg)
	}
}

func TestUsageCompareNeedsFiveSuccessesPerSide(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	response, err := usageHandler(now).UsageCompare(context.Background(), connect.NewRequest(&measurepb.UsageCompareRequest{
		Left:  &measurepb.CohortSelector{RunIds: []string{"00000000-0000-0000-0000-000000000001"}},
		Right: &measurepb.CohortSelector{RunIds: []string{"00000000-0000-0000-0000-000000000002"}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetVerdict() != "insufficient" || response.Msg.GetLeft().GetRuns() != 1 || response.Msg.GetLeft().GetSuccessfulRuns() != 1 || len(response.Msg.GetCaveats()) < 3 {
		t.Fatalf("compare = %v", response.Msg)
	}
}

// F-W4-4: every usage procedure is served by /measures/execute through the
// registry, from the same compute path as the typed RPC.
func TestRegistryServesEveryUsageMeasureFromItsProcedure(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	rpc, err := h.UsageBurners(context.Background(), connect.NewRequest(&measurepb.UsageBurnersRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	top := rpc.Msg.GetWindows()[0].GetBurners()[0]
	params := map[string]map[string]string{
		UsageCompare: {"left_run_ids": "00000000-0000-0000-0000-000000000001", "right_run_ids": "00000000-0000-0000-0000-000000000002"},
	}
	for _, name := range []string{UsageForecast, UsageForecastBacktest, UsageBurners, UsageRecommendations, UsageCoverage, UsageCompare} {
		served, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: name, Params: params[name]})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if name != UsageForecastBacktest && len(served.Fields) == 0 {
			t.Fatalf("%s served no rows", name)
		}
		switch name {
		case UsageBurners:
			first := served.Fields[0]
			if first["key"] != top.GetKey() || first["points"] != strconv.FormatFloat(top.GetPoints(), 'f', -1, 64) || first["windows.provider"] != "openai" || first["tokens.cached_input"] == "" {
				t.Fatalf("burners row = %v, want the RPC's top burner %s with its window", first, top.GetKey())
			}
		case UsageCompare:
			if len(served.Fields) != 2 || served.Fields[0]["side"] != "left" || served.Fields[0]["runs"] != "1" {
				t.Fatalf("compare rows = %v", served.Fields)
			}
		}
	}
}
