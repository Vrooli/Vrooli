package workspaces

import (
	"context"

	"react-component-library/internal/workspaces"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func (h *Handler) ListWorkspaceAssets(ctx context.Context, req *connect.Request[workspacev1.ListWorkspaceAssetsRequest]) (*connect.Response[workspacev1.ListWorkspaceAssetsResponse], error) {
	if err := requireID(req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	assets, err := h.repository.ListWorkspaceAssets(ctx, req.Msg.GetWorkspaceId())
	if err != nil {
		return nil, h.mapError("list workspace assets", err)
	}
	return connect.NewResponse(&workspacev1.ListWorkspaceAssetsResponse{Assets: workspaceAssetMessages(assets)}), nil
}

func (h *Handler) ListAllWorkspaceAssets(ctx context.Context, _ *connect.Request[workspacev1.ListAllWorkspaceAssetsRequest]) (*connect.Response[workspacev1.ListAllWorkspaceAssetsResponse], error) {
	assets, err := h.repository.ListAllWorkspaceAssets(ctx)
	if err != nil {
		return nil, h.mapError("list all workspace assets", err)
	}
	return connect.NewResponse(&workspacev1.ListAllWorkspaceAssetsResponse{Assets: workspaceAssetMessages(assets)}), nil
}

func (h *Handler) CreateWorkspaceAsset(ctx context.Context, req *connect.Request[workspacev1.CreateWorkspaceAssetRequest]) (*connect.Response[workspacev1.CreateWorkspaceAssetResponse], error) {
	msg := req.Msg
	if err := requireID(msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	asset, err := h.repository.CreateWorkspaceAsset(ctx, workspaces.AssetInput{WorkspaceID: msg.GetWorkspaceId(), Name: msg.GetName(), Mime: msg.GetMime(), Data: msg.GetData(), Fidelity: msg.GetFidelity()})
	if err != nil {
		return nil, h.mapError("create workspace asset", err)
	}
	return connect.NewResponse(&workspacev1.CreateWorkspaceAssetResponse{Asset: workspaceAssetMessage(asset)}), nil
}

func (h *Handler) DeleteWorkspaceAsset(ctx context.Context, req *connect.Request[workspacev1.DeleteWorkspaceAssetRequest]) (*connect.Response[workspacev1.DeleteWorkspaceAssetResponse], error) {
	msg := req.Msg
	if err := requireID(msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if err := requireID(msg.GetAssetId()); err != nil {
		return nil, err
	}
	if err := h.repository.DeleteWorkspaceAsset(ctx, msg.GetWorkspaceId(), msg.GetAssetId()); err != nil {
		return nil, h.mapError("delete workspace asset", err)
	}
	return connect.NewResponse(&workspacev1.DeleteWorkspaceAssetResponse{Deleted: true}), nil
}

func workspaceAssetMessages(assets []workspaces.Asset) []*workspacev1.WorkspaceAsset {
	result := make([]*workspacev1.WorkspaceAsset, 0, len(assets))
	for _, asset := range assets {
		result = append(result, workspaceAssetMessage(asset))
	}
	return result
}

func workspaceAssetMessage(asset workspaces.Asset) *workspacev1.WorkspaceAsset {
	return &workspacev1.WorkspaceAsset{Id: asset.ID, WorkspaceId: asset.WorkspaceID, Name: asset.Name, Mime: asset.Mime, Bytes: asset.Bytes, Width: int32(asset.Width), Height: int32(asset.Height), Fidelity: asset.Fidelity, AddedAt: asset.AddedAt, Url: asset.URL}
}
