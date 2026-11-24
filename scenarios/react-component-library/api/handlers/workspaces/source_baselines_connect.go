package workspaces

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	domain "react-component-library/internal/workspaces"
)

func (h *Handler) ListWorkspaceSourceBaselines(ctx context.Context, req *connect.Request[workspacev1.ListWorkspaceSourceBaselinesRequest]) (*connect.Response[workspacev1.ListWorkspaceSourceBaselinesResponse], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace and state ids are invalid"))
	}
	w, err := h.repository.Get(ctx, m.GetWorkspaceId())
	if err != nil {
		return nil, h.mapError("load workspace source baselines", err)
	}
	if !workspaceHasState(w, m.GetStateId()) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state was not found"))
	}
	items, err := h.repository.ListSourceBaselines(ctx, m.GetWorkspaceId(), m.GetStateId())
	if err != nil {
		return nil, h.mapError("list workspace source baselines", err)
	}
	response := &workspacev1.ListWorkspaceSourceBaselinesResponse{Candidates: make([]*workspacev1.WorkspaceSourceBaseline, 0)}
	for _, item := range items {
		if item.AcceptedAt == "" {
			response.Candidates = append(response.Candidates, sourceBaselineMessage(item))
		}
	}
	accepted, err := h.repository.GetAcceptedSourceBaseline(ctx, m.GetWorkspaceId(), m.GetStateId())
	if err == nil {
		response.Accepted = sourceBaselineMessage(accepted)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, h.mapError("load accepted workspace source baseline", err)
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) CreateWorkspaceSourceBaseline(ctx context.Context, req *connect.Request[workspacev1.CreateWorkspaceSourceBaselineRequest]) (*connect.Response[workspacev1.WorkspaceSourceBaseline], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) || m.GetAssetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace, state, and reference asset ids are required"))
	}
	asset, data, err := h.repository.GetWorkspaceAsset(ctx, m.GetWorkspaceId(), m.GetAssetId())
	if err != nil {
		return nil, h.mapError("load source reference image", err)
	}
	if asset.Fidelity != domain.AssetFidelityReference || (asset.Mime != "image/png" && asset.Mime != "image/jpeg") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("source baseline must be a PNG or JPEG reference image"))
	}
	digest := sha256.Sum256(data)
	item, err := h.repository.CreateSourceBaseline(ctx, m.GetWorkspaceId(), m.GetStateId(), m.GetAssetId(), hex.EncodeToString(digest[:]), m.GetSourceLocator(), time.Now().UTC().Format(time.RFC3339Nano))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("workspace state was not found"))
	}
	if err != nil {
		return nil, h.mapError("create workspace source baseline", err)
	}
	return connect.NewResponse(sourceBaselineMessage(item)), nil
}

func (h *Handler) AcceptWorkspaceSourceBaseline(ctx context.Context, req *connect.Request[workspacev1.AcceptWorkspaceSourceBaselineRequest]) (*connect.Response[workspacev1.WorkspaceSourceBaseline], error) {
	m := req.Msg
	if !uuidLikeWorkspaceID(m.GetWorkspaceId()) || !uuidLikeWorkspaceID(m.GetStateId()) || m.GetBaselineId() == "" || m.GetAcceptedBy() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace, state, baseline, and human attribution are required"))
	}
	item, err := h.repository.AcceptSourceBaseline(ctx, m.GetWorkspaceId(), m.GetStateId(), m.GetBaselineId(), m.GetAcceptedBy(), time.Now().UTC().Format(time.RFC3339Nano))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("source baseline candidate was not found"))
	}
	if err != nil {
		return nil, h.mapError("accept workspace source baseline", err)
	}
	return connect.NewResponse(sourceBaselineMessage(item)), nil
}

func sourceBaselineMessage(item domain.SourceBaseline) *workspacev1.WorkspaceSourceBaseline {
	return &workspacev1.WorkspaceSourceBaseline{Id: item.ID, WorkspaceId: item.WorkspaceID, StateId: item.StateID, AssetId: item.AssetID, Sha256: item.SHA256, SourceLocator: item.SourceLocator, CreatedAt: item.CreatedAt, AcceptedAt: item.AcceptedAt, AcceptedBy: item.AcceptedBy}
}
