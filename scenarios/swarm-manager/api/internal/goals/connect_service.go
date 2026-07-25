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
//
// It holds the Handler as well as the Service because the workflow apply hop
// lives on the Handler (it needs the workflow invoker and the proposal
// recorder). Keeping both here is what lets close-out and workflow application
// reach the CLI, which previously could only be driven from the UI.
type ConnectService struct {
	service *Service
	handler *Handler
}

func NewConnectService(service *Service, handler *Handler) *ConnectService {
	return &ConnectService{service: service, handler: handler}
}

func RegisterConnectRoutes(router *mux.Router, service *Service, handler *Handler) {
	path, connectHandler := apiconnect.NewGoalServiceHandler(NewConnectService(service, handler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})
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

// CloseOutGoal asserts the delivered goal outcome. Service validation rejects
// incomplete or unverified milestones, so the gate is identical on every
// surface that reaches it.
func (s *ConnectService) CloseOutGoal(_ context.Context, req *connect.Request[apipb.CloseOutGoalRequest]) (*connect.Response[apipb.GoalResponse], error) {
	if _, err := s.service.CloseOut(req.Msg.GetName()); err != nil {
		return nil, goalError(err)
	}
	return s.GetGoal(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: req.Msg.GetName()}))
}

func (s *ConnectService) ListPendingGoalWorkflows(_ context.Context, req *connect.Request[apipb.ListPendingGoalWorkflowsRequest]) (*connect.Response[apipb.ListPendingGoalWorkflowsResponse], error) {
	if s.handler == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("goal workflow surface is not configured"))
	}
	pending, err := s.handler.ListPendingWorkflows()
	if err != nil {
		return nil, internal(err)
	}
	wanted := strings.TrimSpace(req.Msg.GetGoalName())
	out := &apipb.ListPendingGoalWorkflowsResponse{Pending: make([]*apipb.PendingGoalWorkflow, 0, len(pending))}
	for _, record := range pending {
		if wanted != "" && record.GoalName != wanted {
			continue
		}
		out.Pending = append(out.Pending, &apipb.PendingGoalWorkflow{
			GoalName: record.GoalName, ExecutionId: record.ExecutionID, Transition: record.Transition,
			Milestone: record.Milestone, GoalVersion: record.GoalVersion, Stale: record.Stale,
			Attempts: int32(record.Attempts), LastAttemptAt: record.LastAttemptAt, LastError: record.LastError,
		})
	}
	return connect.NewResponse(out), nil
}

func (s *ConnectService) ApplyGoalWorkflow(ctx context.Context, req *connect.Request[apipb.ApplyGoalWorkflowRequest]) (*connect.Response[apipb.ApplyGoalWorkflowResponse], error) {
	if s.handler == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("goal workflow surface is not configured"))
	}
	applied, err := s.handler.ApplyWorkflowResult(ctx, req.Msg.GetGoalName(), req.Msg.GetExecutionId())
	if err != nil {
		if errors.Is(err, ErrWorkflowNotReady) {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&apipb.ApplyGoalWorkflowResponse{
		ExecutionId: applied.ExecutionID, SessionId: applied.SessionID, ProposalIds: applied.ProposalIDs,
		Outcome: applied.Outcome, AlreadyApplied: applied.AlreadyApplied,
	}), nil
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
		out.Milestones = append(out.Milestones, &domainpb.MilestoneRollup{MilestoneName: milestone.Milestone.Name, Total: int32(len(milestone.Items)), Completed: int32(milestone.CompletedCount), Ready: int32(milestone.ReadyCount), Blocked: int32(milestone.BlockedCount), Orphaned: milestone.Orphaned})
	}
	return out
}

func goalResponse(in *GoalWithScope) *apipb.GoalResponse {
	return &apipb.GoalResponse{Goal: goalToProto(in.Goal), Scope: scopeToProto(in.Scope)}
}
