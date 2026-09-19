package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/jobs"
	"nutrition-planner/internal/entitlements"
	internalJobs "nutrition-planner/internal/jobs"
	internalWorkspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	repo         internalJobs.Repository
	workspaces   internalWorkspace.Service
	logger       *log.Logger
	entitlements entitlements.Repository
}

func NewConnectHandler(repo internalJobs.Repository, workspaces internalWorkspace.Service, logger *log.Logger, entitlementRepos ...entitlements.Repository) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	var entitlementRepo entitlements.Repository
	if len(entitlementRepos) > 0 {
		entitlementRepo = entitlementRepos[0]
	}
	return &connectHandler{repo: repo, workspaces: workspaces, logger: logger, entitlements: entitlementRepo}
}

func (h *connectHandler) ListJobs(ctx context.Context, req *connect.Request[v1.ListJobsRequest]) (*connect.Response[v1.ListJobsResponse], error) {
	if _, err := h.authorize(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.Job, 0, len(items))
	for _, item := range items {
		out = append(out, toProto(item))
	}
	return connect.NewResponse(&v1.ListJobsResponse{Jobs: out}), nil
}

func (h *connectHandler) CreateJob(ctx context.Context, req *connect.Request[v1.CreateJobRequest]) (*connect.Response[v1.CreateJobResponse], error) {
	if _, err := h.authorize(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	if h.entitlements != nil && req.Msg.BudgetUnits > 0 {
		state, err := h.entitlements.Get(ctx, req.Msg.WorkspaceId)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if err := state.CanUseOptional(int64(req.Msg.BudgetUnits)); err != nil {
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}
	}
	hashInput, _ := json.Marshal(req.Msg)
	hash := sha256.Sum256(hashInput)
	item, err := h.repo.Create(ctx, internalJobs.Job{ID: req.Msg.Id, WorkspaceID: req.Msg.WorkspaceId, Type: req.Msg.Type, DedupKey: req.Msg.DedupKey, RequestHash: hex.EncodeToString(hash[:]), InputRevisions: req.Msg.InputRevisions, Provider: req.Msg.Provider, BudgetUnits: int(req.Msg.BudgetUnits)})
	if err != nil {
		var conflict internalJobs.ErrIdempotencyConflict
		if errors.As(err, &conflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.entitlements != nil && req.Msg.BudgetUnits > 0 {
		if _, err := h.entitlements.ReserveOptional(ctx, req.Msg.WorkspaceId, "job:"+item.ID, int64(req.Msg.BudgetUnits)); err != nil {
			_, _ = h.repo.Transition(ctx, internalJobs.TransitionInput{WorkspaceID: req.Msg.WorkspaceId, ID: item.ID, NextState: internalJobs.Canceled, ErrorCode: "entitlement_denied"})
			var budget entitlements.BudgetError
			if errors.As(err, &budget) {
				return nil, connect.NewError(connect.CodeResourceExhausted, err)
			}
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	return connect.NewResponse(&v1.CreateJobResponse{Job: toProto(item)}), nil
}

func (h *connectHandler) TransitionJob(ctx context.Context, req *connect.Request[v1.TransitionJobRequest]) (*connect.Response[v1.TransitionJobResponse], error) {
	if _, err := h.authorize(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	item, err := h.repo.Transition(ctx, internalJobs.TransitionInput{WorkspaceID: req.Msg.WorkspaceId, ID: req.Msg.Id, NextState: internalJobs.State(req.Msg.NextState), Provider: req.Msg.Provider, Attempts: int(req.Msg.Attempts), ResultReference: req.Msg.ResultReference, ErrorCode: req.Msg.ErrorCode})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.TransitionJobResponse{Job: toProto(item)}), nil
}

func (h *connectHandler) CancelJob(ctx context.Context, req *connect.Request[v1.CancelJobRequest]) (*connect.Response[v1.CancelJobResponse], error) {
	if _, err := h.authorize(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	item, err := h.repo.Transition(ctx, internalJobs.TransitionInput{WorkspaceID: req.Msg.WorkspaceId, ID: req.Msg.Id, NextState: internalJobs.Canceled})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.CancelJobResponse{Job: toProto(item)}), nil
}

func (h *connectHandler) authorize(ctx context.Context, workspaceID string) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if workspaceID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, workspaceID, principal.Subject); err != nil {
		var nf internalWorkspace.ErrNotFound
		if errors.As(err, &nf) {
			return "", connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden internalWorkspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		return "", connect.NewError(connect.CodeInternal, err)
	}
	return principal.Subject, nil
}

func toProto(item internalJobs.Job) *v1.Job {
	return &v1.Job{Id: item.ID, WorkspaceId: item.WorkspaceID, Type: item.Type, DedupKey: item.DedupKey, State: string(item.State), InputRevisions: item.InputRevisions, Attempts: int32(item.Attempts), Provider: item.Provider, BudgetUnits: int32(item.BudgetUnits), ResultReference: item.ResultReference, ErrorCode: item.ErrorCode}
}
