package goals

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// ConnectService exposes the canonical goal/milestone contract over Connect.
type ConnectService struct{ service *Service }

func NewConnectService(service *Service) *ConnectService { return &ConnectService{service: service} }

func RegisterConnectRoutes(router *mux.Router, service *Service) {
	path, handler := apiconnect.NewGoalServiceHandler(NewConnectService(service))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

func (s *ConnectService) ListGoals(context.Context, *connect.Request[apipb.ListGoalsRequest]) (*connect.Response[apipb.ListGoalsResponse], error) {
	goals, err := s.service.List()
	if err != nil {
		return nil, internal(err)
	}
	out := &apipb.ListGoalsResponse{Goals: make([]*apipb.GoalResponse, 0, len(goals))}
	for _, goal := range goals {
		out.Goals = append(out.Goals, goalResponse(&goal))
	}
	return connect.NewResponse(out), nil
}

func (s *ConnectService) GetGoal(_ context.Context, req *connect.Request[apipb.GetGoalRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.Get(req.Msg.GetName())
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) CreateGoal(_ context.Context, req *connect.Request[apipb.CreateGoalRequest]) (*connect.Response[apipb.GoalResponse], error) {
	in := req.Msg
	goal, err := s.service.Create(CreateRequest{Name: in.Name, Title: in.Title, Description: in.Description, Priority: int(in.Priority), Targets: in.Targets})
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) UpdateGoal(_ context.Context, req *connect.Request[apipb.UpdateGoalRequest]) (*connect.Response[apipb.GoalResponse], error) {
	in := req.Msg
	goal, err := s.service.Update(in.Name, UpdateRequest{Title: in.Title, Description: in.Description, Priority: int32Ptr(in.Priority)})
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) DeleteGoal(_ context.Context, req *connect.Request[apipb.DeleteGoalRequest]) (*connect.Response[apipb.EmptyGoalResponse], error) {
	if err := s.service.Delete(req.Msg.GetName()); err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(&apipb.EmptyGoalResponse{}), nil
}

func (s *ConnectService) ArchiveGoal(_ context.Context, req *connect.Request[apipb.ArchiveGoalRequest]) (*connect.Response[apipb.GoalResponse], error) {
	if _, err := s.service.Archive(req.Msg.GetName()); err != nil {
		return nil, goalError(err)
	}
	return s.GetGoal(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: req.Msg.GetName()}))
}

func (s *ConnectService) UnarchiveGoal(_ context.Context, req *connect.Request[apipb.UnarchiveGoalRequest]) (*connect.Response[apipb.GoalResponse], error) {
	if _, err := s.service.Unarchive(req.Msg.GetName()); err != nil {
		return nil, goalError(err)
	}
	return s.GetGoal(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: req.Msg.GetName()}))
}

func (s *ConnectService) AddTargets(_ context.Context, req *connect.Request[apipb.UpdateGoalTargetsRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.AddTargets(req.Msg.Name, req.Msg.Targets)
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) RemoveTargets(_ context.Context, req *connect.Request[apipb.UpdateGoalTargetsRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.RemoveTargets(req.Msg.Name, req.Msg.Targets)
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) CreateMilestone(_ context.Context, req *connect.Request[apipb.CreateMilestoneRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.CreateMilestone(req.Msg.GoalName, milestoneFromProto(req.Msg.Milestone))
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) UpdateMilestone(_ context.Context, req *connect.Request[apipb.UpdateMilestoneRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.UpdateMilestone(req.Msg.GoalName, milestoneFromProto(req.Msg.Milestone))
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) ArchiveMilestone(_ context.Context, req *connect.Request[apipb.ArchiveMilestoneRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.ArchiveMilestone(req.Msg.GoalName, req.Msg.MilestoneName)
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) AssignMilestoneItems(_ context.Context, req *connect.Request[apipb.UpdateMilestoneItemsRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.AssignMilestoneItems(req.Msg.GoalName, req.Msg.MilestoneName, req.Msg.Items)
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) UnassignMilestoneItems(_ context.Context, req *connect.Request[apipb.UpdateMilestoneItemsRequest]) (*connect.Response[apipb.GoalResponse], error) {
	goal, err := s.service.UnassignMilestoneItems(req.Msg.GoalName, req.Msg.MilestoneName, req.Msg.Items)
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(goalResponse(goal)), nil
}

func (s *ConnectService) GetScope(_ context.Context, req *connect.Request[apipb.GetGoalRequest]) (*connect.Response[apipb.GoalScopeResponse], error) {
	goal, err := s.service.Get(req.Msg.GetName())
	if err != nil {
		return nil, goalError(err)
	}
	return connect.NewResponse(&apipb.GoalScopeResponse{Scope: scopeToProto(goal.Scope)}), nil
}

func int32Ptr(value *int32) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}

func goalError(err error) error {
	if errors.Is(err, ErrValidation) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil && strings.Contains(err.Error(), "not found") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return internal(err)
}
func internal(err error) error { return connect.NewError(connect.CodeInternal, err) }
func milestoneFromProto(in *domainpb.Milestone) Milestone {
	if in == nil {
		return Milestone{}
	}
	return Milestone{Name: in.Name, Title: in.Title, Description: in.Description, Items: in.Items, AcceptanceCriteria: in.AcceptanceCriteria, DependsOn: in.DependsOn, ArchivedAt: in.ArchivedAt}
}

func goalToProto(in Goal) *domainpb.Goal {
	out := &domainpb.Goal{Name: in.Name, Title: in.Title, Description: in.Description, Status: in.Status, Priority: int32(in.Priority), Targets: in.Targets, Created: in.Created, Updated: in.Updated, ArchivedAt: in.ArchivedAt}
	for _, milestone := range in.Milestones {
		out.Milestones = append(out.Milestones, &domainpb.Milestone{Name: milestone.Name, Title: milestone.Title, Description: milestone.Description, Items: milestone.Items, AcceptanceCriteria: milestone.AcceptanceCriteria, DependsOn: milestone.DependsOn, ArchivedAt: milestone.ArchivedAt})
	}
	return out
}

func scopeToProto(in Scope) *domainpb.GoalScope {
	out := &domainpb.GoalScope{Targets: in.Targets, Closure: in.Closure, Completed: in.Completed, Ready: in.Ready, Blocked: in.Blocked, Unassigned: in.Unassigned}
	for _, milestone := range in.Milestones {
		out.Milestones = append(out.Milestones, &domainpb.MilestoneRollup{MilestoneName: milestone.Milestone.Name, Total: int32(len(milestone.Items)), Completed: int32(milestone.CompletedCount), Ready: int32(milestone.ReadyCount), Blocked: int32(milestone.BlockedCount)})
	}
	return out
}

func goalResponse(in *GoalWithScope) *apipb.GoalResponse {
	return &apipb.GoalResponse{Goal: goalToProto(in.Goal), Scope: scopeToProto(in.Scope)}
}
