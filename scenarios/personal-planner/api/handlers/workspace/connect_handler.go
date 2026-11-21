package workspace

import (
	"context"
	"log"

	"connectrpc.com/connect"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace"
	"google.golang.org/protobuf/types/known/timestamppb"

	d "personal-planner/internal/workspace"
)

type Deps struct {
	Service d.Service
	Logger  *log.Logger
}

type connectHandler struct{ deps Deps }

func NewConnectHandler(x Deps) *connectHandler {
	if x.Logger == nil {
		x.Logger = log.Default()
	}
	return &connectHandler{deps: x}
}

func (h *connectHandler) GetProfile(ctx context.Context, _ *connect.Request[v.GetProfileRequest]) (*connect.Response[v.GetProfileResponse], error) {
	p, err := h.deps.Service.Get(ctx)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.GetProfileResponse{Profile: toProto(p)}), nil
}

func (h *connectHandler) UpdateProfile(ctx context.Context, req *connect.Request[v.UpdateProfileRequest]) (*connect.Response[v.UpdateProfileResponse], error) {
	p, err := h.deps.Service.Update(ctx, d.UpdateInput{
		Timezone: req.Msg.Timezone, WeekStart: req.Msg.WeekStart,
		DailyCapacityMinutes: int(req.Msg.DailyCapacityMinutes), ReserveMinutes: int(req.Msg.ReserveMinutes),
		FocusSessionMinutes: int(req.Msg.FocusSessionMinutes), ExpectedRevision: req.Msg.ExpectedRevision,
	})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.UpdateProfileResponse{Profile: toProto(p)}), nil
}

func (h *connectHandler) ListAvailability(ctx context.Context, _ *connect.Request[v.ListAvailabilityRequest]) (*connect.Response[v.ListAvailabilityResponse], error) {
	a, err := h.deps.Service.ListAvailability(ctx)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.ListAvailabilityResponse{Windows: windowsToProto(a.Windows), Exceptions: exceptionsToProto(a.Exceptions), Revision: a.Revision}), nil
}

func (h *connectHandler) ReplaceAvailability(ctx context.Context, req *connect.Request[v.ReplaceAvailabilityRequest]) (*connect.Response[v.ReplaceAvailabilityResponse], error) {
	in := d.AvailabilityInput{ExpectedRevision: req.Msg.ExpectedRevision}
	for _, w := range req.Msg.Windows {
		in.Windows = append(in.Windows, d.AvailabilityWindow{ID: w.Id, Weekday: int(w.Weekday), StartMinute: int(w.StartMinute), EndMinute: int(w.EndMinute), Timezone: w.Timezone, EffectiveStartDate: w.EffectiveStartDate, EffectiveEndDate: w.EffectiveEndDate, Priority: int(w.Priority), Revision: w.Revision})
	}
	for _, e := range req.Msg.Exceptions {
		in.Exceptions = append(in.Exceptions, d.AvailabilityException{ID: e.Id, Date: e.Date, StartMinute: int(e.StartMinute), EndMinute: int(e.EndMinute), Kind: e.Kind, Reason: e.Reason, Revision: e.Revision})
	}
	a, err := h.deps.Service.ReplaceAvailability(ctx, in)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.ReplaceAvailabilityResponse{Windows: windowsToProto(a.Windows), Exceptions: exceptionsToProto(a.Exceptions), Revision: a.Revision}), nil
}

func windowsToProto(items []d.AvailabilityWindow) []*v.AvailabilityWindow {
	out := make([]*v.AvailabilityWindow, 0, len(items))
	for _, w := range items {
		out = append(out, &v.AvailabilityWindow{Id: w.ID, Weekday: int32(w.Weekday), StartMinute: int32(w.StartMinute), EndMinute: int32(w.EndMinute), Timezone: w.Timezone, EffectiveStartDate: w.EffectiveStartDate, EffectiveEndDate: w.EffectiveEndDate, Priority: int32(w.Priority), Revision: w.Revision})
	}
	return out
}

func exceptionsToProto(items []d.AvailabilityException) []*v.AvailabilityException {
	out := make([]*v.AvailabilityException, 0, len(items))
	for _, e := range items {
		out = append(out, &v.AvailabilityException{Id: e.ID, Date: e.Date, StartMinute: int32(e.StartMinute), EndMinute: int32(e.EndMinute), Kind: e.Kind, Reason: e.Reason, Revision: e.Revision})
	}
	return out
}

func toProto(p d.Profile) *v.PlanningProfile {
	return &v.PlanningProfile{Id: p.ID, Timezone: p.Timezone, WeekStart: p.WeekStart, DailyCapacityMinutes: int32(p.DailyCapacityMinutes), ReserveMinutes: int32(p.ReserveMinutes), FocusSessionMinutes: int32(p.FocusSessionMinutes), Revision: p.Revision, UpdatedAt: timestamppb.New(p.UpdatedAt)}
}
