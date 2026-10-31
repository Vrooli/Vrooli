package measures

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// usageFacts serves a day of usage for two sessions: an orchestrator whose
// re-read context grows from 60k to about 300k per request and a short helper run.
type usageFacts struct {
	now time.Time
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
		out[id] = usageattribution.Run{ID: id, Label: id, Session: id, SessionLabel: id, Runner: "codex", Source: usageattribution.SourceAgentManager, Completed: true, Accepted: true, WorkUnit: id}
	}
	return out, nil
}

// ToolCalls: three Vrooli CLIs fail with one identity error, and one ls.
func (u usageFacts) ToolCalls(_ context.Context, from, to time.Time) (toolfriction.Window, error) {
	var sources []toolfriction.Source
	for i, command := range []string{"search-hub query notes", "prompt-manager skill read x", "audio-tools settings provider", "ls docs"} {
		exit, output := 1, "Error: agent identity identity_refresh_unavailable; check run/session binding"
		if command == "ls docs" {
			exit, output = 0, "a"
		}
		at := u.now.Add(-time.Duration(i+1) * time.Minute)
		if at.Before(from) || !at.Before(to) {
			continue
		}
		call := &domain.ToolCallEventData{ToolName: "Bash", ToolCallID: fmt.Sprintf("c%d", i), Input: map[string]any{"command": command}}
		result := &domain.ToolResultEventData{ToolName: "Bash", ToolCallID: call.ToolCallID, Output: output, ExitCode: &exit}
		events := []*domain.RunEvent{{ID: uuid.New(), EventType: domain.EventTypeToolCall, Timestamp: at, Data: call}, {ID: uuid.New(), EventType: domain.EventTypeToolResult, Timestamp: at, Data: result}}
		for _, fact := range runsignal.DeriveInvocationFacts(events) {
			sources = append(sources, toolfriction.Source{Fact: toolfriction.Fact{InvocationFact: fact, RunID: fmt.Sprintf("run-%d", i), At: at, Runner: "codex"}, Call: call, Result: result})
		}
	}
	features := toolfriction.Extract(sources, nil)
	calls := make([]toolfriction.Call, len(sources))
	for i, source := range sources {
		calls[i] = toolfriction.Observe(source.Fact, features[i])
	}
	return toolfriction.Window{Calls: calls}, nil
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

type bugFiler struct {
	filed []usageattribution.FrictionBug
}

func (b *bugFiler) FileFrictionBug(_ context.Context, bug usageattribution.FrictionBug) (string, error) {
	b.filed = append(b.filed, bug)
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
		if w.GetFoldBelowPoints() != minBurnerPoints {
			t.Errorf("%s: fold threshold %v, want the API's %v so the page states the rule it applied", by, w.GetFoldBelowPoints(), minBurnerPoints)
		}
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

// P4-A6 at the procedure boundary: one ranked list of owner-level actions,
// each with owner, action, effect in its own unit, score, evidence and a
// verifying measure, beside every rule status, the conversions and coverage.
func TestUsageRecommendationsListRankedOwnerLevelActions(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	response, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	msg := response.Msg
	if len(msg.GetRules()) != len(usageattribution.Rules) || len(msg.GetScoring()) == 0 || len(msg.GetCoverage()) == 0 || msg.GetMaterialityScore() != toolfriction.MaterialScore {
		t.Fatalf("rules = %d scoring = %v coverage = %v", len(msg.GetRules()), msg.GetScoring(), msg.GetCoverage())
	}
	if len(msg.GetActions()) == 0 {
		t.Fatal("no actions")
	}
	for i, a := range msg.GetActions() {
		// Quota levers first, then friction actions, each by score on its own scale.
		lever := a.GetSource() == usageattribution.SourceLever
		if i > 0 {
			previous := msg.GetActions()[i-1]
			if lever && previous.GetSource() != usageattribution.SourceLever || lever == (previous.GetSource() == usageattribution.SourceLever) && a.GetScore() > previous.GetScore() {
				t.Errorf("rank %d out of order: %v", i+1, a)
			}
		}
		if a.GetRank() != int32(i+1) || !strings.HasPrefix(a.GetScoreBasis(), map[bool]string{true: usageattribution.ScaleQuota, false: usageattribution.ScaleFriction}[lever]+": ") {
			t.Errorf("rank %d or its scale: %v", i+1, a)
		}
		if a.GetOwner() == "" || a.GetOwnerKind() == "" || a.GetAction() == "" || a.GetVerify() == "" || a.GetScoreBasis() == "" || a.GetEffect().GetUnit() == "" || len(a.GetTitle()) == 0 || len([]rune(a.GetTitle())) > 60 {
			t.Errorf("incomplete action %v", a)
		}
	}
	if msg.GetOutcome() != nil {
		t.Fatalf("an unscoped list projects no window: %v", msg.GetOutcome())
	}
	var lever *measurepb.UsageAction
	for _, a := range msg.GetActions() {
		if a.GetSource() == usageattribution.SourceLever && a.GetRecoverablePoints() > 0 {
			lever = a
			break
		}
	}
	if lever == nil {
		t.Fatal("the fixture has no lever with recoverable points")
	}
	scoped, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{Provider: lever.GetProvider(), Pool: lever.GetPool(), Window: lever.GetWindow()}))
	if err != nil {
		t.Fatal(err)
	}
	var largest, sum float64
	for _, a := range scoped.Msg.GetActions() {
		if a.GetSource() == usageattribution.SourceLever && a.GetRecoverablePoints() > 0 {
			largest, sum = max(largest, a.GetRecoverablePoints()), sum+a.GetRecoverablePoints()
		}
	}
	// The API combines the window's levers once; the page renders it.
	if o := scoped.Msg.GetOutcome(); o == nil || o.GetLevers() == 0 || o.GetRecoverablePoints() < largest-1e-9 || o.GetRecoverablePoints() > sum+1e-9 {
		t.Fatalf("scoped outcome = %v, want the levers combined between the largest %.2f and their sum %.2f", o, largest, sum)
	}
	filtered, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{RunId: "nobody"}))
	if err != nil || len(filtered.Msg.GetActions()) != 0 {
		t.Fatalf("run filter = %v, %v", filtered.Msg.GetActions(), err)
	}
	if _, err := usageHandler(now).UsageRecommendations(context.Background(), connect.NewRequest(&measurepb.UsageRecommendationsRequest{Pool: "primary"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a pool without provider and window must be rejected, got %v", err)
	}
}

// P2-A5 and P4-A2: the friction list carries owner-level tool findings (with
// the coverage that says what the profiles could see), each with owner, count
// and cost; reporting files one bug and records its id; reporting again files
// nothing; dismissing keeps the bug id.
func TestUsageFrictionListsReportsAndDismissesFindings(t *testing.T) {
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
	if len(open.GetToolFindings()) != 1 || open.GetOpen() != 1 || !strings.Contains(open.GetCoverage().GetSentence(), "4 commands, outcome known for 100%") || open.GetWindowStart() != rfc3339(now.Add(-7*24*time.Hour)) {
		t.Fatalf("friction = %v", open)
	}
	f := open.GetToolFindings()[0]
	// All three Vrooli-owned failures have this signature, so its share of the
	// Vrooli-owned failures (the denominator: a failed grep is nobody's to fix
	// here) is 1. The successful fourth command is not in it.
	if f.GetKind() != "failure" || f.GetOwner() != "vrooli" || f.GetOwnerKind() != "control-plane" || f.GetSignature() != "identity_refresh_unavailable" || f.GetCalls() != 3 || f.GetCostUnit() != "calls" || f.GetCostShare() != 1 || len(f.GetExampleCommands()) != 3 || len(f.GetEvidenceRefs()) != 3 {
		t.Fatalf("tool finding = %v", f)
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
		reported, err := service.ReportFriction(context.Background(), f.GetFingerprint(), time.Time{}, now)
		if err != nil || reported.Status != usageattribution.StatusReported || reported.BugID != "knw-1" {
			t.Fatalf("report = %+v, %v", reported, err)
		}
	}
	if len(bugs.filed) != 1 || bugs.filed[0].Context["owner"] != "vrooli" || bugs.filed[0].Fingerprint != f.GetFingerprint() {
		t.Fatalf("bugs filed = %+v, want one however often it is reported", bugs.filed)
	}
	if got := list("reported"); len(got.GetToolFindings()) != 1 || got.GetToolFindings()[0].GetBugId() != "knw-1" || got.GetReported() != 1 || got.GetOpen() != 0 {
		t.Fatalf("reported list = %v", got)
	}
	if _, err := service.ReportFriction(context.Background(), "nope", time.Time{}, now); err == nil {
		t.Fatal("reporting an unknown fingerprint must fail")
	}
	if err := service.DismissFriction(context.Background(), f.GetFingerprint(), time.Time{}, now); err != nil {
		t.Fatal(err)
	}
	if got := list("dismissed"); len(got.GetToolFindings()) != 1 || got.GetDismissed() != 1 {
		t.Fatalf("dismissed list = %v", got)
	}
}

// The tool friction RPC and the three registry measures it backs (rates,
// profiles, error signatures) are one computation; owner narrows it.
func TestToolFrictionServesRatesProfilesAndSignaturesFromOneComputation(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	rpc, err := h.ToolFriction(context.Background(), connect.NewRequest(&measurepb.ToolFrictionRequest{Window: token(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_7D)}))
	if err != nil {
		t.Fatal(err)
	}
	summary := rpc.Msg.GetSummary()
	if summary.GetCalls() != 4 || summary.GetFailedCalls() != 3 || summary.GetVrooliFailureRate() != 1 || summary.GetCoverage().GetOutcomeKnownShare() != 1 || len(rpc.Msg.GetProfiles()) != 4 || len(rpc.Msg.GetClusters()) != 1 || rpc.Msg.GetClusters()[0].GetOwner() != "vrooli" ||
		len(rpc.Msg.GetFindings()) != 1 || rpc.Msg.GetFindings()[0].GetStatus() != "open" || rpc.Msg.GetFindings()[0].GetSignature() != "identity_refresh_unavailable" {
		t.Fatalf("tool friction = %v", rpc.Msg)
	}
	narrowed, err := h.ToolFriction(context.Background(), connect.NewRequest(&measurepb.ToolFrictionRequest{Owner: "external"}))
	if err != nil || len(narrowed.Msg.GetProfiles()) != 1 || narrowed.Msg.GetProfiles()[0].GetCommandPath() != "ls" {
		t.Fatalf("external profiles = %v, %v", narrowed.Msg.GetProfiles(), err)
	}
	registry, err := h.Registry()
	if err != nil {
		t.Fatal(err)
	}
	for name, rows := range map[string]int{ToolFriction: 1, ToolProfiles: 4, ErrorSignatures: 1} {
		served, err := registry.Execute(context.Background(), measureRequest(name, "last_7d"))
		if err != nil || len(served.Fields) != rows {
			t.Fatalf("%s rows = %v, %v", name, served.Fields, err)
		}
		if name == ToolFriction && (served.Fields[0]["vrooli_failure_rate"] != "1" || served.Fields[0]["coverage.sentence"] == "") {
			t.Fatalf("summary row = %v", served.Fields[0])
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
	if response.Msg.GetVerdict() != "insufficient" || response.Msg.GetLeft().GetRuns() != 1 || response.Msg.GetLeft().GetAcceptedUnits() != 1 || len(response.Msg.GetCaveats()) < 3 {
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

// m5: only a finding in the current list can be dismissed.
func TestDismissFrictionRejectsAnUnknownFingerprint(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	service, err := h.usageService()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DismissFriction(context.Background(), "made-up", time.Time{}, now); err == nil {
		t.Fatal("dismissing a fingerprint no current finding has must fail")
	}
	statuses, _ := h.frictionStatuses.FrictionStatuses(context.Background())
	if len(statuses) != 0 {
		t.Fatalf("a rejected dismissal wrote %v", statuses)
	}
}

// Findings and actions carry the tool-friction semantics to the wire: the
// roll-up's lines, the guesses, the top run's share and printed-vs-read tokens.
func TestFindingAndActionMessagesCarryFrictionSemantics(t *testing.T) {
	f := protoToolFinding(usageattribution.ToolFinding{Finding: toolfriction.Finding{
		Kind: toolfriction.KindSlow, TopRunShare: 0.62, RootCause: "identity", PrintedTokens: 262_144, ReadTokens: 10_000,
		Subs:    []toolfriction.SubSignature{{Kind: "failure", Signature: "a", Calls: 3}, {Kind: "degraded", Signature: "b", Calls: 1}},
		Guesses: []toolfriction.Guess{{Token: "runs", Suggestion: "run", Calls: 8}},
	}})
	if f.GetTopRunShare() != 0.62 || f.GetRootCause() != "identity" || len(f.GetSubs()) != 2 || f.GetSubs()[1].GetKind() != "degraded" || f.GetGuesses()[0].GetSuggestion() != "run" || f.GetPrintedTokens() != 262_144 || f.GetReadTokens() != 10_000 {
		t.Errorf("finding = %v", f)
	}
	a := protoAction(usageattribution.Action{Kind: toolfriction.KindSlow, TopRunShare: 0.7, RootCause: "identity", Subs: []toolfriction.SubSignature{{Signature: "a", Calls: 2}}, Guesses: []toolfriction.Guess{{Token: "x", Calls: 1}}})
	if a.GetKind() != "slow" || a.GetTopRunShare() != 0.7 || a.GetRootCause() != "identity" || len(a.GetSubs()) != 1 || len(a.GetGuesses()) != 1 {
		t.Errorf("action = %v", a)
	}
}

// manyRunFacts are 200 runs: 100 hit one failure three times each, one run
// hits another failure three times. The second is under 1% of the failures and
// in one of 200 runs: its score is under the floor.
type manyRunFacts struct{ usageFacts }

func (m manyRunFacts) ToolCalls(_ context.Context, from, to time.Time) (toolfriction.Window, error) {
	var calls []toolfriction.Call
	fail := func(run int, line string) toolfriction.Call {
		return toolfriction.Call{RunID: fmt.Sprintf("run-%03d", run), CallEventID: fmt.Sprintf("%d-%d", run, len(calls)), At: m.now.Add(-time.Hour), CommandPath: "search-hub query", Owner: "search-hub", Outcome: runsignal.OutcomeFailure, Features: toolfriction.Features{Signature: line, ErrorLine: line}}
	}
	for run := 0; run < 200; run++ {
		calls = append(calls, toolfriction.Call{RunID: fmt.Sprintf("run-%03d", run), At: m.now.Add(-time.Hour), CommandPath: "ls", Outcome: runsignal.OutcomeSuccess})
		for i := 0; run < 100 && i < 3; i++ {
			calls = append(calls, fail(run, "Error: index locked"))
		}
	}
	for i := 0; i < 3; i++ {
		calls = append(calls, fail(150, "Error: query too long"))
	}
	return toolfriction.Window{Calls: calls}, nil
}

// The friction list applies the recommendations' materiality floor to the
// same score: a finding under it is counted, not listed, and the response says
// how many, as it says how many material findings the limit cut. A window
// that starts at or after now is refused.
func TestUsageFrictionCountsWhatTheFloorAndLimitLeaveOut(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	h.SetUsageAttribution(manyRunFacts{usageFacts{now: now}}, nil, &staticCoverage{}, &memoryStatuses{})
	response, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	msg := response.Msg
	if len(msg.GetToolFindings()) != 1 || msg.GetBelowFloor() != 1 || msg.GetOpen() != 2 || msg.GetMaterialityScore() != toolfriction.MaterialScore {
		t.Fatalf("listed %d, below %d, open %d: %v", len(msg.GetToolFindings()), msg.GetBelowFloor(), msg.GetOpen(), msg)
	}
	if f := msg.GetToolFindings()[0]; f.GetScore() < toolfriction.MaterialScore || !strings.Contains(f.GetDescription(), "index locked") {
		t.Errorf("listed finding = %v", f)
	}
	limited, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{Limit: 1}))
	if err != nil || limited.Msg.GetBeyondLimit() != 0 {
		t.Fatalf("limit 1 = %v, %v", limited, err)
	}
	if _, err := h.UsageFriction(context.Background(), connect.NewRequest(&measurepb.UsageFrictionRequest{Since: now.Add(time.Hour).Format(time.RFC3339)})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("a window starting after now = %v, want invalid argument", err)
	}
}
