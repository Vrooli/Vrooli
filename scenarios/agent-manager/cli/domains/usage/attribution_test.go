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
	outcomes *measurepb.UsageOutcomesRequest
	friction *measurepb.UsageFrictionRequest
	// interventions is the operator interventions section the friction fixture serves.
	interventions *measurepb.UsageInterventions
}

func (f *attributionFixture) UsageFriction(_ context.Context, req *connect.Request[measurepb.UsageFrictionRequest]) (*connect.Response[measurepb.UsageFrictionResponse], error) {
	f.friction = req.Msg
	return connect.NewResponse(&measurepb.UsageFrictionResponse{
		GeneratedAt: "2026-10-29T06:00:00Z", Open: 2, Reported: 1, WindowStart: "2026-10-22T06:00:00Z", BelowFloor: 371, BeyondLimit: 3, MaterialityScore: 0.01,
		Coverage: &measurepb.ToolFrictionCoverage{Commands: 175706, OutcomeKnown: 70000, Sentence: "175706 commands, outcome known for 40% (24041 Vrooli CLI calls, 35% known)"},
		ToolFindings: []*measurepb.UsageToolFinding{
			{Fingerprint: "a1b2c3", Kind: "failure", Owner: "vrooli", OwnerKind: "control-plane", Description: "`prompt-manager team bug-capture`, `search-hub query` and 6 other commands failed 128 times in 20 runs with: Error: agent identity identity_refresh_unavailable", Calls: 128, Runs: 20, CostUnit: "calls", Cost: 128, CostShare: 0.158, Signature: "identity_refresh_unavailable", ExampleCommands: []string{"search-hub query notes"}, Status: "open", Shares: []*measurepb.UsageToolShare{{Sentence: "failure 128 calls = 15.8% of the window's 812 Vrooli CLI failed calls"}}},
			{Fingerprint: "d4e5f6", Kind: "degraded", Owner: "agent-manager", OwnerKind: "scenario", CommandPath: "test-genie runs", Description: "`test-genie runs` reported \"agent-manager park unavailable … waiting inline instead\" 161 times in 34 runs", Calls: 161, Runs: 34, CostUnit: "calls", Cost: 161, CostShare: 0.0067, Status: "reported", BugId: "knw-9", Shares: []*measurepb.UsageToolShare{{Sentence: "degraded 161 calls = 0.7% of the window's 24041 Vrooli CLI calls"}}},
			{
				Fingerprint: "778899", Kind: "output-weight", Owner: "vrooli", CommandPath: "vrooli locks", Description: "`vrooli locks` returns heavy output", Calls: 18, Runs: 4, CostUnit: "tokens", Cost: 3_200_000, CostShare: 0.01, Status: "open",
				PrintedTokens: 262_144, ReadTokens: 10_000, Shares: []*measurepb.UsageToolShare{{Sentence: "output-weight 3.2M tokens = 1.0% of the window's context tax (320M re-read tokens)"}},
			},
			{Fingerprint: "s10w01", Kind: "slow", Owner: "agent-manager", CommandPath: "agent-manager run status", Description: "`agent-manager run status` should answer in under 5s but its p90 is 41s over 30 timed calls: 12m beyond the budget", Calls: 30, Runs: 9, CostUnit: "ms", Cost: 720_000, CostShare: 0.05, Status: "open", Shares: []*measurepb.UsageToolShare{{Sentence: "slow 12m 00s = 5.0% of the window's Vrooli CLI time (4h 00m)"}}},
			{
				Fingerprint: "r00t01", Kind: "failure", Owner: "vrooli", OwnerKind: "control-plane", RootCause: "identity", Description: "6 commands hit the identity broker or refresh failing 40 times in 20 runs", Calls: 40, Runs: 20, CostUnit: "calls", Cost: 40, CostShare: 0.037, Status: "open",
				Shares: []*measurepb.UsageToolShare{{Sentence: "failure 30 calls = 3.7% of the window's 812 Vrooli CLI failed calls"}, {Sentence: "degraded 10 calls = 0.0% of the window's 24041 Vrooli CLI calls"}},
				Subs:   []*measurepb.UsageToolSub{{Kind: "failure", Signature: "identity broker did not answer", Calls: 30}, {Kind: "degraded", Signature: "identity_refresh_unavailable", Calls: 10}},
			},
			{
				Fingerprint: "g0e550", Kind: "shape-guessing", Owner: "agent-manager", CommandPath: "agent-manager", Description: "Agents guess how to call `agent-manager`", Calls: 12, Runs: 6, CostUnit: "calls", Cost: 12, CostShare: 0.03, Status: "open",
				Guesses: []*measurepb.UsageToolGuess{{Token: "runs", Suggestion: "run", Calls: 8}, {Token: "--tail", Calls: 4}},
			},
			{Fingerprint: "bypa55", Kind: "bypass", Owner: "agent-manager", Description: "Agents reach around agent-manager's CLI: 968 database in 4 runs; 62% of it in one run, so its cost counts that run at most 10x the median run", Calls: 968, Runs: 4, CostUnit: "calls", Cost: 130, CostShare: 0.02, TopRunShare: 0.62, Status: "open"},
		},
		Interventions: f.interventions,
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
		GeneratedAt: "2026-10-29T06:00:00Z", BelowMateriality: 7, MaterialityScore: 0.01,
		Actions: []*measurepb.UsageAction{
			{
				Rank: 1, Source: "lever", Kind: "R-context", OwnerKind: "harness", Owner: "claude-code session context", Title: "Compact claude-code sessions after ~60 requests of growth",
				Action: "Compact claude-code sessions about every 60 requests", Rationale: "70% of claude-code session quota was requests above 200k tokens.", Confidence: "medium",
				Effect: &measurepb.UsageActionEffect{Value: 9.2, Unit: "points", Basis: "projected"}, Score: 0.142, ScoreBasis: "quota scale: 14.2% (9.2 pts) of the anthropic 10080m window's 65.0 observed points", RecoverablePoints: 1.5,
				Drilldown:    []*measurepb.UsageActionDrill{{Kind: "session", Id: "s1", Label: "research orchestrator", Detail: "would have saved ≈6.0 pts (finished); compacted 47×"}},
				EvidenceRefs: []*measurepb.UsageEvidenceRef{{Kind: "run", Id: "a"}, {Kind: "run", Id: "b"}}, Verify: "agent-manager usage compare …",
			},
			{
				Rank: 2, Source: "lever", Kind: "R-model-mix", OwnerKind: "harness", Owner: "claude-code sub-agent model", Title: "Try sonnet for claude-code sub-agents", Action: "Run lighter sub-agent work on sonnet",
				Confidence: "unproven", Effect: &measurepb.UsageActionEffect{Upper: 4.4, Unit: "points", Basis: "range"}, ScoreBasis: "0–6.8% of the anthropic 10080m window's 65.0 observed points", Verify: "compare",
			},
			{
				Rank: 3, Source: "tool", Kind: "failure", OwnerKind: "command", Owner: "test-genie runs wait", Title: "Fix failing `test-genie runs wait`", Action: "Fix `test-genie runs wait`: it failed 161 times",
				Rationale: "161 failures.", Confidence: "high", Effect: &measurepb.UsageActionEffect{Value: 161, Unit: "calls", Basis: "observed"}, Score: 0.075, ScoreBasis: "7.5% of the window's 2149 failed calls",
				EvidenceRefs: []*measurepb.UsageEvidenceRef{{Kind: "run", Id: "a"}}, Verify: "usage friction should no longer list it",
			},
			{
				Rank: 4, Source: "tool", Kind: "failure", OwnerKind: "scenario", Owner: "vrooli", Title: "Fix the identity root cause", Action: "Fix the identity broker or refresh that 2 lines trace to", Rationale: "40 calls.", Confidence: "high",
				Effect: &measurepb.UsageActionEffect{Value: 40, Unit: "calls", Basis: "observed"}, Score: 0.2, ScoreBasis: "20.0%; 62% of it in one run, so the effect counts each run at most 10x the median run", RootCause: "identity", TopRunShare: 0.62,
				Subs: []*measurepb.UsageToolSub{{Kind: "failure", Signature: "identity broker did not answer", Calls: 30}, {Kind: "degraded", Signature: "identity_refresh_unavailable", Calls: 10}},
			},
			{
				Rank: 5, Source: "tool", Kind: "shape-guessing", OwnerKind: "command", Owner: "agent-manager", Title: "Make `agent-manager` easier to call", Action: "add alias `runs`→`run` (x8)", Confidence: "high",
				Effect: &measurepb.UsageActionEffect{Value: 12, Unit: "calls", Basis: "observed"}, Guesses: []*measurepb.UsageToolGuess{{Token: "runs", Suggestion: "run", Calls: 8}, {Token: "--tail", Calls: 4}},
			},
			{
				Rank: 6, Source: "tool", Kind: "slow", OwnerKind: "command", Owner: "agent-manager run status", Title: "Make `agent-manager run status` faster", Action: "Make `agent-manager run status` faster or give it a cheaper status/summary", Confidence: "high",
				Effect: &measurepb.UsageActionEffect{Value: 720_000, Unit: "ms", Basis: "observed"},
			},
		},
		Rules:    []*measurepb.UsageRuleStatus{{Rule: "R-context", Fired: true, Rows: 1}, {Rule: "R-model", Reason: "no workload ran on two models"}},
		Coverage: []string{"operator interventions: 300 messages classified (90% coverage); no category has passed the accuracy gate, so none is recommended: cannot tell, not none"},
		Scoring:  []string{"assumption: kinds of harm weigh equally"},
	}), nil
}

func (f *attributionFixture) UsageCoverage(_ context.Context, req *connect.Request[measurepb.UsageCoverageRequest]) (*connect.Response[measurepb.UsageCoverageResponse], error) {
	f.coverage = req.Msg
	root := &measurepb.UsageCoverageRoot{Kind: "codex-home", Harness: "codex", Files: 40, Indexed: 38, Pending: 1, Unreadable: 1, Bytes: 1000, CapturedBytes: 970, CapturedShare: 0.97, Turns: 900, UsageReports: 2000, Issues: []*measurepb.UsageCoverageIssue{{Path: "/home/x/rollout.jsonl", Reason: "not indexed yet"}}}
	return connect.NewResponse(&measurepb.UsageCoverageResponse{Since: "2026-10-22T06:00:00Z", Roots: []*measurepb.UsageCoverageRoot{root}, Totals: root, Method: "Independent scan: …"}), nil
}

func TestCoverageShareDoesNotPresentEmptyDenominatorAsComplete(t *testing.T) {
	if got := coverageShare(&measurepb.UsageCoverageRoot{CapturedShare: 1}); got != "n/a" {
		t.Fatalf("empty denominator = %q", got)
	}
	if got := coverageShare(&measurepb.UsageCoverageRoot{Bytes: 100, CapturedShare: 0.97}); got != "97.0%" {
		t.Fatalf("nonempty denominator = %q", got)
	}
}

func (f *attributionFixture) UsageCompare(_ context.Context, req *connect.Request[measurepb.UsageCompareRequest]) (*connect.Response[measurepb.UsageCompareResponse], error) {
	f.compare = req.Msg
	return connect.NewResponse(&measurepb.UsageCompareResponse{
		Verdict: "improved", PointsPerAcceptedChange: -0.4,
		Left:    &measurepb.UsageCompareSide{Runs: 10, CompletedRuns: 10, AcceptedUnits: 8, UnknownUnits: 2, PricedRuns: 10, Points: 4, PointsPerAccepted: 0.5},
		Right:   &measurepb.UsageCompareSide{Runs: 10, CompletedRuns: 10, AcceptedUnits: 10, PricedRuns: 10, Points: 3, PointsPerAccepted: 0.3},
		Caveats: []string{"Cohorts are observational."},
	}), nil
}

func (f *attributionFixture) UsageParity(context.Context, *connect.Request[measurepb.UsageParityRequest]) (*connect.Response[measurepb.UsageParityResponse], error) {
	return connect.NewResponse(&measurepb.UsageParityResponse{BackfillRuns: 7, LedgerPending: 2,
		Rows: []*measurepb.UsageParityRow{{Path: "imported", Harness: "codex", Runs: 10, EligibleRuns: 8, NoRequests: 2, Excluded: []*measurepb.UsageParityCount{{Name: "helper", Count: 1}},
			Measures: []*measurepb.UsageParityMeasure{{Name: "tool_facts", Numerator: 4, Denominator: 8, Share: 0.5, NotApplicable: 2, Reason: "left recall"}}}},
		Gaps: []*measurepb.UsageParityGap{{Subject: "grok transcripts", Status: "open", Count: 3, Reason: "no reader"}},
	}), nil
}

func (f *attributionFixture) UsageOutcomes(_ context.Context, req *connect.Request[measurepb.UsageOutcomesRequest]) (*connect.Response[measurepb.UsageOutcomesResponse], error) {
	f.outcomes = req.Msg
	return connect.NewResponse(&measurepb.UsageOutcomesResponse{By: req.Msg.GetBy(), Units: 10, UnknownUnits: 5, Evidence: []*measurepb.UsageEvidenceCount{{Source: "validation", Units: 5}},
		Rows: []*measurepb.UsageOutcomeRow{
			{Key: "luna", Runner: "codex", Verdict: "cheaper", Baseline: "terra", Units: 5, Accepted: 5, BreakEvenRatio: 5, PriceSource: "tier_prior",
				CheaperBy:         &measurepb.UsageRange{Available: true, Value: 1.67, Low: 1.5, High: 1.8},
				PointsPerAccepted: &measurepb.UsageRange{Available: true, Value: 3, Low: 2, High: 4}},
			{Key: "rare", Runner: "codex", Verdict: "not_enough_data", StatusReason: "2 accepted outcomes; need 5", Units: 2, Accepted: 2},
		},
		Refit: []*measurepb.UsageRefitRow{{Runner: "codex", Model: "luna", Status: "insufficient", Reason: "2 blocks dominated"}},
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

func TestUsageRecommendationsShowsOneRankedOwnerLevelList(t *testing.T) {
	fixture := &attributionFixture{}
	out := runUsageCommand(t, fixture, recommendationsCommand, "--run", "a", "--provider", "openai", "--pool", "primary", "--window", "10080m")
	if r := fixture.recs; r.GetRunId() != "a" || r.GetProvider() != "openai" || r.GetPool() != "primary" || r.GetWindow() != "10080m" {
		t.Fatalf("request = %v", r)
	}
	if n := strings.Count(out, "Quota levers ("); n != 1 {
		t.Errorf("the quota levers are one group, headed once: %d headings", n)
	}
	assertContains(t, out,
		"6 action(s) as of Thu Oct 29 06:00 UTC: quota levers ranked by their share of a window's points, then friction actions ranked by √(harm share × runs touched); the scales are not comparable; 7 more scoring under 1%",
		"Actions:\n  Quota levers (quota scale: share of a window's observed points)\n  1. Compact",
		"verify: compare\n  Friction actions (friction scale: √(harm share × runs touched))\n  3. Fix failing",
		"1. Compact claude-code sessions after ~60 requests of growth  14.2%  [lever R-context, medium]",
		"score: quota scale: 14.2% (9.2 pts) of the anthropic 10080m window's 65.0 observed points",
		"owner: harness claude-code session context",
		"effect: 9.2 pts (projected) · 70% of claude-code session quota",
		`root cause "identity" rolls up 2 lines:`,
		"degraded in 10 calls  identity_refresh_unavailable",
		"score: 20.0%; 62% of it in one run, so the effect counts each run at most 10x the median run",
		"guessed `runs` x8: add alias `runs`→`run`",
		"Make `agent-manager run status` faster or give it a cheaper status/summary",
		"effect: 12 min (observed)",
		"≈1.5 pts of it still recoverable before the window resets",
		"· research orchestrator: would have saved ≈6.0 pts (finished); compacted 47×",
		"evidence: 2 run · verify: agent-manager usage compare …",
		"3. Fix failing `test-genie runs wait`  7.5%  [tool failure, high]",
		"effect: 161 calls (observed)",
		"2. Try sonnet for claude-code sub-agents  unproven range  [lever R-model-mix, unproven]",
		"effect: 0 to 4.4 pts (unproven range)",
		"R-model did not fire: no workload ran on two models",
		"coverage: operator interventions: 300 messages classified",
		"assumption: kinds of harm weigh equally",
	)
	if strings.Contains(out, "projected −") || strings.Contains(out, "Dominant consumer") {
		t.Fatalf("per-session post-mortem rendering survived:\n%s", out)
	}
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
	assertContains(t, out, "Verdict: improved (points per accepted unit -40%)", "left  10 runs (10 completed), 8 accepted units, 2 unknown, 10 priced · ≈4.00 pts · ≈0.500 pts per accepted")
}

func TestUsageParityAndOutcomesPrintTheirEvidence(t *testing.T) {
	fixture := &attributionFixture{}
	out := runUsageCommand(t, fixture, parityCommand, "--window", "last_7d")
	assertContains(t, out, "imported / codex: 10 runs, 8 in denominators, 2 with no requests, 1 helper", "tool_facts       4/8 (50.0%), 2 not applicable: left recall", "[open] grok transcripts: 3. no reader", "7 old sessions imported in this window are excluded")
	out = runUsageCommand(t, fixture, outcomesCommand, "--by", "role")
	if fixture.outcomes.GetBy() != "role" {
		t.Fatalf("outcomes request = %v", fixture.outcomes)
	}
	assertContains(t, out, "codex / luna: cheaper", "cheaper by 1.67x (1.50x to 1.80x)", "break-even 5.00x (tier_prior)", "codex / rare: not_enough_data: 2 accepted outcomes; need 5", "5 unknown (50%)", "codex/luna: insufficient. 2 blocks dominated")
}

func TestUsageFrictionListsPlainProblemsWithOwnerCostAndStatus(t *testing.T) {
	fixture := &attributionFixture{interventions: &measurepb.UsageInterventions{
		OperatorMessages: 120, Classified: 108, Coverage: 0.9, AgentHours: 40,
		UnshownReason: "no category has passed the accuracy gate yet, so the classified messages are not shown",
	}}
	out := runUsageCommand(t, fixture, frictionCommand, "--status", "open", "--limit", "5", "--since", "2026-10-22T06:00:00Z")
	if f := fixture.friction; f.GetStatus() != "open" || f.GetLimit() != 5 || f.GetSince() != "2026-10-22T06:00:00Z" {
		t.Fatalf("request = %v", f)
	}
	assertContains(t, out,
		"2 open · 1 reported · 0 dismissed as of Thu Oct 29 06:00 UTC; tool window since Thu Oct 22 06:00 UTC",
		"Listed 7; 371 more under the materiality floor (score 1%), 3 more past --limit",
		"Coverage: 175706 commands, outcome known for 40% (24041 Vrooli CLI calls, 35% known)",
		"a1b2c3  [failure] `prompt-manager team bug-capture`, `search-hub query` and 6 other commands failed 128 times in 20 runs with: Error: agent identity identity_refresh_unavailable",
		"owner vrooli · 128 calls in 20 runs · score 0.0% · cost failure 128 calls = 15.8% of the window's 812 Vrooli CLI failed calls · open",
		"cost failure 30 calls = 3.7% of the window's 812 Vrooli CLI failed calls; degraded 10 calls = 0.0% of the window's 24041 Vrooli CLI calls · open",
		"e.g. search-hub query notes",
		"[slow] `agent-manager run status` should answer in under 5s",
		"cost slow 12m 00s = 5.0% of the window's Vrooli CLI time (4h 00m)",
		`root cause "identity" rolls up 2 lines:`,
		"failure in 30 calls  identity broker did not answer",
		"degraded in 10 calls  identity_refresh_unavailable",
		"guessed `runs` x8: add alias `runs`→`run`",
		"guessed `--tail` x4: fix the help",
		"62% of it in one run, so its cost counts that run at most 10x the median run",
		"largest call printed ~262k tokens; the model read 10k",
		"owner agent-manager · 161 calls in 34 runs · score 0.0% · cost degraded 161 calls = 0.7% of the window's 24041 Vrooli CLI calls · reported knw-9",
		"cost output-weight 3.2M tokens = 1.0% of the window's context tax (320M re-read tokens)",
		"Operator interventions: 120 operator messages, 90% classified over 40.0 agent-hours",
		"none shown: no category has passed the accuracy gate yet, so the classified messages are not shown (this is not \"no friction\")",
	)
	if strings.Contains(out, "of the window's command calls") {
		t.Fatalf("a share is labelled with a base it was not divided by:\n%s", out)
	}
	if strings.Contains(out, "Re-analysing") || strings.Contains(out, "operator correction") {
		t.Fatalf("output still speaks of the deleted episode path:\n%s", out)
	}
}

// A category the gate admits is counted with its owner, rate, preceding agent
// work and redacted examples linking to their run and the report they repeat.
func TestUsageFrictionShowsAdmittedInterventionsWithOwnerWorkAndExamples(t *testing.T) {
	fixture := &attributionFixture{interventions: &measurepb.UsageInterventions{
		OperatorMessages: 120, Classified: 108, Coverage: 0.9, AgentHours: 40,
		Categories: []*measurepb.UsageInterventionCategory{{Category: "quality_miss", Owner: "proof before claim: scientific-debugging", Interventions: 20, PerAgentHour: 0.5, Repeats: 3, WorkRequests: 80, WorkToolCalls: 55, WorkTokens: 1_200_000}},
		Examples:   []*measurepb.UsageInterventionExample{{Category: "quality_miss", RunId: "run-7", Text: "Still broken on mobile", RepeatOfEventId: "evt-1"}},
	}}
	out := runUsageCommand(t, fixture, frictionCommand)
	assertContains(t, out,
		"quality_miss · 20 (0.50 per agent-hour) · owner proof before claim: scientific-debugging · agent work before: 80 requests, 55 tool calls, 1.2M tokens · 3 repeat reports",
		`e.g. [quality_miss] "Still broken on mobile"  run run-7 (agent-manager run report run-7) · repeats evt-1`,
	)
	if strings.Contains(out, "none shown") {
		t.Fatalf("an admitted category must not print the nothing-shown line:\n%s", out)
	}
}

type frictionFixture struct {
	domainconnect.UnimplementedFrictionServiceHandler
	reported  *domainpb.ReportFrictionRequest
	dismissed *domainpb.DismissFrictionRequest
}

func (f *frictionFixture) ReportFriction(_ context.Context, req *connect.Request[domainpb.ReportFrictionRequest]) (*connect.Response[domainpb.ReportFrictionResponse], error) {
	f.reported = req.Msg
	return connect.NewResponse(&domainpb.ReportFrictionResponse{Fingerprint: req.Msg.GetFingerprint(), Description: "Agents run `rg` repeatedly.", Status: "reported", BugId: "knw-7"}), nil
}

func (f *frictionFixture) DismissFriction(_ context.Context, req *connect.Request[domainpb.DismissFrictionRequest]) (*connect.Response[domainpb.DismissFrictionResponse], error) {
	f.dismissed = req.Msg
	return connect.NewResponse(&domainpb.DismissFrictionResponse{Fingerprint: req.Msg.GetFingerprint(), Status: "dismissed"}), nil
}

func TestUsageFrictionReportAndDismissWriteThroughTheFrictionService(t *testing.T) {
	fixture := &frictionFixture{}
	path, handler := domainconnect.NewFrictionServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := func(*cliapp.ScenarioApp) domainconnect.FrictionServiceClient {
		return domainconnect.NewFrictionServiceClient(server.Client(), server.URL)
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
	assertContains(t, run(frictionReportCommand(client), "4454d97b", "--since", "2026-10-01T00:00:00Z"), "Reported as knw-7: Agents run `rg` repeatedly.")
	if fixture.reported.GetFingerprint() != "4454d97b" || fixture.reported.GetSince() != "2026-10-01T00:00:00Z" {
		t.Fatalf("report request = %v", fixture.reported)
	}
	assertContains(t, run(frictionDismissCommand(client), "4454d97b", "--since", "2026-10-01T00:00:00Z"), "Dismissed 4454d97b")
	if fixture.dismissed.GetFingerprint() != "4454d97b" || fixture.dismissed.GetSince() != "2026-10-01T00:00:00Z" {
		t.Fatalf("dismiss request = %v", fixture.dismissed)
	}
	var decoded struct {
		BugID string `json:"bug_id"`
	}
	if err := json.Unmarshal([]byte(run(frictionReportCommand(client), "4454d97b", "--json")), &decoded); err != nil || decoded.BugID != "knw-7" {
		t.Fatalf("--json must be the wire shape: %v %+v", err, decoded)
	}
}

// The pooled bucket prints as "category unproven" with each hint labelled unproven.
func TestUsageFrictionLabelsThePooledBucketAndItsHintsUnproven(t *testing.T) {
	fixture := &attributionFixture{interventions: &measurepb.UsageInterventions{
		OperatorMessages: 120, Classified: 108, Coverage: 0.9, AgentHours: 40,
		Categories: []*measurepb.UsageInterventionCategory{{Category: "operator_friction", Pooled: true, Interventions: 18, PerAgentHour: 0.45, Hints: []*measurepb.UsageInterventionHint{{Category: "quality_miss", Owner: "scientific-debugging", Count: 11}}}},
		Examples:   []*measurepb.UsageInterventionExample{{Category: "quality_miss", Pooled: true, RunId: "run-7", Text: "Still broken"}},
	}}
	out := runUsageCommand(t, fixture, frictionCommand)
	assertContains(t, out,
		"operator friction (category unproven) · 18 (0.45 per agent-hour) · owners by hint",
		"hint (unproven): 11 look like quality_miss · owner scientific-debugging",
		`e.g. [quality_miss, unproven] "Still broken"`,
	)
}
