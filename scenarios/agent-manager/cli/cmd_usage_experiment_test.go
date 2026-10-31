package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

type experimentServer struct {
	measureconnect.UnimplementedMeasuresServiceHandler
	planned   int
	started   *measurepb.StartUsageExperimentRequest
	proposed  string
	setAction string
	denyWrite bool
	refusal   string
	operator  []bool
}

var testPlan = &measurepb.UsageExperimentPlan{
	UnitsPerDay: 0.2, CostCv: 1.84, PilotUnits: 6, VarianceAssumed: true, Effect: 0.5, Power: 0.8, MinAcceptedPerArm: 75, MinEffect: 0.3, ReworkMargin: 0.2,
	Detect: &measurepb.UsageExperimentHorizon{Reached: true, Days: 1200, LowDays: 460, HighDays: 3300, PerArm: 120},
	Same:   &measurepb.UsageExperimentHorizon{Reached: false},
}

func (s *experimentServer) PlanUsageExperiment(context.Context, *connect.Request[measurepb.PlanUsageExperimentRequest]) (*connect.Response[measurepb.PlanUsageExperimentResponse], error) {
	s.planned++
	if s.refusal != "" {
		return connect.NewResponse(&measurepb.PlanUsageExperimentResponse{Refusal: s.refusal}), nil
	}
	return connect.NewResponse(&measurepb.PlanUsageExperimentResponse{Plan: testPlan}), nil
}

func (s *experimentServer) StartUsageExperiment(_ context.Context, req *connect.Request[measurepb.StartUsageExperimentRequest]) (*connect.Response[measurepb.StartUsageExperimentResponse], error) {
	if s.denyWrite {
		return nil, connect.NewError(connect.CodePermissionDenied, errString("usage modes can be started or ended only by a verified operator, not an agent"))
	}
	s.started = req.Msg
	return connect.NewResponse(&measurepb.StartUsageExperimentResponse{
		Mode: &measurepb.UsageMode{Id: "um-learn-1", StartedBy: "operator@example.com", Kind: "learn", Scope: &measurepb.UsageModeScope{Runner: "codex", Role: "code.default"},
			Learn: &measurepb.UsageModeLearn{SplitPercent: 50, Challenger: []*measurepb.UsageModeSettings{{Runner: "codex", ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}}}}},
		Plan: testPlan, Notes: []string{"New managed runs of code.default are assigned at random"},
	}), nil
}

func (s *experimentServer) UsageExperiment(context.Context, *connect.Request[measurepb.UsageExperimentRequest]) (*connect.Response[measurepb.UsageExperimentResponse], error) {
	rng := func(v, l, h float64) *measurepb.UsageRange {
		return &measurepb.UsageRange{Available: true, Value: v, Low: l, High: h}
	}
	return connect.NewResponse(&measurepb.UsageExperimentResponse{Experiments: []*measurepb.UsageExperiment{{
		ModeId: "um-learn-1", Role: "code.default", Runner: "codex", IncumbentModel: "gpt-6.1-terra", ChallengerModel: "gpt-6-luna", Design: "randomized", SplitPercent: 50, Active: true,
		Incumbent:  &measurepb.UsageExperimentArm{Arm: "incumbent", Units: 100, Known: 90, Accepted: 80, PointsPerAccepted: rng(10, 8, 12), ReworkRate: rng(0.2, 0.1, 0.3)},
		Challenger: &measurepb.UsageExperimentArm{Arm: "challenger", Units: 100, Known: 90, Accepted: 80, Exceptions: 2, PointsPerAccepted: rng(5, 4, 6), ReworkRate: rng(0.2, 0.1, 0.3)},
		CostRatio:  rng(0.5, 0.4, 0.6), CostPValue: 0.001, ReworkDifference: rng(0, -0.1, 0.1), ReworkPValue: 0.9, Verdict: "cheaper_looks_better",
		Reason: "the challenger cost 50% less per accepted outcome", Plan: testPlan,
		Recommendation: &measurepb.UsageRoleRecommendation{Available: true, Role: "code.default", Runner: "codex", FromModel: "gpt-6.1-terra", ToModel: "gpt-6-luna", PolicyFile: "resources/codex/model-policy.json",
			PolicyOps: []string{`replace /roles/code.default/model: "gpt-6.1-terra" -> "gpt-6-luna"`}, DecisionId: "decision-1"},
	}}}), nil
}

func (s *experimentServer) ProposeRoleDefault(_ context.Context, req *connect.Request[measurepb.ProposeRoleDefaultRequest]) (*connect.Response[measurepb.ProposeRoleDefaultResponse], error) {
	if s.denyWrite {
		return nil, connect.NewError(connect.CodePermissionDenied, errString("usage modes can be started or ended only by a verified operator, not an agent"))
	}
	s.proposed = req.Msg.GetModeId()
	return connect.NewResponse(&measurepb.ProposeRoleDefaultResponse{Recommendation: &measurepb.UsageRoleRecommendation{
		Role: "code.default", ToModel: "gpt-6-luna", DecisionId: "decision-1", PolicyFile: "resources/codex/model-policy.json", PolicyOps: []string{`replace /roles/code.default/model`}}}), nil
}

func (s *experimentServer) EndUsageMode(_ context.Context, req *connect.Request[measurepb.EndUsageModeRequest]) (*connect.Response[measurepb.EndUsageModeResponse], error) {
	return connect.NewResponse(&measurepb.EndUsageModeResponse{EndedIds: req.Msg.GetIds()}), nil
}

func (s *experimentServer) UsageSignalsStatus(context.Context, *connect.Request[measurepb.UsageSignalsStatusRequest]) (*connect.Response[measurepb.UsageSignalsStatusResponse], error) {
	return connect.NewResponse(&measurepb.UsageSignalsStatusResponse{Enabled: true, Source: "default", Stored: 12}), nil
}

func (s *experimentServer) SetUsageSignals(_ context.Context, req *connect.Request[measurepb.SetUsageSignalsRequest]) (*connect.Response[measurepb.SetUsageSignalsResponse], error) {
	if s.denyWrite {
		return nil, connect.NewError(connect.CodePermissionDenied, errString("usage modes can be started or ended only by a verified operator, not an agent"))
	}
	s.setAction = req.Msg.GetAction()
	return connect.NewResponse(&measurepb.SetUsageSignalsResponse{Purged: 12, Status: &measurepb.UsageSignalsStatusResponse{Enabled: req.Msg.GetAction() == "on", Source: "operator", ChangedBy: "operator@example.com"}}), nil
}

func useExperimentServer(t *testing.T, server *experimentServer) {
	t.Helper()
	path, handler := measureconnect.NewMeasuresServiceHandler(server)
	srv := httptest.NewServer(handlerMux(path, handler))
	t.Cleanup(func() { modeClientOverride = nil; srv.Close() })
	modeClientOverride = func(operator bool) measureconnect.MeasuresServiceClient {
		server.operator = append(server.operator, operator)
		return measureconnect.NewMeasuresServiceClient(srv.Client(), srv.URL)
	}
}

func TestExperimentStartShowsTheExpectedTimeAndStartsNothingWithoutConfirm(t *testing.T) {
	server := &experimentServer{}
	useExperimentServer(t, server)
	out := captureStdout(t, func() error {
		return (&App{}).usageExperiment([]string{"start", "--runner", "codex", "--role", "code.default", "--until", "90d"})
	})
	if server.started != nil {
		t.Fatal("a start without --confirm must not start a comparison")
	}
	for _, text := range []string{"nothing started yet", "over 3 years (120 units per arm)", "75 accepted outcomes", "assumed (only 6 finished units", "not reached within the simulated range", "--confirm", "In 90 days no reduction"} {
		if !strings.Contains(out, text) {
			t.Errorf("plan output lacks %q:\n%s", text, out)
		}
	}
}

func TestExperimentStartWithConfirmIsAnOperatorWriteCarryingEveryFlag(t *testing.T) {
	server := &experimentServer{}
	useExperimentServer(t, server)
	out := captureStdout(t, func() error {
		return (&App{}).usageExperiment([]string{"start", "--runner", "codex", "--role", "code.default", "--until", "90d", "--split", "30", "--sessions", "--block-days", "2", "--model", "gpt-6-luna", "--confirm"})
	})
	want := &measurepb.StartUsageExperimentRequest{Runner: "codex", Role: "code.default", Until: "90d", SplitPercent: 30, Sessions: true, BlockDays: 2, Model: "gpt-6-luna"}
	if server.started == nil || server.started.String() != want.String() {
		t.Fatalf("request = %v, want %v", server.started, want)
	}
	if !server.operator[len(server.operator)-1] {
		t.Fatalf("the start itself must use operator authority: %v", server.operator)
	}
	if !strings.Contains(out, "um-learn-1") || !strings.Contains(out, "gpt-6-luna instead of gpt-6.1-terra") {
		t.Errorf("output: %s", out)
	}
}

func TestExperimentStartRefusesAnExemptRoleAndAnAgent(t *testing.T) {
	server := &experimentServer{refusal: "role code.review is exempt from usage modes and is never randomized"}
	useExperimentServer(t, server)
	err := (&App{}).usageExperiment([]string{"start", "--runner", "codex", "--role", "code.review", "--until", "90d", "--confirm"})
	if err == nil || !strings.Contains(err.Error(), "never randomized") || server.started != nil {
		t.Fatalf("err = %v started = %v", err, server.started)
	}
	denied := &experimentServer{denyWrite: true}
	useExperimentServer(t, denied)
	err = (&App{}).usageExperiment([]string{"start", "--runner", "codex", "--role", "code.default", "--until", "90d", "--confirm"})
	if err == nil || !strings.Contains(err.Error(), "verified operator, not an agent") {
		t.Fatalf("agent start err = %v", err)
	}
	if err := (&App{}).usageExperiment([]string{"propose", "um-learn-1"}); err == nil || !strings.Contains(err.Error(), "not an agent") {
		t.Fatalf("agent propose err = %v", err)
	}
}

func TestExperimentStatusShowsDesignVerdictAndTheExactPolicyChangeNotApplied(t *testing.T) {
	server := &experimentServer{}
	useExperimentServer(t, server)
	out := captureStdout(t, func() error { return (&App{}).usageExperiment([]string{"status"}) })
	for _, text := range []string{"design: Randomized", "verdict: cheaper looks better", "p=0.001", "2 exceptions", "nothing is applied", "resources/codex/model-policy.json", "/roles/code.default/model", "apply it yourself", "usage experiment propose um-learn-1", "decision decision-1"} {
		if !strings.Contains(out, text) && !strings.Contains(out, strings.ReplaceAll(text, "decision decision-1", "")) {
			t.Errorf("status output lacks %q:\n%s", text, out)
		}
	}
	if len(server.operator) != 1 || server.operator[0] {
		t.Fatalf("status is a read: %v", server.operator)
	}
}

func TestExperimentProposeAsksForApprovalAndPrintsTheChange(t *testing.T) {
	server := &experimentServer{}
	useExperimentServer(t, server)
	out := captureStdout(t, func() error { return (&App{}).usageExperiment([]string{"propose", "um-learn-1"}) })
	if server.proposed != "um-learn-1" || !strings.Contains(out, "nothing is applied either way") || !strings.Contains(out, "/roles/code.default/model") {
		t.Fatalf("proposed %q out %s", server.proposed, out)
	}
}

func TestSignalsCommandsReadWithoutAuthorityAndChangeAsOperator(t *testing.T) {
	server := &experimentServer{}
	useExperimentServer(t, server)
	out := captureStdout(t, func() error { return (&App{}).usageSignals([]string{"status"}) })
	if !strings.Contains(out, "Operator signals are on") || !strings.Contains(out, "12 labels stored") || server.operator[0] {
		t.Fatalf("status %q operator %v", out, server.operator)
	}
	captureStdout(t, func() error { return (&App{}).usageSignals([]string{"off"}) })
	if server.setAction != "off" || !server.operator[len(server.operator)-1] {
		t.Fatalf("off: action %q operator %v", server.setAction, server.operator)
	}
	if err := (&App{}).usageSignals([]string{"purge"}); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("purge without --yes err = %v", err)
	}
	out = captureStdout(t, func() error { return (&App{}).usageSignals([]string{"purge", "--yes"}) })
	if server.setAction != "purge" || !strings.Contains(out, "Purged 12") {
		t.Fatalf("purge: %q %s", server.setAction, out)
	}
	denied := &experimentServer{denyWrite: true}
	useExperimentServer(t, denied)
	if err := (&App{}).usageSignals([]string{"off"}); err == nil || !strings.Contains(err.Error(), "not an agent") {
		t.Fatalf("agent off err = %v", err)
	}
}

func TestHumanDaysRoundsToWhatAnOperatorPlansWith(t *testing.T) {
	for days, want := range map[float64]string{0: "not reached", 3.2: "4 days", 20: "3 weeks", 100: "4 months", 600: "20 months", 1200: "over 3 years", 1500: "over 4 years"} {
		if got := humanDays(days); got != want {
			t.Errorf("humanDays(%v) = %q, want %q", days, got, want)
		}
	}
}
