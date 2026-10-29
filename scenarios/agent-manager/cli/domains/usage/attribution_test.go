package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

// attributionFixture serves canned W5 Measures answers and records requests.
type attributionFixture struct {
	measureconnect.UnimplementedMeasuresServiceHandler
	burners  *measurepb.UsageBurnersRequest
	recs     *measurepb.UsageRecommendationsRequest
	coverage *measurepb.UsageCoverageRequest
	compare  *measurepb.UsageCompareRequest
}

func (f *attributionFixture) UsageBurners(_ context.Context, req *connect.Request[measurepb.UsageBurnersRequest]) (*connect.Response[measurepb.UsageBurnersResponse], error) {
	f.burners = req.Msg
	return connect.NewResponse(&measurepb.UsageBurnersResponse{GeneratedAt: "2026-10-29T06:00:00Z", Windows: []*measurepb.UsageWindowAttribution{{
		Provider: "openai", Pool: "primary", Window: "10080m", WindowMinutes: 10080, PeriodStart: "2026-10-28T06:57:00Z", PeriodEnd: "2026-10-29T05:48:31Z",
		UsedPercent: 34, ObservedIncreasePoints: 34, AttributedPoints: 29, UnattributedPoints: 3, OtherPoints: 2, OtherBurners: 4, GroupBy: "session",
		Burners:       []*measurepb.UsageBurner{{Key: "web-console:1e5904ed", Label: "1e5904ed research orchestrator", Points: 26, Share: 26.0 / 34, Model: "gpt-6.1-terra", Runner: "codex", Runs: 45, Tokens: &measurepb.UsageTokens{Total: 3_100_000_000, CachedInput: 3_000_000_000}}, {Key: "vega", Label: "run Vega E1", Points: 1, Share: 1.0 / 34, Runs: 1, Tokens: &measurepb.UsageTokens{Total: 10_000}}},
		Calibration:   &measurepb.UsageCalibration{Method: "fitted", Slices: 12, RSquared: 0.6, Weights: []*measurepb.UsageModelWeight{{Model: "gpt-6.1-terra", CachedInputTokensPerPoint: 100_000_000}}},
		PendingTokens: &measurepb.UsageTokens{Total: 5_000_000},
		MeteredLines:  []*measurepb.UsageMeteredLine{{Kind: "codex_credits", State: "known", Amount: "0", Note: "credits balance 62500 unchanged in this window"}},
	}}}), nil
}

func (f *attributionFixture) UsageRecommendations(_ context.Context, req *connect.Request[measurepb.UsageRecommendationsRequest]) (*connect.Response[measurepb.UsageRecommendationsResponse], error) {
	f.recs = req.Msg
	return connect.NewResponse(&measurepb.UsageRecommendationsResponse{
		GeneratedAt: "2026-10-29T06:00:00Z", DominantConsumer: "web-console:1e5904ed", DominantConsumerLabel: "1e5904ed research orchestrator",
		Recommendations: []*measurepb.UsageRecommendation{{
			Rank: 1, Rule: "R-context", TargetKind: "session", TargetId: "web-console:1e5904ed", Provider: "openai", Window: "10080m", EstWeeklyQuotaPointsSaved: 8, Confidence: "high", ConsumerPoints: 26,
			Action: "Start a fresh session or compact 1e5904ed (185k re-read per request over 900 requests)", EvidenceRefs: []*measurepb.UsageEvidenceRef{{Kind: "session", Id: "web-console:1e5904ed"}, {Kind: "run", Id: "a"}, {Kind: "run", Id: "b"}},
		}},
		Rules: []*measurepb.UsageRuleStatus{{Rule: "R-context", Fired: true, Rows: 1}, {Rule: "R-model", Reason: "no workload ran on two models"}},
	}), nil
}

func (f *attributionFixture) UsageCoverage(_ context.Context, req *connect.Request[measurepb.UsageCoverageRequest]) (*connect.Response[measurepb.UsageCoverageResponse], error) {
	f.coverage = req.Msg
	root := &measurepb.UsageCoverageRoot{Kind: "codex-home", Harness: "codex", Files: 40, Indexed: 38, Pending: 1, Unreadable: 1, Bytes: 1000, CapturedBytes: 970, CapturedShare: 0.97, Turns: 900, UsageReports: 2000, Issues: []*measurepb.UsageCoverageIssue{{Path: "/home/x/rollout.jsonl", Reason: "not indexed yet"}}}
	return connect.NewResponse(&measurepb.UsageCoverageResponse{Since: "2026-10-22T06:00:00Z", Roots: []*measurepb.UsageCoverageRoot{root}, Totals: root, Method: "Independent scan: …"}), nil
}

func (f *attributionFixture) UsageCompare(_ context.Context, req *connect.Request[measurepb.UsageCompareRequest]) (*connect.Response[measurepb.UsageCompareResponse], error) {
	f.compare = req.Msg
	return connect.NewResponse(&measurepb.UsageCompareResponse{
		Verdict: "improved", PointsPerSuccessChange: -0.4,
		Left:    &measurepb.UsageCompareSide{Runs: 10, SuccessfulRuns: 8, PricedRuns: 10, Points: 4, PointsPerSuccess: 0.5, SuccessRate: 0.8},
		Right:   &measurepb.UsageCompareSide{Runs: 10, SuccessfulRuns: 10, PricedRuns: 10, Points: 3, PointsPerSuccess: 0.3, SuccessRate: 1},
		Caveats: []string{"Cohorts are observational."},
	}), nil
}

func runUsageCommand(t *testing.T, fixture *attributionFixture, build func(clientFor) cliapp.Command, args ...string) string {
	t.Helper()
	path, handler := measureconnect.NewMeasuresServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	cmd := build(func(*cliapp.ScenarioApp) measureconnect.MeasuresServiceClient {
		return measureconnect.NewMeasuresServiceClient(server.Client(), server.URL)
	})
	var stdout bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("usage %s %v: %v", cmd.Name, args, err)
	}
	return stdout.String()
}

func assertContains(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestUsageBurnersShowsSharesRemainderAndCalibration(t *testing.T) {
	fixture := &attributionFixture{}
	out := runUsageCommand(t, fixture, burnersCommand, "--by", "model", "--provider", "openai", "--run", "r1", "--limit", "5", "--since", "2026-10-28T12:00:00Z")
	if f := fixture.burners; f.GetGroupBy() != "model" || f.GetProvider() != "openai" || f.GetRunId() != "r1" || f.GetLimit() != 5 || f.GetSince() != "2026-10-28T12:00:00Z" {
		t.Fatalf("request = %v", f)
	}
	assertContains(t, out,
		"Codex · weekly  Wed Oct 28 06:57 UTC → Thu Oct 29 05:48 UTC  observed +34 pts (now 34%)",
		"≈ 26.0 pts   76%  1e5904ed research orchestrator  [gpt-6.1-terra, codex, 45 runs, 3.1B tokens (3.0B cached)]",
		"≈  2.0 pts        4 more",
		"≈  3.0 pts    9%  unattributed / off-disk",
		"calibration fitted over 12 block(s), R² 0.60; gpt-6.1-terra 100.0M cached/pt",
		"pending: 5.0M tokens after the newest reading",
		"metered outside tokens: codex_credits ×0 (known) 0 — credits balance 62500 unchanged in this window",
	)
	var decoded struct {
		Windows []struct {
			UnattributedPoints float64 `json:"unattributed_points"`
			Burners            []struct {
				Key string `json:"key"`
			} `json:"burners"`
		} `json:"windows"`
	}
	if err := json.Unmarshal([]byte(runUsageCommand(t, fixture, burnersCommand, "--json")), &decoded); err != nil || decoded.Windows[0].UnattributedPoints != 3 || decoded.Windows[0].Burners[0].Key != "web-console:1e5904ed" {
		t.Fatalf("--json must be the Measures wire shape: %v %+v", err, decoded)
	}
}

func TestUsageRecommendationsShowsRankedActionsAndSilentRules(t *testing.T) {
	fixture := &attributionFixture{}
	out := runUsageCommand(t, fixture, recommendationsCommand, "--run", "a")
	if fixture.recs.GetRunId() != "a" {
		t.Fatalf("request = %v", fixture.recs)
	}
	assertContains(t, out,
		"Dominant consumer: 1e5904ed research orchestrator",
		"1. Start a fresh session or compact 1e5904ed (185k re-read per request over 900 requests)  −8.0 pts/wk of Codex · weekly  [high]",
		"R-context · session web-console:1e5904ed · uses ≈26.0 pts · evidence: 1 session, 2 run",
		"R-model did not fire: no workload ran on two models",
	)
}

func TestUsageCoverageAndCompareRender(t *testing.T) {
	fixture := &attributionFixture{}
	assertContains(t, runUsageCommand(t, fixture, coverageCommand, "--since", "2026-10-22T06:00:00Z"),
		"captured 97.0% of transcript bytes since Thu Oct 22 06:00 UTC · 40 files · 1 pending · 1 unreadable",
		"codex-home               codex        97.0%  files 40 (indexed 38, owned 0, excluded 0, pending 1, unreadable 1)  turns 900, usage reports 2000",
		"not indexed yet: /home/x/rollout.jsonl",
	)
	if fixture.coverage.GetSince() != "2026-10-22T06:00:00Z" {
		t.Fatalf("coverage request = %v", fixture.coverage)
	}
	out := runUsageCommand(t, fixture, compareCommand, "--left-run-ids", "00000000-0000-0000-0000-000000000001", "--right-cohort", "after", "--change-binding", "luna-move")
	if c := fixture.compare; len(c.GetLeft().GetRunIds()) != 1 || c.GetRight().GetCohort() != "after" || c.GetChangeBinding() != "luna-move" {
		t.Fatalf("compare request = %v", c)
	}
	assertContains(t, out, "Verdict: improved (points per successful run -40%)", "left  10 runs, 8 successful (80%), 10 priced · ≈4.00 pts · ≈0.500 pts per success")
}
