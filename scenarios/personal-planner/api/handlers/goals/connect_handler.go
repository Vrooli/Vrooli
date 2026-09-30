package goals

import (
	"context"
	"log"

	"connectrpc.com/connect"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	g "personal-planner/internal/goals"
)

type (
	Deps struct {
		Service g.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListGoals(c context.Context, _ *connect.Request[v.ListGoalsRequest]) (*connect.Response[v.ListGoalsResponse], error) {
	xs, e := h.deps.Service.List(c)
	if e != nil {
		return nil, g.ToConnectError(e)
	}
	out := &v.ListGoalsResponse{Goals: make([]*v.Goal, 0, len(xs))}
	for _, x := range xs {
		out.Goals = append(out.Goals, toProto(x))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateGoal(c context.Context, r *connect.Request[v.CreateGoalRequest]) (*connect.Response[v.CreateGoalResponse], error) {
	x, e := h.deps.Service.Create(c, g.CreateInput{Title: r.Msg.Title, Purpose: r.Msg.Purpose, ProgressMethod: r.Msg.ProgressMethod, TargetBasisPoints: r.Msg.TargetBasisPoints})
	if e != nil {
		return nil, g.ToConnectError(e)
	}
	return connect.NewResponse(&v.CreateGoalResponse{Goal: toProto(x)}), nil
}

func (h *connectHandler) UpdateGoalProgress(c context.Context, r *connect.Request[v.UpdateGoalProgressRequest]) (*connect.Response[v.UpdateGoalProgressResponse], error) {
	x, e := h.deps.Service.UpdateProgress(c, r.Msg.Id, r.Msg.ProgressBasisPoints, r.Msg.ExpectedRevision)
	if e != nil {
		return nil, g.ToConnectError(e)
	}
	return connect.NewResponse(&v.UpdateGoalProgressResponse{Goal: toProto(x)}), nil
}

func (h *connectHandler) ListMilestones(c context.Context, r *connect.Request[v.ListMilestonesRequest]) (*connect.Response[v.ListMilestonesResponse], error) {
	xs, e := h.deps.Service.ListMilestones(c, r.Msg.GoalId)
	if e != nil {
		return nil, g.ToConnectError(e)
	}
	out := &v.ListMilestonesResponse{Milestones: make([]*v.Milestone, 0, len(xs))}
	for _, x := range xs {
		out.Milestones = append(out.Milestones, toMilestoneProto(x))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateMilestone(c context.Context, r *connect.Request[v.CreateMilestoneRequest]) (*connect.Response[v.CreateMilestoneResponse], error) {
	x, e := h.deps.Service.CreateMilestone(c, g.CreateMilestoneInput{GoalID: r.Msg.GoalId, Title: r.Msg.Title, Criteria: r.Msg.Criteria, DueDate: r.Msg.DueDate, LinkedWorkItemID: r.Msg.LinkedWorkItemId, PrerequisiteMilestoneIDs: r.Msg.PrerequisiteMilestoneIds})
	if e != nil {
		return nil, g.ToConnectError(e)
	}
	return connect.NewResponse(&v.CreateMilestoneResponse{Milestone: toMilestoneProto(x)}), nil
}

func (h *connectHandler) UpdateMilestoneStatus(c context.Context, r *connect.Request[v.UpdateMilestoneStatusRequest]) (*connect.Response[v.UpdateMilestoneStatusResponse], error) {
	x, e := h.deps.Service.UpdateMilestoneStatus(c, r.Msg.Id, r.Msg.Status, r.Msg.ExpectedRevision)
	if e != nil {
		return nil, g.ToConnectError(e)
	}
	return connect.NewResponse(&v.UpdateMilestoneStatusResponse{Milestone: toMilestoneProto(x)}), nil
}

func toMilestoneProto(m g.Milestone) *v.Milestone {
	return &v.Milestone{Id: m.ID, GoalId: m.GoalID, Title: m.Title, Criteria: m.Criteria, DueDate: m.DueDate, Status: m.Status, Revision: m.Revision, LinkedWorkItemId: m.LinkedWorkItemID, PrerequisiteMilestoneIds: m.PrerequisiteMilestoneIDs}
}
