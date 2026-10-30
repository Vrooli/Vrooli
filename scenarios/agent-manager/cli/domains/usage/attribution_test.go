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

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
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
	friction *measurepb.UsageFrictionRequest
}

func (f *attributionFixture) UsageFriction(_ context.Context, req *connect.Request[measurepb.UsageFrictionRequest]) (*connect.Response[measurepb.UsageFrictionResponse], error) {
	f.friction = req.Msg
	return connect.NewResponse(&measurepb.UsageFrictionResponse{
		GeneratedAt: "2026-10-29T06:00:00Z", Open: 3, Reported: 1, RunsAnalysed: 3, RunsTotal: 10,
		Problems: []*measurepb.UsageFrictionProblem{
			{Fingerprint: "4454d97b01096183cda4b8798ba39b7e", Description: "Agents run `rg` repeatedly instead of the preferred `search-hub query`.", Owner: "rg", OwnerKind: "command", Episodes: 87, Runs: 53, TypicalWallClockMs: 477_000, WallClockMs: 196_560_000, Points: 0.68, Provider: "anthropic", Window: "10080m", Status: "open", TimeKind: "redundant"},
			{Fingerprint: "95e3a7a62e5a82c4dcb7aca8ddac38c6", Description: "An agent reads a file, then reads the same file again with no edit in between.", Owner: "Agent behaviour", OwnerKind: "agent", Episodes: 839, Runs: 18, TypicalWallClockMs: 280, WallClockMs: 677_000_000, Points: 7.19, Provider: "anthropic", Window: "10080m", Status: "reported", BugId: "knw-1790111426711964967", TimeKind: "redundant"},
			{Fingerprint: "85dd60381afd06f12320256ee6c24d40", Description: "An agent repeats a short wait instead of one blocking wait.", Owner: "Agent behaviour", OwnerKind: "agent", Episodes: 50, Runs: 11, EpisodeTurns: 533, Tokens: 4_200_000, Points: 2.5, Provider: "anthropic", Window: "10080m", Status: "open", TimeKind: "redundant"},
			{Fingerprint: "4885b8f01096183cda4b8798ba39b7ef", Description: "A run sits idle between tool calls for a long stretch.", Owner: "Agent behaviour", OwnerKind: "agent", Episodes: 229, Runs: 57, TypicalWallClockMs: 819_000, WallClockMs: 693_000_000, Points: 1.4, Provider: "anthropic", Window: "10080m", Status: "open", TimeKind: "idle"},
		},
	}), nil
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
		FoldedGroups: 3, FoldedPoints: 0.12,
		Groups: []*measurepb.UsageRecommendationGroup{{
			Rank: 1, TargetKind: "session", TargetId: "web-console:1e5904ed", TargetLabel: "1e5904ed research orchestrator", Provider: "openai", Window: "10080m",
			CombinedSavedPoints: 9.2, UsedSoFarPoints: 26, AtThisRatePoints: 60, RecoverablePoints: 1.5, RemainingHours: 24, ShareOfWindowBurn: 0.76, StillRunning: true,
			Levers: []*measurepb.UsageLever{
				{Rule: "R-context", Confidence: "high", SavedPoints: 8, MarginalPoints: 8, Action: "Start a fresh session or compact 1e5904ed (185k re-read per request over 900 requests)", EvidenceRefs: []*measurepb.UsageEvidenceRef{{Kind: "session", Id: "web-console:1e5904ed"}, {Kind: "run", Id: "a"}, {Kind: "run", Id: "b"}}},
				{Rule: "R-idle", Confidence: "low", SavedPoints: 1.5, MarginalPoints: 1.2, Action: "Replace polling in 1e5904ed with an event wait or park it", EvidenceRefs: []*measurepb.UsageEvidenceRef{{Kind: "session", Id: "web-console:1e5904ed"}}},
			},
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
		"≈  2.0 pts        4 smaller items",
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

func TestUsageRecommendationsShowsGroupsWithCombinedSavingAndSilentRules(t *testing.T) {
	fixture := &attributionFixture{}
	out := runUsageCommand(t, fixture, recommendationsCommand, "--run", "a", "--provider", "openai", "--pool", "primary", "--window", "10080m")
	if r := fixture.recs; r.GetRunId() != "a" || r.GetProvider() != "openai" || r.GetPool() != "primary" || r.GetWindow() != "10080m" {
		t.Fatalf("request = %v", r)
	}
	assertContains(t, out,
		"1 group(s) as of Thu Oct 29 06:00 UTC; savings are estimates (3 more saving under 0.1 pts together ≈0.12)",
		"Dominant consumer: 1e5904ed research orchestrator",
		"1. 1e5904ed research orchestrator  projected −9.2 pts of Codex · weekly at this rate (levers alone sum −9.5), −1.5 of it still recoverable in the 24h left",
		"uses ≈26.0 pts so far, ≈60.0 at this rate · 76% of the window's burn · still running",
		"projected −8.0 pts  Start a fresh session or compact 1e5904ed (185k re-read per request over 900 requests)  [R-context, high] evidence: 1 session, 2 run",
		"projected −1.2 more (−1.5 alone) pts  Replace polling in 1e5904ed with an event wait or park it  [R-idle, low]",
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

func TestUsageFrictionListsPlainProblemsWithOwnerCostAndStatus(t *testing.T) {
	fixture := &attributionFixture{}
	out := runUsageCommand(t, fixture, frictionCommand, "--status", "open", "--limit", "5")
	if f := fixture.friction; f.GetStatus() != "open" || f.GetLimit() != 5 {
		t.Fatalf("request = %v", f)
	}
	assertContains(t, out,
		"3 open · 1 reported · 0 dismissed as of Thu Oct 29 06:00 UTC",
		"4454d97b01096183cda4b8798ba39b7e  Agents run `rg` repeatedly instead of the preferred `search-hub query`.",
		"owner rg · 87 episodes in 53 runs · redundant call time typically 7m 57s, 54h 36m in total · ≈0.7 pts of Claude · weekly · open",
		"owner Agent behaviour · 839 episodes in 18 runs · redundant call time typically 280 ms, 188h 03m in total · ≈7.2 pts of Claude · weekly · reported knw-1790111426711964967",
		"Re-analysing 3 of 10 runs: counts and totals cover the analysed runs only.",
		"idle gap typically 13m 39s, 192h 30m in total",
		"owner Agent behaviour · 50 episodes in 11 runs · ≈2.5 pts",
	)
}

type episodesFixture struct {
	domainconnect.UnimplementedEpisodesServiceHandler
	reported  *domainpb.ReportFrictionProblemRequest
	dismissed *domainpb.DismissFrictionProblemRequest
}

func (f *episodesFixture) ReportFrictionProblem(_ context.Context, req *connect.Request[domainpb.ReportFrictionProblemRequest]) (*connect.Response[domainpb.ReportFrictionProblemResponse], error) {
	f.reported = req.Msg
	return connect.NewResponse(&domainpb.ReportFrictionProblemResponse{Fingerprint: req.Msg.GetFingerprint(), Description: "Agents run `rg` repeatedly.", Status: "reported", BugId: "knw-7"}), nil
}

func (f *episodesFixture) DismissFrictionProblem(_ context.Context, req *connect.Request[domainpb.DismissFrictionProblemRequest]) (*connect.Response[domainpb.DismissFrictionProblemResponse], error) {
	f.dismissed = req.Msg
	return connect.NewResponse(&domainpb.DismissFrictionProblemResponse{Fingerprint: req.Msg.GetFingerprint(), Status: "dismissed"}), nil
}

func TestUsageFrictionReportAndDismissWriteThroughTheEpisodesService(t *testing.T) {
	fixture := &episodesFixture{}
	path, handler := domainconnect.NewEpisodesServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := func(*cliapp.ScenarioApp) domainconnect.EpisodesServiceClient {
		return domainconnect.NewEpisodesServiceClient(server.Client(), server.URL)
	}
	run := func(cmd cliapp.Command, args ...string) string {
		var stdout bytes.Buffer
		ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.RunCtx(ctx); err != nil {
			t.Fatalf("%s %v: %v", cmd.Name, args, err)
		}
		return stdout.String()
	}
	assertContains(t, run(frictionReportCommand(client), "4454d97b"), "Reported as knw-7: Agents run `rg` repeatedly.")
	if fixture.reported.GetFingerprint() != "4454d97b" {
		t.Fatalf("report request = %v", fixture.reported)
	}
	assertContains(t, run(frictionDismissCommand(client), "4454d97b"), "Dismissed 4454d97b")
	if fixture.dismissed.GetFingerprint() != "4454d97b" {
		t.Fatalf("dismiss request = %v", fixture.dismissed)
	}
	var decoded struct {
		BugID string `json:"bug_id"`
	}
	if err := json.Unmarshal([]byte(run(frictionReportCommand(client), "4454d97b", "--json")), &decoded); err != nil || decoded.BugID != "knw-7" {
		t.Fatalf("--json must be the wire shape: %v %+v", err, decoded)
	}
}

// A problem with no measured time prints no time sentence, never "0s".
func TestFrictionTimeCostOmitsUnmeasuredTime(t *testing.T) {
	if got := frictionTimeCost(&measurepb.UsageFrictionProblem{}); got != "" {
		t.Errorf("unmeasured = %q, want empty", got)
	}
	if got := frictionTimeCost(&measurepb.UsageFrictionProblem{WallClockMs: 60_000, TypicalWallClockMs: 30_000}); !strings.Contains(got, "redundant call time") {
		t.Errorf("measured = %q", got)
	}
}
