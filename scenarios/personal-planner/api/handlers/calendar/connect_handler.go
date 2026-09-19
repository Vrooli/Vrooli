package calendar

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/schedule"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"

	d "personal-planner/internal/calendar"
)

type Deps struct {
	Service d.Service
	Logger  *log.Logger
	Clock   schedule.Clock
}
type connectHandler struct{ deps Deps }

func NewConnectHandler(x Deps) *connectHandler {
	if x.Logger == nil {
		x.Logger = log.Default()
	}
	if x.Clock == nil {
		x.Clock = schedule.System()
	}
	return &connectHandler{deps: x}
}

func (h *connectHandler) ListTodayAllocations(ctx context.Context, req *connect.Request[v.ListTodayAllocationsRequest]) (*connect.Response[v.ListTodayAllocationsResponse], error) {
	date := req.Msg.LocalDate
	if date == "" {
		date = h.deps.Clock.Now().Format("2006-01-02")
	}
	today, err := h.deps.Service.ListToday(ctx, date)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ListTodayAllocationsResponse{PlannedMinutes: int32(today.PlannedMinutes), AvailableMinutes: int32(today.AvailableMinutes), BreathingRoomMinutes: int32(today.BreathingRoomMinutes), ExternalBusyMinutes: int32(today.ExternalBusyMinutes), ExternalEventCount: int32(today.ExternalEventCount), ExternalFreshness: today.ExternalFreshness}
	for _, a := range today.Allocations {
		out.Allocations = append(out.Allocations, toProto(a))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListAllocations(ctx context.Context, req *connect.Request[v.ListAllocationsRequest]) (*connect.Response[v.ListAllocationsResponse], error) {
	rangeResult, err := h.deps.Service.ListRange(ctx, req.Msg.StartLocalDate, req.Msg.EndLocalDate)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ListAllocationsResponse{}
	for _, a := range rangeResult.Allocations {
		out.Allocations = append(out.Allocations, toProto(a))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateAllocation(ctx context.Context, req *connect.Request[v.CreateAllocationRequest]) (*connect.Response[v.CreateAllocationResponse], error) {
	a, err := h.deps.Service.Create(ctx, d.CreateInput{WorkItemID: req.Msg.WorkItemId, LocalDate: req.Msg.LocalDate, StartMinutes: int(req.Msg.StartMinutes), DurationMinutes: int(req.Msg.DurationMinutes)})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.CreateAllocationResponse{Allocation: toProto(a)}), nil
}

func (h *connectHandler) CarryForwardAllocation(ctx context.Context, req *connect.Request[v.CarryForwardAllocationRequest]) (*connect.Response[v.CarryForwardAllocationResponse], error) {
	a, err := h.deps.Service.CarryForward(ctx, d.CarryForwardInput{AllocationID: req.Msg.AllocationId, TargetLocalDate: req.Msg.TargetLocalDate, StartMinutes: int(req.Msg.StartMinutes)})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.CarryForwardAllocationResponse{Allocation: toProto(a)}), nil
}

func (h *connectHandler) ListRoutines(ctx context.Context, _ *connect.Request[v.ListRoutinesRequest]) (*connect.Response[v.ListRoutinesResponse], error) {
	routines, err := h.deps.Service.ListRoutines(ctx)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ListRoutinesResponse{}
	for _, routine := range routines {
		out.Routines = append(out.Routines, routineToProto(routine))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateRoutine(ctx context.Context, req *connect.Request[v.CreateRoutineRequest]) (*connect.Response[v.CreateRoutineResponse], error) {
	routine, err := h.deps.Service.CreateRoutine(ctx, d.CreateRoutineInput{Title: req.Msg.Title, Kind: req.Msg.Kind, Timezone: req.Msg.Timezone, StartDate: req.Msg.StartDate, EndDate: req.Msg.EndDate, Weekdays: ints(req.Msg.Weekdays), StartMinute: int(req.Msg.StartMinute), DurationMinutes: int(req.Msg.DurationMinutes), FrequencyPerWeek: int(req.Msg.FrequencyPerWeek)})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.CreateRoutineResponse{Routine: routineToProto(routine)}), nil
}

func (h *connectHandler) ListRoutineOccurrences(ctx context.Context, req *connect.Request[v.ListRoutineOccurrencesRequest]) (*connect.Response[v.ListRoutineOccurrencesResponse], error) {
	occurrences, err := h.deps.Service.ListRoutineOccurrences(ctx, req.Msg.StartLocalDate, req.Msg.EndLocalDate)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ListRoutineOccurrencesResponse{}
	for _, occurrence := range occurrences {
		out.Occurrences = append(out.Occurrences, &v.RoutineOccurrence{RoutineId: occurrence.RoutineID, Title: occurrence.Title, LocalDate: occurrence.LocalDate, StartMinute: int32(occurrence.StartMinute), DurationMinutes: int32(occurrence.DurationMinutes), Kind: occurrence.Kind, Generated: occurrence.Generated})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SkipRoutineOccurrence(ctx context.Context, req *connect.Request[v.SkipRoutineOccurrenceRequest]) (*connect.Response[v.SkipRoutineOccurrenceResponse], error) {
	if err := h.deps.Service.SkipRoutineOccurrence(ctx, req.Msg.RoutineId, req.Msg.LocalDate, req.Msg.ExpectedRevision); err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.SkipRoutineOccurrenceResponse{Skipped: true}), nil
}

func (h *connectHandler) RescheduleRoutineOccurrence(ctx context.Context, req *connect.Request[v.RescheduleRoutineOccurrenceRequest]) (*connect.Response[v.RescheduleRoutineOccurrenceResponse], error) {
	if err := h.deps.Service.RescheduleRoutineOccurrence(ctx, req.Msg.RoutineId, req.Msg.LocalDate, int64(req.Msg.StartMinute), req.Msg.ExpectedRevision); err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.RescheduleRoutineOccurrenceResponse{Rescheduled: true}), nil
}

func ints(values []int32) []int {
	out := make([]int, len(values))
	for i, value := range values {
		out[i] = int(value)
	}
	return out
}

func routineToProto(r d.Routine) *v.Routine {
	days := make([]int32, len(r.Weekdays))
	for i, day := range r.Weekdays {
		days[i] = int32(day)
	}
	return &v.Routine{Id: r.ID, Title: r.Title, Kind: r.Kind, Timezone: r.Timezone, StartDate: r.StartDate, EndDate: r.EndDate, Weekdays: days, StartMinute: int32(r.StartMinute), DurationMinutes: int32(r.DurationMinutes), FrequencyPerWeek: int32(r.FrequencyPerWeek), Revision: r.Revision, Active: r.Active}
}
