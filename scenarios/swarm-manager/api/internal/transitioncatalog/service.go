// Package transitioncatalog exposes the immutable transition declarations to
// typed consumers. It translates the registry only; transition execution stays
// in the domain adapters and runner.
package transitioncatalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"
)

type Runner interface {
	Start(context.Context, string, string) (transitionrun.Correlation, error)
	Apply(context.Context, string, string) (transitionrun.Correlation, error)
}

type DeterministicDispatcher interface {
	Dispatch(context.Context, string, string) (string, error)
}

type Service struct {
	registry      transitions.Registry
	runner        Runner
	deterministic DeterministicDispatcher
}

func NewService(registry transitions.Registry, runner Runner, deterministic ...DeterministicDispatcher) *Service {
	svc := &Service{registry: registry, runner: runner}
	if len(deterministic) > 0 {
		svc.deterministic = deterministic[0]
	}
	return svc
}

func RegisterRoutes(router *mux.Router, registry transitions.Registry, runner Runner, deterministic ...DeterministicDispatcher) {
	path, handler := apiconnect.NewTransitionServiceHandler(NewService(registry, runner, deterministic...))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

func (s *Service) StartTransition(ctx context.Context, req *connect.Request[api.StartTransitionRequest]) (*connect.Response[api.StartTransitionResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetTransitionKey()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("transition_key is required"))
	}
	definition, ok := s.registry.Get(req.Msg.GetTransitionKey())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("transition is not declared"))
	}
	subjectRef, err := declaredSubjectReference(definition, req.Msg.GetSubjectRef())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if definition.Kind == transitions.KindDeterministic {
		if s.deterministic == nil {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("deterministic transition dispatcher is not configured"))
		}
		outcome, err := s.deterministic.Dispatch(ctx, definition.Key, subjectRef)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return connect.NewResponse(&api.StartTransitionResponse{ExecutionId: subjectRef, EntityVersion: outcome, ApplyState: "complete", Outcome: outcome}), nil
	}
	if definition.Kind != transitions.KindWorkflow {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("session transitions remain on the Agent Session surface"))
	}
	if s.runner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("transition runner is not configured"))
	}
	correlation, err := s.runner.Start(ctx, req.Msg.GetTransitionKey(), subjectRef)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&api.StartTransitionResponse{
		ExecutionId: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest, EntityVersion: correlation.EntityVersion,
		ApplyState: string(correlation.ApplyState), Outcome: correlation.Outcome, TerminalCode: correlation.TerminalCode,
	}), nil
}

func (s *Service) ApplyTransition(ctx context.Context, req *connect.Request[api.ApplyTransitionRequest]) (*connect.Response[api.ApplyTransitionResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetTransitionKey()) == "" || strings.TrimSpace(req.Msg.GetExecutionId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("transition_key and execution_id are required"))
	}
	if s.runner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("transition runner is not configured"))
	}
	correlation, err := s.runner.Apply(ctx, req.Msg.GetTransitionKey(), req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&api.ApplyTransitionResponse{
		ExecutionId: correlation.ExecutionID, TransitionKey: correlation.TransitionKey, SubjectRef: correlation.SubjectRef,
		Outcome: correlation.Outcome, TerminalCode: correlation.TerminalCode, AppliedTime: correlation.AppliedTime,
		DefinitionDigest: correlation.DefinitionDigest, EntityVersion: correlation.EntityVersion, ApplyState: string(correlation.ApplyState),
	}), nil
}

func declaredSubjectReference(definition transitions.Definition, reference *api.SubjectReference) (string, error) {
	if reference == nil || strings.TrimSpace(reference.GetSubject()) == "" || strings.TrimSpace(reference.GetValue()) == "" {
		return "", errors.New("subject_ref.subject and subject_ref.value are required")
	}
	if reference.GetSubject() != definition.Subject {
		return "", fmt.Errorf("transition %q requires subject %q, got %q", definition.Key, definition.Subject, reference.GetSubject())
	}
	return reference.GetValue(), nil
}

func (s *Service) ListTransitions(_ context.Context, req *connect.Request[api.ListTransitionsRequest]) (*connect.Response[api.ListTransitionsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response := &api.ListTransitionsResponse{Transitions: make([]*domain.Transition, 0, len(s.registry.Definitions()))}
	for _, definition := range s.registry.Definitions() {
		response.Transitions = append(response.Transitions, definitionProto(definition))
	}
	return connect.NewResponse(response), nil
}

func definitionProto(definition transitions.Definition) *domain.Transition {
	transition := &domain.Transition{
		Key: definition.Key, Subject: definition.Subject, Kind: kindProto(definition.Kind),
		Requires: definition.Requires, InputContract: definition.InputContract,
		TerminalOutcomes: definition.TerminalOutcomes, ApplyAction: definition.ApplyAction,
	}
	if definition.Workflow != nil {
		transition.Workflow = &domain.WorkflowLocator{Owner: definition.Workflow.Owner, Key: definition.Workflow.Key}
	}
	for _, strategy := range definition.Strategies {
		transition.Strategies = append(transition.Strategies, &domain.ExecutionStrategy{Id: strategy.ID, WorkflowKey: strategy.WorkflowKey, DisplayName: strategy.DisplayName, Description: strategy.Description, WhenToUse: strategy.WhenToUse, CostBand: strategy.CostBand})
	}
	return transition
}

func kindProto(kind transitions.Kind) domain.TransitionKind {
	switch kind {
	case transitions.KindSession:
		return domain.TransitionKind_TRANSITION_KIND_SESSION
	case transitions.KindWorkflow:
		return domain.TransitionKind_TRANSITION_KIND_WORKFLOW
	case transitions.KindDeterministic:
		return domain.TransitionKind_TRANSITION_KIND_DETERMINISTIC
	default:
		return domain.TransitionKind_TRANSITION_KIND_UNSPECIFIED
	}
}
