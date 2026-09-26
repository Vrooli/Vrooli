package work

import (
	"context"
	"log"

	"connectrpc.com/connect"

	workv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work"

	"personal-planner/internal/work"
)

type (
	Deps struct {
		Service work.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) ListWorkItems(ctx context.Context, _ *connect.Request[workv1.ListWorkItemsRequest]) (*connect.Response[workv1.ListWorkItemsResponse], error) {
	items, err := h.deps.Service.List(ctx, 0)
	if err != nil {
		h.deps.Logger.Printf("work.ListWorkItems: %v", err)
		return nil, work.ToConnectError(err)
	}
	out := &workv1.ListWorkItemsResponse{WorkItems: make([]*workv1.WorkItem, 0, len(items))}
	for _, item := range items {
		out.WorkItems = append(out.WorkItems, domainToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateWorkItem(ctx context.Context, req *connect.Request[workv1.CreateWorkItemRequest]) (*connect.Response[workv1.CreateWorkItemResponse], error) {
	item, err := h.deps.Service.Create(ctx, work.CreateInput{Title: req.Msg.Title, Description: req.Msg.Description, RemainingMinutes: int(req.Msg.RemainingMinutes), SourceLabel: req.Msg.SourceLabel})
	if err != nil {
		return nil, work.ToConnectError(err)
	}
	return connect.NewResponse(&workv1.CreateWorkItemResponse{WorkItem: domainToProto(item)}), nil
}

func (h *connectHandler) GetWorkItem(ctx context.Context, req *connect.Request[workv1.GetWorkItemRequest]) (*connect.Response[workv1.GetWorkItemResponse], error) {
	item, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, work.ToConnectError(err)
	}
	return connect.NewResponse(&workv1.GetWorkItemResponse{WorkItem: domainToProto(item)}), nil
}

func (h *connectHandler) GetTodayPlan(ctx context.Context, _ *connect.Request[workv1.GetTodayPlanRequest]) (*connect.Response[workv1.GetTodayPlanResponse], error) {
	plan, err := h.deps.Service.TodayPlan(ctx)
	if err != nil {
		h.deps.Logger.Printf("work.GetTodayPlan: %v", err)
		return nil, work.ToConnectError(err)
	}
	out := &workv1.GetTodayPlanResponse{PlannedMinutes: int32(plan.PlannedMinutes), AvailableMinutes: int32(plan.AvailableMinutes), BreathingRoomMinutes: int32(plan.BreathingRoom), Entries: make([]*workv1.TodayPlanEntry, 0, len(plan.Entries))}
	for _, entry := range plan.Entries {
		out.Entries = append(out.Entries, &workv1.TodayPlanEntry{WorkItemId: entry.WorkItemID, Title: entry.Title, SourceLabel: entry.SourceLabel, StartMinutes: int32(entry.StartMinutes), DurationMinutes: int32(entry.DurationMinutes)})
	}
	return connect.NewResponse(out), nil
}
