package calendar

import (
	"context"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/api-core/schedule"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"

	d "personal-planner/internal/calendar"
)

type Deps struct {
	Service d.Service
	Events  d.EventService
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

func (h *connectHandler) PreviewAllocation(ctx context.Context, req *connect.Request[v.PreviewAllocationRequest]) (*connect.Response[v.PreviewAllocationResponse], error) {
	proposal, err := h.deps.Service.Preview(ctx, d.PreviewInput{WorkItemID: req.Msg.WorkItemId, LocalDate: req.Msg.LocalDate, StartMinutes: int(req.Msg.StartMinutes), DurationMinutes: int(req.Msg.DurationMinutes)})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.PreviewAllocationResponse{Proposal: &v.PlacementProposal{Id: proposal.ID, WorkItemId: proposal.WorkItemID, LocalDate: proposal.LocalDate, State: proposal.State, StartMinutes: int32(proposal.StartMinutes), DurationMinutes: int32(proposal.DurationMinutes), Reason: proposal.Reason, BaseRevision: proposal.BaseRevision}}), nil
}

func (h *connectHandler) ApplyAllocationProposal(ctx context.Context, req *connect.Request[v.ApplyAllocationProposalRequest]) (*connect.Response[v.ApplyAllocationProposalResponse], error) {
	a, err := h.deps.Service.ApplyProposal(ctx, d.ApplyProposalInput{ProposalID: req.Msg.ProposalId, ExpectedRevision: req.Msg.ExpectedRevision, IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.ApplyAllocationProposalResponse{Allocation: toProto(a)}), nil
}

func (h *connectHandler) PreviewSchedule(ctx context.Context, req *connect.Request[v.PreviewScheduleRequest]) (*connect.Response[v.PreviewScheduleResponse], error) {
	proposal, err := h.deps.Service.PreviewSchedule(ctx, d.SchedulePreviewInput{LocalDate: req.Msg.LocalDate, StartMinutes: int(req.Msg.StartMinutes), WorkItemIDs: req.Msg.WorkItemIds})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ScheduleProposal{Id: proposal.ID, LocalDate: proposal.LocalDate, BaseRevision: proposal.BaseRevision, State: proposal.State, Reason: proposal.Reason}
	for _, placement := range proposal.Placements {
		out.Placements = append(out.Placements, &v.ProposedPlacement{WorkItemId: placement.WorkItemID, Title: placement.Title, LocalDate: placement.LocalDate, StartMinutes: int32(placement.StartMinutes), DurationMinutes: int32(placement.DurationMinutes), State: placement.State, Reason: placement.Reason})
	}
	return connect.NewResponse(&v.PreviewScheduleResponse{Proposal: out}), nil
}

func (h *connectHandler) ApplyScheduleProposal(ctx context.Context, req *connect.Request[v.ApplyScheduleProposalRequest]) (*connect.Response[v.ApplyScheduleProposalResponse], error) {
	allocations, err := h.deps.Service.ApplyScheduleProposal(ctx, d.ApplyScheduleProposalInput{ProposalID: req.Msg.ProposalId, ExpectedRevision: req.Msg.ExpectedRevision, IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ApplyScheduleProposalResponse{}
	for _, allocation := range allocations {
		out.Allocations = append(out.Allocations, toProto(allocation))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CarryForwardAllocation(ctx context.Context, req *connect.Request[v.CarryForwardAllocationRequest]) (*connect.Response[v.CarryForwardAllocationResponse], error) {
	a, err := h.deps.Service.CarryForward(ctx, d.CarryForwardInput{AllocationID: req.Msg.AllocationId, TargetLocalDate: req.Msg.TargetLocalDate, StartMinutes: int(req.Msg.StartMinutes), ReasonCode: d.RescheduleDeprioritized})
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

func (h *connectHandler) ListEvents(ctx context.Context, req *connect.Request[v.ListEventsRequest]) (*connect.Response[v.ListEventsResponse], error) {
	events, err := h.deps.Events.List(ctx, req.Msg.StartLocalDate, req.Msg.EndLocalDate)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ListEventsResponse{Events: make([]*v.CalendarEvent, 0, len(events))}
	for _, event := range events {
		out.Events = append(out.Events, eventToProto(event))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetEvent(ctx context.Context, req *connect.Request[v.GetEventRequest]) (*connect.Response[v.GetEventResponse], error) {
	event, err := h.deps.Events.Get(ctx, req.Msg.EventId)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.GetEventResponse{Event: eventToProto(event)}), nil
}

func (h *connectHandler) CreateEvent(ctx context.Context, req *connect.Request[v.CreateEventRequest]) (*connect.Response[v.CreateEventResponse], error) {
	event, err := h.deps.Events.Create(ctx, d.CreateEventInput{Event: eventFromProto(req.Msg.Event), IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.CreateEventResponse{Event: eventToProto(event)}), nil
}

func (h *connectHandler) UpdateEvent(ctx context.Context, req *connect.Request[v.UpdateEventRequest]) (*connect.Response[v.UpdateEventResponse], error) {
	event := eventFromProto(req.Msg.Event)
	updated, err := h.deps.Events.Update(ctx, d.UpdateEventInput{Event: event, ExpectedRevision: req.Msg.ExpectedRevision})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.UpdateEventResponse{Event: eventToProto(updated)}), nil
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

func eventToProto(event d.Event) *v.CalendarEvent {
	return &v.CalendarEvent{Id: event.ID, Title: event.Title, Subject: event.Subject, Notes: event.Notes, Availability: event.Availability, Timezone: event.Timezone, AllDay: event.AllDay, StartDate: event.StartDate, EndDateExclusive: event.EndDateExclusive, StartAt: event.StartAt, EndAt: event.EndAt, Provider: event.Provider, ProviderCalendarId: event.ProviderCalendarID, ProviderEventId: event.ProviderEventID, OccurrenceId: event.OccurrenceID, Revision: event.Revision, CreatedAt: timestamppb.New(event.CreatedAt), UpdatedAt: timestamppb.New(event.UpdatedAt)}
}

func eventFromProto(event *v.CalendarEvent) d.Event {
	if event == nil {
		return d.Event{}
	}
	return d.Event{ID: event.Id, Title: event.Title, Subject: event.Subject, Notes: event.Notes, Availability: event.Availability, Timezone: event.Timezone, AllDay: event.AllDay, StartDate: event.StartDate, EndDateExclusive: event.EndDateExclusive, StartAt: event.StartAt, EndAt: event.EndAt, Provider: event.Provider, ProviderCalendarID: event.ProviderCalendarId, ProviderEventID: event.ProviderEventId, OccurrenceID: event.OccurrenceId}
}
