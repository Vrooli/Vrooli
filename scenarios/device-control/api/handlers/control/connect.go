package control

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
	internal "device-control/internal/control"
	"device-control/strategy"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices/devices_v1connect"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/evidence"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/evidence/evidence_v1connect"
	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows"
	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows/flows_v1connect"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/sessions/sessions_v1connect"
	strategiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/strategies"
	strategiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/strategies/strategies_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func registerConnectServices(r *mux.Router, h *handler) {
	registerAuthConnectService(r, h)
	path, svc := devicesconnect.NewDeviceServiceHandler(&deviceConnect{h})
	connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
	path, svc = strategiesconnect.NewStrategyServiceHandler(&strategyConnect{h})
	connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
	path, svc = sessionsconnect.NewSessionServiceHandler(&sessionConnect{h})
	connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
	path, svc = flowsconnect.NewFlowServiceHandler(&flowConnect{h})
	connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
	path, svc = evidenceconnect.NewEvidenceServiceHandler(&evidenceConnect{h})
	connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
}

type deviceConnect struct{ h *handler }

func (c *deviceConnect) ListDevices(ctx context.Context, _ *connect.Request[devicesv1.ListDevicesRequest]) (*connect.Response[devicesv1.ListDevicesResponse], error) {
	items := c.h.service.Devices(ctx)
	out := make([]*devicesv1.Device, 0, len(items))
	for _, d := range items {
		out = append(out, deviceProto(d))
	}
	return connect.NewResponse(&devicesv1.ListDevicesResponse{Devices: out}), nil
}

func (c *deviceConnect) ConnectDevice(ctx context.Context, req *connect.Request[devicesv1.ConnectDeviceRequest]) (*connect.Response[devicesv1.ConnectDeviceResponse], error) {
	rungs := c.h.service.OnboardingLive(ctx, req.Msg.Kind)
	out := make([]*devicesv1.OnboardingRung, 0, len(rungs))
	first := ""
	for _, rung := range rungs {
		out = append(out, &devicesv1.OnboardingRung{Id: rung["id"], Prerequisite: rung["prerequisite"], Owner: rung["owner"], Status: rung["status"], NextAction: rung["next_action"]})
		if first == "" && rung["status"] != "available" {
			first = rung["next_action"]
		}
	}
	return connect.NewResponse(&devicesv1.ConnectDeviceResponse{Rungs: out, FirstNextAction: first}), nil
}

func (c *deviceConnect) ReconnectDevice(ctx context.Context, req *connect.Request[devicesv1.ReconnectDeviceRequest]) (*connect.Response[devicesv1.ReconnectDeviceResponse], error) {
	device, err := c.h.service.ReconnectWireless(ctx, req.Msg.DeviceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&devicesv1.ReconnectDeviceResponse{Device: deviceProto(device)}), nil
}

func deviceProto(d internal.Device) *devicesv1.Device {
	caps := make([]*devicesv1.CapabilitySnapshot, 0, len(d.Capabilities))
	for _, c := range d.Capabilities {
		caps = append(caps, &devicesv1.CapabilitySnapshot{Name: c.Name, Status: c.Status, Prerequisite: c.Prerequisite, NextAction: c.NextAction})
	}
	return &devicesv1.Device{Id: d.ID, Name: d.Name, Kind: d.Kind, Serial: d.Serial, Model: d.Model, OsVersion: d.OSVersion, StrategyId: d.StrategyID, Status: d.Status, Health: d.Health, HealthReason: d.HealthReason, HostNodeId: d.HostNodeID, Transport: d.Transport, Capabilities: caps, ObservedAt: timestamppb.New(d.ObservedAt), FirstSeenAt: d.FirstSeenAt.Format(time.RFC3339Nano), LastSeenAt: d.LastSeenAt.Format(time.RFC3339Nano)}
}

type strategyConnect struct{ h *handler }

func (c *strategyConnect) ListStrategies(ctx context.Context, _ *connect.Request[strategiesv1.ListStrategiesRequest]) (*connect.Response[strategiesv1.ListStrategiesResponse], error) {
	out := make([]*strategiesv1.Strategy, 0)
	for _, d := range c.h.service.Strategies(ctx) {
		out = append(out, strategyProto(d))
	}
	return connect.NewResponse(&strategiesv1.ListStrategiesResponse{Strategies: out}), nil
}

func (c *strategyConnect) VerifyStrategy(ctx context.Context, req *connect.Request[strategiesv1.VerifyStrategyRequest]) (*connect.Response[strategiesv1.ConformanceReport], error) {
	r, err := c.h.service.Verify(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&strategiesv1.ConformanceReport{StrategyId: r.StrategyID, Status: r.Status, Passed: r.Passed, Failed: r.Failed, Tiers: r.Tiers, ExecutableStepKinds: r.ExecutableStepKinds, NextActions: r.NextActions, Promotable: r.Promotable, EvidenceClass: r.EvidenceClass, MinimumUsefulFps: r.MinimumUsefulFPS}), nil
}

func strategyProto(d strategy.Declaration) *strategiesv1.Strategy {
	out := &strategiesv1.Strategy{Id: d.StrategyID, Description: d.Description, Status: d.Status, Tiers: d.Tiers, ExecutableStepKinds: strategy.StepKinds(d), NextActions: d.NextActions, Promotable: d.Promotable, EvidenceClass: d.EvidenceClass, MinimumUsefulFps: d.MinimumUsefulFPS}
	for _, c := range d.Capabilities {
		out.Capabilities = append(out.Capabilities, &strategiesv1.Capability{Name: c.Name, Status: c.Status, Prerequisite: c.Prerequisite, NextAction: c.NextAction})
	}
	return out
}

type sessionConnect struct{ h *handler }

func (c *sessionConnect) ListSessions(ctx context.Context, _ *connect.Request[sessionsv1.ListSessionsRequest]) (*connect.Response[sessionsv1.ListSessionsResponse], error) {
	out := make([]*sessionsv1.Session, 0)
	for _, s := range c.h.service.ListLiveSessionsContext(ctx) {
		out = append(out, sessionProto(s, false))
	}
	return connect.NewResponse(&sessionsv1.ListSessionsResponse{Sessions: out}), nil
}

func (c *sessionConnect) AcquireSession(ctx context.Context, req *connect.Request[sessionsv1.AcquireSessionRequest]) (*connect.Response[sessionsv1.SessionResponse], error) {
	s, err := c.h.service.AcquireContext(ctx, req.Msg.DeviceId, req.Msg.Actor, time.Duration(req.Msg.TtlSeconds)*time.Second)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	}
	return connect.NewResponse(&sessionsv1.SessionResponse{Session: sessionProto(s, true)}), nil
}

func (c *sessionConnect) KillSession(ctx context.Context, req *connect.Request[sessionsv1.KillSessionRequest]) (*connect.Response[sessionsv1.SessionResponse], error) {
	s, err := c.h.service.KillContext(ctx, req.Msg.Id, "operator requested kill")
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&sessionsv1.SessionResponse{Session: sessionProto(s, false)}), nil
}

func (c *sessionConnect) ReleaseSession(ctx context.Context, req *connect.Request[sessionsv1.ReleaseSessionRequest]) (*connect.Response[sessionsv1.SessionResponse], error) {
	s, err := c.h.service.ReleaseContext(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&sessionsv1.SessionResponse{Session: sessionProto(s, false)}), nil
}

func sessionProto(s internal.Session, includeToken bool) *sessionsv1.Session {
	out := &sessionsv1.Session{Id: s.ID, DeviceId: s.DeviceID, Actor: s.Actor, State: s.State, ExpiresAt: timestamppb.New(s.ExpiresAt), CreatedAt: timestamppb.New(s.CreatedAt), KillReason: s.KillReason}
	if includeToken {
		out.LeaseToken = s.LeaseToken
	}
	return out
}

type flowConnect struct{ h *handler }

func (c *flowConnect) ValidateFlow(ctx context.Context, req *connect.Request[flowsv1.ValidateFlowRequest]) (*connect.Response[flowsv1.CapabilityGapReport], error) {
	f := flowFromProto(req.Msg.Flow)
	g := c.h.service.Validate(ctx, f, req.Msg.StrategyId)
	return connect.NewResponse(&flowsv1.CapabilityGapReport{Runnable: g.Runnable, Gaps: g.Gaps, Warnings: g.Warnings}), nil
}

func (c *flowConnect) RunFlow(ctx context.Context, req *connect.Request[flowsv1.RunFlowRequest]) (*connect.Response[flowsv1.RunResult], error) {
	result, err := c.h.service.RunWithLease(ctx, flowFromProto(req.Msg.Flow), req.Msg.DeviceId, req.Msg.Actor, req.Msg.LeaseToken)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if strings.Contains(err.Error(), "lease") || strings.Contains(err.Error(), "held") {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(runResultProto(result)), nil
}

func runResultProto(result internal.RunResult) *flowsv1.RunResult {
	out := &flowsv1.RunResult{
		RunId:            result.RunID,
		Disposition:      result.Disposition,
		Incomplete:       result.Incomplete,
		DisconnectReason: result.DisconnectReason,
		DisconnectStep:   result.DisconnectStep,
	}
	for _, ch := range result.Chapters {
		out.Chapters = append(out.Chapters, &flowsv1.Chapter{Id: ch.ID, Title: ch.Title, Disposition: ch.Disposition, Message: ch.Message})
	}
	for _, res := range result.Resolutions {
		out.Resolutions = append(out.Resolutions, &flowsv1.Resolution{Target: res.Target, Rung: res.Rung, Confidence: res.Confidence})
	}
	for _, ref := range result.Evidence {
		out.Evidence = append(out.Evidence, &flowsv1.EvidenceReference{Id: ref.ID, Sha256: ref.SHA256, SizeBytes: ref.SizeBytes, CreatedAt: ref.CreatedAt.Format(time.RFC3339Nano), RedactionVerified: ref.RedactionVerified, RecordingMethod: ref.RecordingMethod, EffectiveFps: ref.EffectiveFPS, Producer: ref.Producer, Kind: ref.Kind, AppliedRules: ref.AppliedRules, OptedOut: ref.OptedOut, ClaimClass: string(ref.ClaimClass), MinimumUsefulFps: ref.MinimumUsefulFPS, Disposition: string(ref.Disposition), DispositionReason: ref.DispositionReason})
	}
	return out
}

func flowFromProto(f *flowsv1.Flow) internal.Flow {
	if f == nil {
		return internal.Flow{}
	}
	out := internal.Flow{ID: f.Id, Name: f.Name, Transport: f.Transport, RequireUnlocked: f.RequireUnlocked, AuthProfileID: f.AuthProfileId, AllowUnredactedCapture: f.AllowUnredactedCapture}
	for _, s := range f.Steps {
		args := map[string]any{}
		if s.Arguments != nil {
			args = s.Arguments.AsMap()
		}
		out.Steps = append(out.Steps, internal.Step{ID: s.Id, Kind: s.Kind, RequiredCapabilities: s.RequiredCapabilities, Target: s.Target, TimeoutMS: s.TimeoutMs, Arguments: args})
	}
	return out
}

type evidenceConnect struct{ h *handler }

func (c *evidenceConnect) ListAudit(ctx context.Context, _ *connect.Request[evidencev1.ListAuditRequest]) (*connect.Response[evidencev1.ListAuditResponse], error) {
	out := make([]*evidencev1.AuditRecord, 0)
	for _, a := range c.h.service.AuditContext(ctx) {
		out = append(out, &evidencev1.AuditRecord{Id: a.ID, Actor: a.Actor, DeviceId: a.DeviceID, LeaseId: a.LeaseID, Verb: a.Verb, Outcome: a.Outcome, CreatedAt: timestamppb.New(a.CreatedAt), RedactionVerified: a.RedactionVerified, RedactionOptedOut: a.RedactionOptedOut})
	}
	return connect.NewResponse(&evidencev1.ListAuditResponse{Records: out}), nil
}
