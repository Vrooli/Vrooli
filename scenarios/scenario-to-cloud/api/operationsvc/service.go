// Package operationsvc is the Connect implementation of
// vrooli.scenario_to_cloud.v1.operations.OperationsService. It is a transport
// adapter over operations.Service: the same standing, the same wait and the
// same typed errors (carried as an errors.v1.Error detail) as the REST routes.
package operationsvc

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/operations"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations/operationsv1connect"
)

// Lister is the repository surface the list RPC needs.
type Lister interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
	ListOperationsByDeployment(ctx context.Context, deploymentID string) ([]*domain.CloudOperation, error)
}

// Service implements operationsv1connect.OperationsServiceHandler.
type Service struct {
	ops    *operations.Service
	lister Lister
}

// New builds the service.
func New(ops *operations.Service, lister Lister) *Service {
	return &Service{ops: ops, lister: lister}
}

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return operationsv1connect.NewOperationsServiceHandler(s, opts...)
}

// notConfigured is the refusal when no owner is wired (route-only tests).
func notConfigured() error {
	return deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInternal, "Operation owner is not configured"))
}

func (s *Service) GetOperation(ctx context.Context, req *connect.Request[operationsv1.GetOperationRequest]) (*connect.Response[operationsv1.OperationStanding], error) {
	if s.ops == nil {
		return nil, notConfigured()
	}
	if req.Msg.GetOperationId() == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "operation_id is required"))
	}
	op, err := s.ops.Get(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(StandingProto(operations.StandingOf(op))), nil
}

func (s *Service) WaitOperation(ctx context.Context, req *connect.Request[operationsv1.WaitOperationRequest]) (*connect.Response[operationsv1.OperationStanding], error) {
	if s.ops == nil {
		return nil, notConfigured()
	}
	if req.Msg.GetOperationId() == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "operation_id is required"))
	}
	timeout := time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	op, pending, err := s.ops.Wait(ctx, req.Msg.GetOperationId(), timeout)
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	standing := operations.StandingOf(op)
	if pending {
		standing = standing.Pending(s.ops.Config().LeaseTTL)
	}
	return connect.NewResponse(StandingProto(standing)), nil
}

func (s *Service) CancelOperation(ctx context.Context, req *connect.Request[operationsv1.CancelOperationRequest]) (*connect.Response[operationsv1.OperationStanding], error) {
	if s.ops == nil {
		return nil, notConfigured()
	}
	if req.Msg.GetOperationId() == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "operation_id is required"))
	}
	op, err := s.ops.Cancel(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(StandingProto(operations.StandingOf(op))), nil
}

func (s *Service) ListDeploymentOperations(ctx context.Context, req *connect.Request[operationsv1.ListDeploymentOperationsRequest]) (*connect.Response[operationsv1.ListDeploymentOperationsResponse], error) {
	id := req.Msg.GetDeploymentId()
	if id == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "deployment_id is required"))
	}
	dep, err := s.lister.GetDeployment(ctx, id)
	if err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Internal("Failed to get deployment", err))
	}
	if dep == nil {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeDeploymentNotFound, "Deployment not found").WithDetail("id", id))
	}
	ops, err := s.lister.ListOperationsByDeployment(ctx, id)
	if err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Internal("Failed to list operations", err))
	}
	resp := &operationsv1.ListDeploymentOperationsResponse{SchemaVersion: operations.StandingSchemaVersion, DeploymentId: id}
	for _, op := range ops {
		resp.Operations = append(resp.Operations, StandingProto(operations.StandingOf(op)))
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) ReconcileOperations(ctx context.Context, _ *connect.Request[operationsv1.ReconcileOperationsRequest]) (*connect.Response[operationsv1.ReconcileOperationsResponse], error) {
	if s.ops == nil {
		return nil, notConfigured()
	}
	taken, err := s.ops.Reconcile(ctx)
	if err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Internal("Reconciliation pass failed", err))
	}
	return connect.NewResponse(&operationsv1.ReconcileOperationsResponse{SchemaVersion: operations.StandingSchemaVersion, WorkerId: s.ops.WorkerID(), Acquired: taken}), nil
}

// StandingProto converts the standing to its proto form.
func StandingProto(st operations.Standing) *operationsv1.OperationStanding {
	msg := &operationsv1.OperationStanding{
		SchemaVersion:               st.SchemaVersion,
		OperationId:                 st.OperationID,
		DeploymentId:                st.DeploymentID,
		RequestKey:                  st.RequestKey,
		PlanDigest:                  st.PlanDigest,
		State:                       string(st.State),
		Terminal:                    st.Terminal,
		Fence:                       st.Fence,
		WorkerId:                    st.WorkerID,
		LeaseExpiresAt:              st.LeaseExpiresAt,
		HeartbeatAt:                 st.HeartbeatAt,
		CancelRequested:             st.CancelRequested,
		ActiveStep:                  st.ActiveStep,
		CompletedSteps:              st.CompletedSteps,
		ReattachCommand:             st.ReattachCommand,
		StillPending:                st.StillPending,
		RecommendedNextCheckSeconds: st.RecommendedNextCheckSeconds,
		CreatedAt:                   st.CreatedAt,
		UpdatedAt:                   st.UpdatedAt,
		TerminalAt:                  st.TerminalAt,
	}
	for _, r := range st.StepReceipts {
		msg.StepReceipts = append(msg.StepReceipts, &operationsv1.StepReceipt{
			Step: r.Step, OwnerOperation: r.OwnerOperation, Outcome: string(r.Outcome), Fence: r.Fence, Replayed: r.Replayed,
			Source: r.Source, Detail: r.Detail, Error: r.Error, StartedAt: r.StartedAt, CompletedAt: r.CompletedAt,
		})
	}
	for _, e := range st.UnknownEffects {
		msg.UnknownEffects = append(msg.UnknownEffects, &operationsv1.UnknownEffect{Step: e.Step, Fence: e.Fence, Reason: e.Reason, Retry: e.Retry, NextAction: e.NextAction, RecordedAt: e.RecordedAt})
	}
	if st.Result != nil {
		msg.Result = &operationsv1.OperationResult{Outcome: st.Result.Outcome, RecoveryOutcome: st.Result.RecoveryOutcome, CompletedSteps: int32(st.Result.CompletedSteps), Message: st.Result.Message}
	}
	if st.Error != nil {
		msg.Error = deploymentsvc.ErrorProto(st.Error)
	}
	if st.NextAction != nil {
		msg.NextAction = &errorsv1.NextAction{Owner: st.NextAction.Owner, Kind: st.NextAction.Kind, Reference: st.NextAction.Reference, Label: st.NextAction.Label}
	}
	return msg
}
