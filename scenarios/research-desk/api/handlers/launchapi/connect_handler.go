// Package launch serves packets, launches and controls (SYSTEM.md §5–§6).
// Rendering and gating belong to packet.py; terminal input and its checks
// belong to internal/launch.
package launchapi

import (
	"context"
	"errors"
	"log"
	"path/filepath"

	"connectrpc.com/connect"

	"research-desk/internal/home"
	"research-desk/internal/launch"
	"research-desk/internal/skillscripts"
	"research-desk/internal/wire"

	launchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch"
)

// Deps wires the launch handler.
type Deps struct {
	// ResearchDir locates each commission's LAUNCH.json.
	ResearchDir string
	Service     *launch.Service
	// Automatic says how a launch reaches Web Console (the access, in
	// words), or why it cannot.
	Automatic func(ctx context.Context) (string, error)
	Logger    *log.Logger
}

type handler struct{ deps Deps }

// NewHandler returns the Connect handler.
func NewHandler(deps Deps) *handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &handler{deps: deps}
}

// fail maps an error to its wire code by cause (wire.Error).
func (h *handler) fail(op string, err error) error {
	return wire.Error(h.deps.Logger, "launch: "+op, err)
}

func (h *handler) GetLaunchOptions(ctx context.Context, req *connect.Request[launchv1.GetLaunchOptionsRequest]) (*connect.Response[launchv1.GetLaunchOptionsResponse], error) {
	limits, err := h.deps.Service.Limits()
	if err != nil {
		return nil, h.fail("options", err)
	}
	resp := &launchv1.GetLaunchOptionsResponse{
		LimitsPath:              limits.Path,
		DefaultSupervisorModel:  limits.Defaults.SupervisorModel,
		DefaultSupervisorEffort: limits.Defaults.SupervisorEffort,
		DefaultOwnerModelTier:   limits.Defaults.OwnerModelTier,
		SupervisorModels:        limits.Models.SupervisorModel,
		SupervisorEfforts:       limits.Models.SupervisorEffort,
		OwnerModelTiers:         limits.Models.OwnerModelTier,
		CharterBytesLimit:       int32(limits.Gates.CharterBytes),
		StanceCharsLimit:        int32(limits.Gates.StanceChars),
		KernelCharsLimit:        int32(limits.Gates.KernelChars),
		AutomaticAvailable:      true,
		PacketLimits: &launchv1.PacketLimits{
			TopicsMax:             int32(limits.Packet.TopicsMax),
			AllowanceHoursMin:     int32(limits.Packet.AllowanceHoursMin),
			AllowanceHoursMax:     int32(limits.Packet.AllowanceHoursMax),
			AllowanceHoursDefault: int32(limits.Packet.AllowanceHoursDefault),
			OrderedWorkstreamsMax: int32(limits.Packet.OrderedWorkstreamsMax),
		},
	}
	access, err := h.deps.Automatic(ctx)
	resp.WebConsoleAccess, resp.WebConsoleAccessKind = access, wire.AccessKind(access)
	if err != nil {
		resp.AutomaticAvailable, resp.AutomaticReason = false, err.Error()+"; launch by hand with the two messages"
	}
	sug, err := h.deps.Service.Suggest(ctx, req.Msg.GetTopics())
	if err != nil {
		return nil, h.fail("options", err)
	}
	resp.StanceQuotesReason = sug.QuotesReason
	for _, q := range sug.Quotes {
		resp.StanceQuotes = append(resp.StanceQuotes, &launchv1.SuggestedQuote{Text: q.Text, Source: q.Source, CommissionId: q.CommissionID})
	}
	for _, t := range sug.Topics {
		resp.TopicWorkstreams = append(resp.TopicWorkstreams, &launchv1.TopicWorkstreams{Topic: t.Topic, Workstreams: wire.Workstreams(t.Workstreams), Evidence: wire.Evidence(t.Evidence)})
	}
	return connect.NewResponse(resp), nil
}

var resolutionWire = map[launch.Resolution]launchv1.Resolution{
	launch.Supersede:    launchv1.Resolution_RESOLUTION_SUPERSEDE,
	launch.RunAlongside: launchv1.Resolution_RESOLUTION_RUN_ALONGSIDE,
}

// choices reads a request's per-blocker resolutions and its assertion that the packet never started.
func choices(list []*launchv1.BlockerResolution, assert bool) (launch.Choices, error) {
	out := launch.Choices{AssertNeverStarted: assert, Resolutions: map[string]launch.Resolution{}}
	for _, r := range list {
		var choice launch.Resolution
		for k, v := range resolutionWire {
			if v == r.GetChoice() {
				choice = k
			}
		}
		if choice == "" || r.GetCommissionId() == "" {
			return out, home.Cause(home.ErrInvalid, errors.New("each resolution names a commission_id and a choice of RESOLUTION_SUPERSEDE or RESOLUTION_RUN_ALONGSIDE"))
		}
		out.Resolutions[r.GetCommissionId()] = choice
	}
	return out, nil
}

func (h *handler) Preflight(ctx context.Context, req *connect.Request[launchv1.PreflightRequest]) (*connect.Response[launchv1.PreflightResponse], error) {
	ch, err := choices(req.Msg.GetResolutions(), req.Msg.GetAssertNeverStarted())
	if err != nil {
		return nil, h.fail("preflight", err)
	}
	var p launch.Preflight
	switch t := req.Msg.GetTarget().(type) {
	case *launchv1.PreflightRequest_SpecJson:
		p, err = h.deps.Service.PreflightSpec(ctx, []byte(t.SpecJson), ch)
	case *launchv1.PreflightRequest_CommissionId:
		p, err = h.deps.Service.PreflightCommission(ctx, t.CommissionId, ch)
	default:
		return nil, h.fail("preflight", home.Cause(home.ErrInvalid, errors.New("give spec_json or commission_id")))
	}
	if err != nil {
		return nil, h.fail("preflight", err)
	}
	resp := &launchv1.PreflightResponse{Packet: packet(p.Packet), Topics: verdicts(p.Verdicts), Launchable: p.Launchable(), Blockers: p.Blockers, OwnerReading: wire.ReadSets(p.OwnerReading)}
	if e := p.PacketState; e != nil {
		resp.PacketState = &launchv1.StateExcerpt{Path: e.Path, Text: e.Text, Truncated: e.Truncated, Reason: e.Reason}
	}
	if p.NeverStarted != (home.NeverStarted{}) {
		resp.NeverStarted = wire.NeverStarted(p.NeverStarted)
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) Build(ctx context.Context, req *connect.Request[launchv1.BuildRequest]) (*connect.Response[launchv1.BuildResponse], error) {
	ch, err := choices(req.Msg.GetResolutions(), false)
	if err != nil {
		return nil, h.fail("build", err)
	}
	var p launch.Preflight
	switch s := req.Msg.GetSource().(type) {
	case *launchv1.BuildRequest_SpecJson:
		p, err = h.deps.Service.Build(ctx, []byte(s.SpecJson), ch.Resolutions)
	case *launchv1.BuildRequest_RelaunchOf:
		p, err = h.deps.Service.Relaunch(ctx, s.RelaunchOf, ch.Resolutions)
	default:
		return nil, h.fail("build", home.Cause(home.ErrInvalid, errors.New("give spec_json or relaunch_of")))
	}
	if err != nil {
		return nil, h.fail("build", err)
	}
	return connect.NewResponse(&launchv1.BuildResponse{Packet: packet(p.Packet), Topics: verdicts(p.Verdicts)}), nil
}

func (h *handler) Launch(ctx context.Context, req *connect.Request[launchv1.LaunchRequest]) (*connect.Response[launchv1.LaunchResponse], error) {
	id := req.Msg.GetCommissionId()
	if err := wire.ResearchOperation(ctx, "research.launch", id); err != nil {
		return nil, h.fail("authorize launch", err)
	}
	ch, err := choices(req.Msg.GetResolutions(), req.Msg.GetAssertNeverStarted())
	if err != nil {
		return nil, h.fail("launch", err)
	}
	r, err := h.deps.Service.Launch(ctx, id, ch)
	if err != nil {
		return nil, h.fail("launch", err)
	}
	return connect.NewResponse(&launchv1.LaunchResponse{Launch: h.record(id, r, true)}), nil
}

func (h *handler) GetLaunchMessages(ctx context.Context, req *connect.Request[launchv1.GetLaunchMessagesRequest]) (*connect.Response[launchv1.GetLaunchMessagesResponse], error) {
	m, err := h.deps.Service.Messages(ctx, req.Msg.GetCommissionId())
	if err != nil {
		return nil, h.fail("launch messages", err)
	}
	return connect.NewResponse(&launchv1.GetLaunchMessagesResponse{CharterPath: m.CharterPath, Stance: m.Stance, Kernel: m.Kernel}), nil
}

func (h *handler) GetLaunch(ctx context.Context, req *connect.Request[launchv1.GetLaunchRequest]) (*connect.Response[launchv1.GetLaunchResponse], error) {
	r, running, err := h.deps.Service.Record(ctx, req.Msg.GetCommissionId())
	if err != nil {
		return nil, h.fail("get launch", err)
	}
	return connect.NewResponse(&launchv1.GetLaunchResponse{Launch: h.record(req.Msg.GetCommissionId(), r, running)}), nil
}

var actions = map[launchv1.ControlAction]string{
	launchv1.ControlAction_CONTROL_ACTION_PAUSE:         launch.ActPause,
	launchv1.ControlAction_CONTROL_ACTION_RESUME:        launch.ActResume,
	launchv1.ControlAction_CONTROL_ACTION_STOP:          launch.ActStop,
	launchv1.ControlAction_CONTROL_ACTION_DISCARD:       launch.ActDiscard,
	launchv1.ControlAction_CONTROL_ACTION_RESEND_STANCE: launch.ActResendStance,
}

func (h *handler) Control(ctx context.Context, req *connect.Request[launchv1.ControlRequest]) (*connect.Response[launchv1.ControlResponse], error) {
	if err := wire.ResearchOperation(ctx, "research.control", req.Msg.GetCommissionId()); err != nil {
		return nil, h.fail("authorize control", err)
	}
	action, ok := actions[req.Msg.GetAction()]
	if !ok {
		return nil, h.fail("control", home.Cause(home.ErrInvalid, errors.New("action must be pause, resume, stop, discard or resend-stance")))
	}
	out, err := h.deps.Service.Control(ctx, req.Msg.GetCommissionId(), action, req.Msg.GetReason())
	if err != nil {
		return nil, h.fail("control", err)
	}
	return connect.NewResponse(&launchv1.ControlResponse{Launch: h.record(req.Msg.GetCommissionId(), out.Record, false), GoalStatus: out.GoalStatus, Message: out.Message}), nil
}

func (h *handler) TestLaunch(ctx context.Context, _ *connect.Request[launchv1.TestLaunchRequest]) (*connect.Response[launchv1.LaunchTestReport], error) {
	r, err := h.deps.Service.TestLaunch(ctx)
	if err != nil {
		return nil, h.fail("test launch", err)
	}
	return connect.NewResponse(testReport(r)), nil
}

func (h *handler) GetLaunchTest(context.Context, *connect.Request[launchv1.GetLaunchTestRequest]) (*connect.Response[launchv1.LaunchTestReport], error) {
	r, err := h.deps.Service.LatestTest()
	if err != nil {
		return nil, h.fail("get launch test", err)
	}
	return connect.NewResponse(testReport(r)), nil
}

func testReport(r launch.TestReport) *launchv1.LaunchTestReport {
	return &launchv1.LaunchTestReport{
		SessionId: r.SessionID, CodexHome: r.CodexHome, Steps: steps(r.Steps), KernelChars: int32(r.KernelChars),
		StatusAfterGoal: r.StatusAfterGoal, StatusAfterPause: r.StatusAfterPause, StatusAfterResume: r.StatusAfterResume, StatusFinal: r.StatusFinal,
		Stopped: r.Stopped, FailedStep: r.FailedStep, FailedReason: r.FailedReason, ReportPath: r.ReportPath, Running: r.Running, At: wire.Timestamp(r.At),
	}
}

func packet(r *skillscripts.PacketResult) *launchv1.PacketResult {
	if r == nil {
		return nil
	}
	out := &launchv1.PacketResult{Ok: r.OK, CommissionId: r.CommissionID, Folder: r.Folder, Written: r.Written, ExitCode: int32(r.ExitCode)}
	for _, g := range r.Gates {
		out.Gates = append(out.Gates, &launchv1.Gate{Name: g.Name, Value: string(g.Value), Limit: string(g.Limit), Ok: g.OK})
	}
	for _, c := range r.Cards {
		out.Cards = append(out.Cards, &launchv1.Card{Topic: c.Topic, Mode: deref(c.Mode), Order: c.Order, OrderSource: deref(c.OrderSource), ReadFirst: deref(c.ReadFirst)})
	}
	if r.Texts != nil {
		out.Texts = &launchv1.Texts{Charter: r.Texts.Charter, Stance: r.Texts.Stance, Kernel: r.Texts.Kernel}
	}
	for _, e := range r.Errors {
		out.Errors = append(out.Errors, &launchv1.PacketError{Code: e.Code, Message: e.Message})
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func verdicts(vs []launch.Verdict) []*launchv1.TopicVerdict {
	out := make([]*launchv1.TopicVerdict, 0, len(vs))
	for _, v := range vs {
		tv := &launchv1.TopicVerdict{
			Topic: v.Topic, CommissionId: v.CommissionID, State: string(v.State), Blocks: v.Blocks, Message: v.Message, ReadFirst: v.ReadFirst,
			Quiet: v.Quiet, Recommended: resolutionWire[v.Recommended], RecommendedReason: v.RecommendedReason, Chosen: resolutionWire[v.Chosen],
		}
		for _, c := range v.Choices {
			tv.Choices = append(tv.Choices, resolutionWire[c])
		}
		out = append(out, tv)
	}
	return out
}

func (h *handler) record(id string, r *home.LaunchRecord, running bool) *launchv1.LaunchRecord {
	out := &launchv1.LaunchRecord{CommissionId: id, Running: running}
	if r == nil {
		return out
	}
	out.Path = filepath.Join(h.deps.ResearchDir, "cohorts", id, home.LaunchFile)
	out.LaunchedBy, out.WebConsoleSessionId, out.ThreadId, out.Steps = r.LaunchedBy, r.WebConsoleSessionID, r.ThreadID, steps(r.Steps)
	if r.Failed != nil {
		out.FailedStep, out.FailedReason = r.Failed.Step, r.Failed.Reason
	}
	if r.Discarded != nil {
		out.DiscardedAt, out.DiscardedReason = wire.Timestamp(r.Discarded.At), r.Discarded.Reason
	}
	for _, c := range r.Controls {
		out.Controls = append(out.Controls, &launchv1.ControlRecord{Action: c.Action, At: wire.Timestamp(c.At), Outcome: c.Outcome})
	}
	out.Supersedes, out.RunAlongside, out.SupersededBy = r.Supersedes, r.RunAlongside, r.SupersededBy
	if r.NeverStarted != nil {
		out.NeverStarted = &launchv1.NeverStartedAssertion{At: wire.Timestamp(r.NeverStarted.At), Evidence: r.NeverStarted.Evidence}
	}
	return out
}

func steps(in []home.LaunchStep) []*launchv1.LaunchStep {
	out := make([]*launchv1.LaunchStep, 0, len(in))
	for _, s := range in {
		out = append(out, &launchv1.LaunchStep{Step: s.Step, At: wire.Timestamp(s.At)})
	}
	return out
}
