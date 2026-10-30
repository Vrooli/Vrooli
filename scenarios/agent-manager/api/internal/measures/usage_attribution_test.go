package measures

import (
	"context"
	"fmt"
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

// usageFacts serves a day of usage for two sessions: an orchestrator whose
// re-read context grows from 60k to about 300k per request and a short helper run.
type usageFacts struct {
	now       time.Time
	staleRuns int
}

func (u usageFacts) UsageTurns(_ context.Context, from, to time.Time) ([]usageattribution.Turn, error) {
	var turns []usageattribution.Turn
	for i, at := 0, u.now.Add(-24*time.Hour); at.Before(u.now); i, at = i+1, at.Add(10*time.Minute) {
		if at.Before(from) || at.After(to) {
			continue
		}
		cached := 3 * (60_000 + 1_700*i)
		turns = append(turns,
			usageattribution.Turn{At: at, RunID: "orchestrator", Runner: "codex", Model: "gpt-6.1-terra", Reports: 3, Tokens: domain.UsageCounters{InputTokens: 6000, CacheReadTokens: cached, OutputTokens: 2400}, Facts: transcriptindex.TurnFacts{ToolCalls: 3, ToolKinds: map[string]int{"shell": 3}}},
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

// Five episodes of one fingerprint: `rg` fails twice in a row.
func (u usageFacts) UsageEpisodes(context.Context, time.Time, time.Time) ([]usageattribution.Episode, error) {
	var episodes []usageattribution.Episode
	for i := 0; i < 5; i++ {
		episodes = append(episodes, usageattribution.Episode{
			RunID: "orchestrator", EpisodeID: fmt.Sprintf("ep-%d", i), At: u.now.Add(-time.Duration(i+1) * time.Hour), Pattern: "command-failure", Fingerprint: "fp-rg",
			OwnerScenario: "rg", OwnerCommand: "rg", WallClockMS: 20_000, Turns: 6, Tokens: 90_000,
		})
	}
	return episodes, nil
}

// All five analysed runs of the day are the orchestrator, analysed by the
// current classifier except while a test sets staleRuns.
func (u usageFacts) UsageEpisodeCoverage(_ context.Context, runIDs []string) (int, error) {
	return len(runIDs) - u.staleRuns, nil
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
	h.SetUsageAttribution(usageFacts{now: now}, nil, &staticCoverage{}, &memoryStatuses{})
	return h
}

// memoryStatuses and bugFiler are the in-memory seams of the friction actions.
type memoryStatuses struct {
	byFingerprint map[string]usageattribution.ProblemStatus
}

func (m *memoryStatuses) FrictionStatuses(context.Context) (map[string]usageattribution.ProblemStatus, error) {
	return m.byFingerprint, nil
}

func (m *memoryStatuses) SetFrictionStatus(_ context.Context, status usageattribution.ProblemStatus) error {
	if m.byFingerprint == nil {
		m.byFingerprint = map[string]usageattribution.ProblemStatus{}
	}
	m.byFingerprint[status.Fingerprint] = status
	return nil
}

type bugFiler struct{ filed []usageattribution.Problem }

func (b *bugFiler) FileFrictionBug(_ context.Context, p usageattribution.Problem) (string, error) {
	b.filed = append(b.filed, p)
	return fmt.Sprintf("knw-%d", len(b.filed)), nil
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
			if burner.GetPoints() < minBurnerPoints {
				t.Errorf("%s: row %s has %.3f points; rows under %.1f fold into other", by, burner.GetLabel(), burner.GetPoints(), minBurnerPoints)
			}
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
	top := response.Msg.GetGroups()[0]
	lever := top.GetLevers()[0]
	if top.GetTargetId() != "orchestrator" || top.GetRank() != 1 || lever.GetRule() != usageattribution.RuleContext || lever.GetFacts().GetContext().GetRequests() == 0 || len(lever.GetEvidenceRefs()) == 0 {
		t.Fatalf("top group = %v", top)
	}
	if top.GetUsedSoFarPoints() <= 0 || top.GetAtThisRatePoints() < top.GetUsedSoFarPoints() || top.GetShareOfWindowBurn() <= 0 || !top.GetStillRunning() {
		t.Fatalf("top group states no use: %v", top)
	}
	if top.GetRecoverablePoints() <= 0 || top.GetRecoverablePoints() > top.GetCombinedSavedPoints() || top.GetRemainingHours() <= 0 || top.GetCrossCut() {
		t.Fatalf("a running session recovers part of its saving: %v", top)
	}
	var sum float64
	for _, l := range top.GetLevers() {
		sum += l.GetSavedPoints()
	}
	if top.GetCombinedSavedPoints() <= 0 || top.GetCombinedSavedPoints() > sum+1e-9 {
		t.Fatalf("combined saving %.3f of levers %.3f", top.GetCombinedSavedPoints(), sum)
	}
	filtered, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{RunId: "nobody"}))
	if err != nil || len(filtered.Msg.GetGroups()) != 0 {
		t.Fatalf("run filter = %v, %v", filtered.Msg.GetGroups(), err)
	}
	if _, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{Pool: "primary"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a pool without provider and window must be rejected, got %v", err)
	}
}

// P2-A5: the friction list describes each problem in plain words with owner,
// count and cost, and reporting files one bug and records its id; reporting
// again files nothing; dismissing keeps the bug id and drops the problem from
// recommendations.
func TestUsageFrictionListsReportsAndDismissesProblems(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	list := func(status string) *measurepb.UsageFrictionResponse {
		t.Helper()
		response, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{Status: status}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg
	}
	open := list("")
	if len(open.GetProblems()) != 1 || open.GetOpen() != 1 {
		t.Fatalf("problems = %v", open)
	}
	p := open.GetProblems()[0]
	if p.GetDescription() != "`rg` failed." || p.GetOwner() != "rg" || p.GetOwnerKind() != "command" ||
		p.GetEpisodes() != 5 || p.GetRuns() != 1 || p.GetWallClockMs() != 100_000 || p.GetTypicalWallClockMs() != 20_000 || p.GetPoints() <= 0 || p.GetStatus() != "open" {
		t.Fatalf("problem = %v", p)
	}
	if _, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{Status: "wip"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid status error = %v", err)
	}

	bugs := &bugFiler{}
	service, err := h.usageService()
	if err != nil {
		t.Fatal(err)
	}
	service.Bugs = bugs
	for range 2 {
		reported, err := service.ReportFriction(context.Background(), p.GetFingerprint(), now)
		if err != nil || reported.Status != usageattribution.StatusReported || reported.BugID != "knw-1" {
			t.Fatalf("report = %+v, %v", reported, err)
		}
	}
	if len(bugs.filed) != 1 {
		t.Fatalf("bugs filed = %d, want one however often it is reported", len(bugs.filed))
	}
	if got := list("reported"); len(got.GetProblems()) != 1 || got.GetProblems()[0].GetBugId() != "knw-1" || got.GetReported() != 1 || got.GetOpen() != 0 {
		t.Fatalf("reported list = %v", got)
	}
	if _, err := service.ReportFriction(context.Background(), "nope", now); err == nil {
		t.Fatal("reporting an unknown fingerprint must fail")
	}
	if err := service.DismissFriction(context.Background(), p.GetFingerprint(), now); err != nil {
		t.Fatal(err)
	}
	if got := list("dismissed"); len(got.GetProblems()) != 1 || got.GetProblems()[0].GetBugId() != "knw-1" {
		t.Fatalf("dismissed list = %v", got)
	}
	response, err := h.UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range response.Msg.GetGroups() {
		if group.GetTargetKind() == "friction" {
			t.Fatalf("dismissed problem recommended: %v", group)
		}
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
	for _, name := range []string{UsageForecast, UsageForecastBacktest, UsageBurners, UsageRecommendations, UsageFriction, UsageCoverage, UsageCompare} {
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

// m4: the friction response says how many of the window's runs the current
// classifier has analysed, so the UI can say it is still re-analysing.
func TestUsageFrictionReportsReprojectionProgress(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := NewHandler(&fakeStore{}, func() time.Time { return now })
	h.SetUsageForecast(&quotaReadings{spans: codexReadings(now)})
	h.SetUsageAttribution(usageFacts{now: now, staleRuns: 1}, nil, &staticCoverage{}, &memoryStatuses{})
	response, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetRunsTotal() != 2 || response.Msg.GetRunsAnalysed() != 1 {
		t.Fatalf("analysed %d of %d runs, want 1 of 2", response.Msg.GetRunsAnalysed(), response.Msg.GetRunsTotal())
	}
	if p := response.Msg.GetProblems()[0]; p.GetTimeKind() != "redundant" {
		t.Fatalf("command-failure prices redundant call time: %v", p)
	}
}

// m5: only a problem in the current list can be dismissed.
func TestDismissFrictionRejectsAnUnknownFingerprint(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	service, err := h.usageService()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DismissFriction(context.Background(), "made-up", now); err == nil {
		t.Fatal("dismissing a fingerprint no current problem has must fail")
	}
	statuses, _ := h.frictionStatuses.FrictionStatuses(context.Background())
	if len(statuses) != 0 {
		t.Fatalf("a rejected dismissal wrote %v", statuses)
	}
	if err := service.DismissFriction(context.Background(), "fp-rg", now); err != nil {
		t.Fatalf("dismissing a listed problem: %v", err)
	}
}
