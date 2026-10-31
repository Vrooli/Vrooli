package measures

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/interventions"
	"agent-manager/internal/outcomeledger"
	"agent-manager/internal/usageattribution"
	"agent-manager/internal/usagemodes"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/usagemode"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// armTurns gives every synthetic managed run one request whose size is drawn
// from a lognormal around the arm's mean: the challenger arm is cheaper by a
// known factor, so the whole pipeline (ledger rows, attribution, analysis,
// verdict) must recover it.
type armTurns struct {
	usageFacts
	tokens map[string]int
	models map[string]string
}

func (a armTurns) UsageTurns(_ context.Context, from, to time.Time) ([]usageattribution.Turn, error) {
	var turns []usageattribution.Turn
	for id, tokens := range a.tokens {
		at := a.now.Add(-6 * time.Hour)
		if at.Before(from) || at.After(to) {
			continue
		}
		turns = append(turns, usageattribution.Turn{At: at, RunID: id, Runner: "codex", Model: a.models[id], Reports: 1, Tokens: domain.UsageCounters{InputTokens: tokens, OutputTokens: tokens / 4}})
	}
	return turns, nil
}

func (a armTurns) UsageRuns(_ context.Context, ids []string) (map[string]usageattribution.Run, error) {
	out := map[string]usageattribution.Run{}
	for _, id := range ids {
		out[id] = usageattribution.Run{ID: id, Runner: "codex", Source: usageattribution.SourceAgentManager, Status: "complete", Completed: true, Accepted: true, WorkUnit: id}
	}
	return out, nil
}

type experimentFixture struct {
	h       *Handler
	now     time.Time
	service *usagemodes.Service
	mode    usagemode.Mode
}

// newExperiment records a Learn comparison for code.default and n managed
// units per arm. cheaper is the challenger's mean cost over the incumbent's.
func newExperiment(t *testing.T, perArm int, cheaper float64, acceptedShare float64, seed int64) experimentFixture {
	t.Helper()
	t.Setenv("VROOLI_HOME", t.TempDir())
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(seed))
	h := NewHandler(&fakeStore{}, func() time.Time { return now })
	h.SetUsageForecast(&quotaReadings{spans: codexReadings(now)})
	service := &usagemodes.Service{Tiers: modeTiers{}, Now: func() time.Time { return now.Add(-20 * 24 * time.Hour) }, Path: usagemode.Path()}
	mode, err := service.StartLearn(context.Background(), usagemodes.LearnRequest{Runner: "codex", Role: "code.default", Until: "90d", StartedBy: "operator@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	service.Now = func() time.Time { return now }
	h.SetUsageModes(service)
	turns := armTurns{usageFacts: usageFacts{now: now}, tokens: map[string]int{}, models: map[string]string{}}
	var runs []outcomeledger.RunFacts
	for _, arm := range []string{usagemode.ArmIncumbent, usagemode.ArmChallenger} {
		mean, model := 60_000.0, "gpt-6.1-terra"
		if arm == usagemode.ArmChallenger {
			mean = 60_000 * cheaper
			if cheaper < 1 { // a null comparison runs both arms on one model, so one price
				model = "gpt-6-luna"
			}
		}
		for i := 0; i < perArm; i++ {
			id := fmt.Sprintf("%s-%d", arm, i)
			sigma2 := math.Log(1 + 0.8*0.8)
			turns.tokens[id] = int(mean * math.Exp(-sigma2/2+math.Sqrt(sigma2)*rng.NormFloat64()))
			turns.models[id] = model
			outcome := outcomeledger.OutcomeFailed
			if rng.Float64() < acceptedShare {
				outcome = outcomeledger.OutcomeAccepted
			}
			runs = append(runs, outcomeledger.RunFacts{
				RunID: id, WorkUnitID: id, Path: outcomeledger.PathHeadless, Harness: "codex", Role: "code.default", Requests: 1, ActualModel: model,
				CanaryArm: arm, Arm: arm, ArmMode: mode.ID, ArmBasis: outcomeledger.ArmBasisCanary, SelectionReason: "usage_mode_learn:" + mode.ID,
				UnitOutcome: outcome, UnitSource: outcomeledger.SourceValidation, FirstActivity: now.Add(-6 * time.Hour), LastActivity: now.Add(-6 * time.Hour),
			})
		}
	}
	h.SetUsageAttribution(turns, nil, &staticCoverage{}, &memoryStatuses{})
	h.SetOutcomeLedger(ledgerFixture{runs: runs, periods: []outcomeledger.ModePeriod{
		{ID: mode.ID, Kind: usagemode.KindLearn, Runner: "codex", Role: "code.default", Split: 0.5, StartedAt: mode.StartedAt, ExpiresAt: *mode.ExpiresAt, Targets: []string{"gpt-6-luna"}},
	}})
	return experimentFixture{h: h, now: now, service: service, mode: mode}
}

func (f experimentFixture) read(t *testing.T) *measurepb.UsageExperiment {
	t.Helper()
	response, err := f.h.UsageExperiment(context.Background(), connect.NewRequest(&measurepb.UsageExperimentRequest{}))
	if err != nil || len(response.Msg.GetExperiments()) != 1 {
		t.Fatalf("experiments = %v err=%v", response, err)
	}
	return response.Msg.GetExperiments()[0]
}

func TestEvidenceRecoversAKnownCheaperChallengerEndToEnd(t *testing.T) {
	f := newExperiment(t, 150, 0.4, 0.9, 11)
	experiment := f.read(t)
	if experiment.GetDesign() != "randomized" || experiment.GetVerdict() != outcomeledger.ExperimentCheaperBetter {
		t.Fatalf("verdict %q (%s), design %q", experiment.GetVerdict(), experiment.GetReason(), experiment.GetDesign())
	}
	if ratio := experiment.GetCostRatio(); !ratio.GetAvailable() || ratio.GetValue() >= 0.7 || ratio.GetHigh() >= 1 {
		t.Fatalf("a challenger priced well below the incumbent must read well below 1: %v", ratio)
	}
	if experiment.GetIncumbent().GetAccepted() < 80 || experiment.GetChallenger().GetAccepted() < 80 || experiment.GetIncumbent().GetUnits() != 150 {
		t.Fatalf("arms = %v / %v", experiment.GetIncumbent(), experiment.GetChallenger())
	}
	recommendation := experiment.GetRecommendation()
	if !recommendation.GetAvailable() || recommendation.GetToModel() != "gpt-6-luna" || recommendation.GetFromModel() != "gpt-6.1-terra" || recommendation.GetApplied() || recommendation.GetDecisionId() == "" {
		t.Fatalf("recommendation = %v", recommendation)
	}
	if experiment.GetPlan() == nil || experiment.GetPlan().GetMinAcceptedPerArm() == 0 {
		t.Fatalf("plan = %v", experiment.GetPlan())
	}
}

func TestEvidenceForAnIdenticalChallengerNeverClaimsADifference(t *testing.T) {
	f := newExperiment(t, 150, 1, 0.9, 12)
	experiment := f.read(t)
	switch experiment.GetVerdict() {
	case outcomeledger.ExperimentAboutSame, outcomeledger.ExperimentInconclusive:
	default:
		t.Fatalf("a null comparison read %q: %s", experiment.GetVerdict(), experiment.GetReason())
	}
}

func TestEvidenceWithTooLittleDataSaysWhatIsStillNeeded(t *testing.T) {
	f := newExperiment(t, 12, 0.4, 0.9, 13)
	experiment := f.read(t)
	if experiment.GetVerdict() != outcomeledger.ExperimentNotEnough || experiment.GetNeedAccepted() <= 0 || !strings.Contains(experiment.GetReason(), "reading this earlier cannot produce a verdict") {
		t.Fatalf("low data: %q need %d (%s)", experiment.GetVerdict(), experiment.GetNeedAccepted(), experiment.GetReason())
	}
	if experiment.GetRecommendation().GetAvailable() {
		t.Fatal("no recommendation without a verdict")
	}
	if experiment.GetPlan().GetDetect() == nil {
		t.Fatalf("the view must say how long a verdict will take: %v", experiment.GetPlan())
	}
}

func TestOutcomesByArmReportsARandomizedDesign(t *testing.T) {
	f := newExperiment(t, 20, 0.4, 0.9, 14)
	response, err := f.h.UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{By: "arm"}))
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, row := range response.Msg.GetRows() {
		keys[row.GetKey()] = true
	}
	if response.Msg.GetDesign() != "randomized" || !keys["incumbent"] || !keys["challenger"] {
		t.Fatalf("design %q rows %v", response.Msg.GetDesign(), keys)
	}
	byModel, err := f.h.UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{By: "model"}))
	if err != nil || byModel.Msg.GetDesign() != "observational" {
		t.Fatalf("model comparisons stay observational: %q %v", byModel.Msg.GetDesign(), err)
	}
}

// Randomized is read from the assignment basis on record. Work with an arm whose
// basis is missing never reads as randomized, and a comparison whose sessions were
// assigned by day block says so.
func TestOutcomesByArmGatesTheDesignOnTheAssignmentBasis(t *testing.T) {
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	build := func(basis string) string {
		h := usageHandler(now)
		h.SetOutcomeLedger(ledgerFixture{runs: []outcomeledger.RunFacts{
			{RunID: "a", WorkUnitID: "a", Path: outcomeledger.PathImported, Harness: "codex", Requests: 1, ActualModel: "gpt-6-luna", Arm: usagemode.ArmChallenger, ArmMode: "um-t", ArmBasis: basis, UnitOutcome: outcomeledger.OutcomeUnknown},
		}})
		response, err := h.UsageOutcomes(context.Background(), connect.NewRequest(&measurepb.UsageOutcomesRequest{By: "arm"}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetDesign()
	}
	if got := build(outcomeledger.ArmBasisCanary); got != outcomeledger.DesignRandomized {
		t.Fatalf("recorded canary arm: %q", got)
	}
	if got := build(outcomeledger.ArmBasisCalendar); got != outcomeledger.DesignDayBlockRandomized {
		t.Fatalf("session calendar: %q", got)
	}
	if got := build(""); got != outcomeledger.DesignObservational {
		t.Fatalf("no basis on record: %q", got)
	}
}

// A role whose managed work only ever finishes has no verdict to wait for.
func TestPlanSaysNoVerdictIsReachableFromCompletedOnlyWork(t *testing.T) {
	var runs []outcomeledger.RunFacts
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("r%d", i)
		runs = append(runs, outcomeledger.RunFacts{RunID: id, WorkUnitID: id, Path: outcomeledger.PathHeadless, Harness: "codex", Role: "code.default", Requests: 1, ActualModel: "gpt-6.1-terra", UnitOutcome: outcomeledger.OutcomeCompleted, UnitSource: outcomeledger.SourceHeadlessStatus})
	}
	now := time.Date(2026, 10, 29, 6, 0, 0, 0, time.UTC)
	h := usageHandler(now)
	h.SetOutcomeLedger(ledgerFixture{runs: runs})
	pilot, err := h.rolePilot(context.Background(), "code.default", now)
	if err != nil {
		t.Fatal(err)
	}
	plan := protoPlan(planFor(pilot, 50, 0))
	if plan.GetDetect().GetReached() || plan.GetAcceptedShare() != 0 || plan.GetCompletedShare() != 1 || !plan.GetCompletedOnly().GetReached() || !strings.Contains(strings.Join(plan.GetNotes(), " "), "no verdict is reachable") {
		t.Fatalf("plan = %v", plan)
	}
}

func TestEvidenceReadyNamesOnlyComparisonsWithAVerdict(t *testing.T) {
	decided := newExperiment(t, 150, 0.4, 0.9, 15)
	reports, err := decided.h.ReadyEvidence(context.Background(), decided.now)
	if err != nil || len(reports) != 1 || reports[0].ModeID != decided.mode.ID || !strings.Contains(reports[0].Line, "cheaper model looks better") {
		t.Fatalf("ready = %+v err=%v", reports, err)
	}
	early := newExperiment(t, 12, 0.4, 0.9, 16)
	if reports, err := early.h.ReadyEvidence(context.Background(), early.now); err != nil || len(reports) != 0 {
		t.Fatalf("a comparison with no verdict must not announce a result: %+v %v", reports, err)
	}
}

type proposerFake struct {
	proposed []usagemodes.RoleDefaultProposal
	states   map[string]string
}

func (p *proposerFake) ProposeRoleDefault(_ context.Context, proposal usagemodes.RoleDefaultProposal) (string, error) {
	p.proposed = append(p.proposed, proposal)
	return "decision-1", nil
}

func (p *proposerFake) AskState(id string) (string, string, bool) {
	state, ok := p.states[id]
	return state, "", ok
}

func TestRoleDefaultProposalNeedsAVerifiedOperatorAndASupportingVerdict(t *testing.T) {
	decided := newExperiment(t, 150, 0.4, 0.9, 17)
	proposer := &proposerFake{}
	decided.h.SetRoleProposer(proposer)
	request := func() *connect.Request[measurepb.ProposeRoleDefaultRequest] {
		return connect.NewRequest(&measurepb.ProposeRoleDefaultRequest{ModeId: decided.mode.ID})
	}
	if _, err := decided.h.ProposeRoleDefault(withPrincipal(agent), request()); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent proposal code = %v", connect.CodeOf(err))
	}
	if len(proposer.proposed) != 0 {
		t.Fatal("a refused proposal must raise nothing")
	}
	response, err := decided.h.ProposeRoleDefault(withPrincipal(operator), request())
	if err != nil || len(proposer.proposed) != 1 || proposer.proposed[0].To != "gpt-6-luna" || proposer.proposed[0].Role != "code.default" || response.Msg.GetRecommendation().GetApplied() {
		t.Fatalf("proposal = %+v err=%v", proposer.proposed, err)
	}
	early := newExperiment(t, 12, 0.4, 0.9, 18)
	early.h.SetRoleProposer(&proposerFake{})
	if _, err := early.h.ProposeRoleDefault(withPrincipal(operator), connect.NewRequest(&measurepb.ProposeRoleDefaultRequest{ModeId: early.mode.ID})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("a comparison with no verdict must not be proposed: %v", err)
	}
}

func TestExperimentWritesRefuseAgentsAndTheStartIsRecorded(t *testing.T) {
	f := newExperiment(t, 5, 0.4, 0.9, 19)
	f.h.SetUsageModes(&usagemodes.Service{Tiers: modeTiers{}, Now: func() time.Time { return f.now }, Path: filepath.Join(t.TempDir(), "other.json")})
	start := func() *connect.Request[measurepb.StartUsageExperimentRequest] {
		return connect.NewRequest(&measurepb.StartUsageExperimentRequest{Runner: "codex", Role: "code.default", Until: "90d", Sessions: true})
	}
	for name, ctx := range map[string]context.Context{"agent": withPrincipal(agent), "no identity": context.Background()} {
		if _, err := f.h.StartUsageExperiment(ctx, start()); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: start code = %v", name, connect.CodeOf(err))
		}
	}
	started, err := f.h.StartUsageExperiment(withPrincipal(operator), start())
	if err != nil {
		t.Fatal(err)
	}
	mode := started.Msg.GetMode()
	if mode.GetKind() != "learn" || mode.GetStartedBy() != "operator@example.com" || !mode.GetLearn().GetSessionsOptIn() || mode.GetLearn().GetSplitPercent() != 50 || len(mode.GetSettings()) != 0 {
		t.Fatalf("started = %v", mode)
	}
	if started.Msg.GetPlan() == nil || started.Msg.GetPlan().GetDetect() == nil {
		t.Fatalf("a start must restate the expected time to a verdict: %v", started.Msg.GetPlan())
	}
	exempt := connect.NewRequest(&measurepb.StartUsageExperimentRequest{Runner: "codex", Role: "code.review", Until: "90d"})
	if _, err := f.h.StartUsageExperiment(withPrincipal(operator), exempt); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("exempt role start code = %v", connect.CodeOf(err))
	}
}

func TestPlanRefusesExemptRolesAndStatesATimeForOthers(t *testing.T) {
	f := newExperiment(t, 30, 0.4, 0.9, 20)
	refused, err := f.h.PlanUsageExperiment(context.Background(), connect.NewRequest(&measurepb.PlanUsageExperimentRequest{Runner: "codex", Role: "code.review"}))
	if err != nil || refused.Msg.GetRefusal() == "" || refused.Msg.GetPlan() != nil {
		t.Fatalf("exempt plan = %v err=%v", refused, err)
	}
	if _, err := f.h.PlanUsageExperiment(context.Background(), connect.NewRequest(&measurepb.PlanUsageExperimentRequest{Runner: "grok", Role: "x"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown runner: %v", err)
	}
	plan, err := f.h.PlanUsageExperiment(context.Background(), connect.NewRequest(&measurepb.PlanUsageExperimentRequest{Runner: "codex", Role: "code.default"}))
	if err != nil || plan.Msg.GetPlan().GetPilotUnits() == 0 || plan.Msg.GetPlan().GetMinAcceptedPerArm() != int64(outcomeledger.DefaultExperimentConfig().MinAcceptedPerArm) {
		t.Fatalf("plan = %v err=%v", plan, err)
	}
}

type signalsFake struct{ enabled bool }

func (s *signalsFake) Status(context.Context) (interventions.SignalStatus, error) {
	return interventions.SignalStatus{Enabled: s.enabled, Source: interventions.SignalSourceOperator, Stored: 4}, nil
}
func (s *signalsFake) Set(enabled bool, _ string) error { s.enabled = enabled; return nil }
func (s *signalsFake) Purge(context.Context) (int64, error) {
	return 4, nil
}

func TestSignalSwitchChangesAreOperatorOnlyAndStatusIsReadable(t *testing.T) {
	h := NewHandler(nil, nil)
	control := &signalsFake{enabled: true}
	h.SetSignalControl(control)
	for _, action := range []string{"off", "on", "purge"} {
		if _, err := h.SetUsageSignals(withPrincipal(agent), connect.NewRequest(&measurepb.SetUsageSignalsRequest{Action: action})); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("agent %s code = %v", action, connect.CodeOf(err))
		}
	}
	if !control.enabled {
		t.Fatal("a refused agent changed the switch")
	}
	status, err := h.UsageSignalsStatus(withPrincipal(agent), connect.NewRequest(&measurepb.UsageSignalsStatusRequest{}))
	if err != nil || !status.Msg.GetEnabled() || status.Msg.GetStored() != 4 {
		t.Fatalf("status = %v err=%v", status, err)
	}
	off, err := h.SetUsageSignals(withPrincipal(operator), connect.NewRequest(&measurepb.SetUsageSignalsRequest{Action: "off"}))
	if err != nil || off.Msg.GetStatus().GetEnabled() {
		t.Fatalf("off = %v err=%v", off, err)
	}
	purged, err := h.SetUsageSignals(withPrincipal(operator), connect.NewRequest(&measurepb.SetUsageSignalsRequest{Action: "purge"}))
	if err != nil || purged.Msg.GetPurged() != 4 {
		t.Fatalf("purge = %v err=%v", purged, err)
	}
	if _, err := h.SetUsageSignals(withPrincipal(operator), connect.NewRequest(&measurepb.SetUsageSignalsRequest{Action: "pause"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown action: %v", err)
	}
	if _, err := NewHandler(nil, nil).UsageSignalsStatus(context.Background(), connect.NewRequest(&measurepb.UsageSignalsStatusRequest{})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("no control: %v", err)
	}
}

// Window charts shade a mode after it ended, so the list carries every
// recorded period, labelled, not just the active ones.
func TestListUsageModesCarriesEndedPeriodsForTheWindowChart(t *testing.T) {
	h := modeHandler(t)
	now := h.now().UTC()
	h.SetOutcomeLedger(ledgerFixture{periods: []outcomeledger.ModePeriod{
		{ID: "um-ended", Label: "Stretch the week", Runner: "codex", StartedAt: now.Add(-48 * time.Hour), EndedAt: now.Add(-24 * time.Hour)},
		{ID: "um-ancient", Runner: "codex", StartedAt: now.Add(-90 * 24 * time.Hour), EndedAt: now.Add(-89 * 24 * time.Hour)},
	}})
	started, err := h.StartUsageMode(withPrincipal(operator), startRequest())
	if err != nil {
		t.Fatal(err)
	}
	listed, err := h.ListUsageModes(context.Background(), connect.NewRequest(&measurepb.ListUsageModesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetModes()) != 1 || listed.Msg.GetModes()[0].GetMode().GetId() != started.Msg.GetMode().GetId() {
		t.Fatalf("active modes = %v", listed.Msg.GetModes())
	}
	periods := listed.Msg.GetPeriods()
	if len(periods) != 1 || periods[0].GetId() != "um-ended" || periods[0].GetLabel() != "Stretch the week" || periods[0].GetEndedAt() == "" {
		t.Fatalf("periods = %v, want the ended one, labelled, and nothing older than 35 days", periods)
	}
}
