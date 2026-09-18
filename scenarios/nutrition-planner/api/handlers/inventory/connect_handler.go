package inventory

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/inventory"
	"log"
	"nutrition-planner/internal/decimalx"
	internal "nutrition-planner/internal/inventory"
	"nutrition-planner/internal/workspace"
	"time"
)

type connectHandler struct {
	repo   internal.Repository
	ws     workspace.Service
	logger *log.Logger
}

func NewConnectHandler(repo internal.Repository, ws workspace.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{repo: repo, ws: ws, logger: logger}
}

func (h *connectHandler) scope(ctx context.Context, workspaceID string) error {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if workspaceID == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.ws.Get(ctx, workspaceID, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden workspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func (h *connectHandler) ListEvents(ctx context.Context, req *connect.Request[v1.ListEventsRequest]) (*connect.Response[v1.ListEventsResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListEventsResponse{Events: make([]*v1.InventoryEvent, 0, len(items))}
	for _, item := range items {
		out.Events = append(out.Events, toProto(req.Msg.WorkspaceId, item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RecordEvent(ctx context.Context, req *connect.Request[v1.RecordEventRequest]) (*connect.Response[v1.RecordEventResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	amount, err := decimalx.Parse(req.Msg.Amount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	created := time.Now().UTC()
	if req.Msg.CreatedAt != "" {
		created, err = time.Parse(time.RFC3339Nano, req.Msg.CreatedAt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("created_at must be RFC3339"))
		}
	}
	event := internal.Event{ID: req.Msg.Id, Kind: internal.EventKind(req.Msg.Kind), ItemID: req.Msg.ItemId, BatchID: req.Msg.BatchId, Amount: amount, Unit: req.Msg.Unit, RecipeID: req.Msg.RecipeId, CreatedAt: created}
	if err := h.repo.Append(ctx, req.Msg.WorkspaceId, event); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.RecordEventResponse{Event: toProto(req.Msg.WorkspaceId, event)}), nil
}

func (h *connectHandler) PrepareBatch(ctx context.Context, req *connect.Request[v1.PrepareBatchRequest]) (*connect.Response[v1.PrepareBatchResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	repo, ok := h.repo.(internal.BatchRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("batch inventory is unavailable"))
	}
	yield, err := decimalx.Parse(req.Msg.YieldAmount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	requirements := make([]internal.Event, 0, len(req.Msg.Requirements))
	for _, item := range req.Msg.Requirements {
		amount, parseErr := decimalx.Parse(item.Amount)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}
		requirements = append(requirements, internal.Event{Kind: internal.Preparation, ItemID: item.ItemId, Amount: amount, Unit: item.Unit})
	}
	batch, err := repo.PrepareBatch(ctx, req.Msg.WorkspaceId, req.Msg.EventId, req.Msg.BatchId, req.Msg.RecipeId, req.Msg.RecipeRevision, yield, req.Msg.Unit, requirements)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.PrepareBatchResponse{Batch: batchProto(batch)}), nil
}

func (h *connectHandler) ConsumeBatchPortion(ctx context.Context, req *connect.Request[v1.ConsumeBatchPortionRequest]) (*connect.Response[v1.BatchResponse], error) {
	return h.batchPortion(ctx, req, false)
}

func (h *connectHandler) UndoBatchPortion(ctx context.Context, req *connect.Request[v1.ConsumeBatchPortionRequest]) (*connect.Response[v1.BatchResponse], error) {
	return h.batchPortion(ctx, req, true)
}

func (h *connectHandler) batchPortion(ctx context.Context, req *connect.Request[v1.ConsumeBatchPortionRequest], undo bool) (*connect.Response[v1.BatchResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	repo, ok := h.repo.(internal.BatchRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("batch inventory is unavailable"))
	}
	amount, err := decimalx.Parse(req.Msg.Amount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	batch, err := repo.ConsumeBatchPortion(ctx, req.Msg.WorkspaceId, req.Msg.EventId, req.Msg.BatchId, amount, req.Msg.Unit, req.Msg.RecipeId, undo)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.BatchResponse{Batch: batchProto(batch)}), nil
}

func (h *connectHandler) StageReceiptProposal(ctx context.Context, req *connect.Request[v1.StageReceiptProposalRequest]) (*connect.Response[v1.ReceiptProposalResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	repo, ok := h.repo.(internal.ReceiptRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("receipt proposals are unavailable"))
	}
	amount, err := decimalx.Parse(req.Msg.Amount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	proposal, err := repo.StageReceiptProposal(ctx, req.Msg.WorkspaceId, internal.ReceiptProposal{SourceID: req.Msg.SourceId, TransactionID: req.Msg.TransactionId, LineKey: req.Msg.LineKey, Description: req.Msg.Description, ItemID: req.Msg.ItemId, Amount: amount, Unit: req.Msg.Unit, Price: req.Msg.Price})
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.ReceiptProposalResponse{Proposal: receiptProposalProto(req.Msg.WorkspaceId, proposal)}), nil
}

func (h *connectHandler) ListReceiptProposals(ctx context.Context, req *connect.Request[v1.ListReceiptProposalsRequest]) (*connect.Response[v1.ListReceiptProposalsResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	repo, ok := h.repo.(internal.ReceiptRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("receipt proposals are unavailable"))
	}
	proposals, err := repo.ListReceiptProposals(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListReceiptProposalsResponse{Proposals: make([]*v1.ReceiptProposal, 0, len(proposals))}
	for _, proposal := range proposals {
		out.Proposals = append(out.Proposals, receiptProposalProto(req.Msg.WorkspaceId, proposal))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ApplyReceiptProposal(ctx context.Context, req *connect.Request[v1.ApplyReceiptProposalRequest]) (*connect.Response[v1.ReceiptProposalResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	repo, ok := h.repo.(internal.ReceiptRepository)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("receipt proposals are unavailable"))
	}
	proposal, err := repo.ApplyReceiptProposal(ctx, req.Msg.WorkspaceId, req.Msg.ProposalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.ReceiptProposalResponse{Proposal: receiptProposalProto(req.Msg.WorkspaceId, proposal)}), nil
}

func batchProto(batch internal.Batch) *v1.Batch {
	return &v1.Batch{Id: batch.ID, RecipeId: batch.RecipeID, RecipeRevision: batch.RecipeRevision, YieldAmount: batch.Yield.String(), AvailableAmount: batch.Available.String(), Unit: batch.Unit}
}

func toProto(workspaceID string, v internal.Event) *v1.InventoryEvent {
	return &v1.InventoryEvent{Id: v.ID, WorkspaceId: workspaceID, Kind: string(v.Kind), ItemId: v.ItemID, BatchId: v.BatchID, Amount: v.Amount.String(), Unit: v.Unit, RecipeId: v.RecipeID, CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

func receiptProposalProto(workspaceID string, v internal.ReceiptProposal) *v1.ReceiptProposal {
	out := &v1.ReceiptProposal{Id: v.ID, WorkspaceId: workspaceID, SourceId: v.SourceID, TransactionId: v.TransactionID, LineKey: v.LineKey, Description: v.Description, ItemId: v.ItemID, Amount: v.Amount.String(), Unit: v.Unit, Price: v.Price, Status: string(v.Status), EventId: v.EventID, CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano)}
	if !v.AppliedAt.IsZero() {
		out.AppliedAt = v.AppliedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}
