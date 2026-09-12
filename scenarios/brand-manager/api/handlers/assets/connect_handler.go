package assets

import (
	"context"
	"log"

	"brand-manager/internal/assets"

	"connectrpc.com/connect"

	assetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets"
)

// Deps wires the seams the Connect assets handler needs.
type Deps struct {
	Service assets.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC assets handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListAssets(ctx context.Context, req *connect.Request[assetsv1.ListAssetsRequest]) (*connect.Response[assetsv1.ListAssetsResponse], error) {
	results, err := h.deps.Service.List(ctx, req.Msg.GetBrandId())
	if err != nil {
		h.deps.Logger.Printf("assets.ListAssets: %v", err)
		return nil, assets.ToConnectError(err)
	}
	resp := &assetsv1.ListAssetsResponse{Assets: make([]*assetsv1.Asset, 0, len(results))}
	for _, a := range results {
		resp.Assets = append(resp.Assets, domainToProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) UploadAsset(ctx context.Context, req *connect.Request[assetsv1.UploadAssetRequest]) (*connect.Response[assetsv1.UploadAssetResponse], error) {
	uploaded, err := h.deps.Service.Upload(ctx, assets.UploadInput{
		BrandID:  req.Msg.GetBrandId(),
		Filename: req.Msg.GetFilename(),
		MimeType: req.Msg.GetMimeType(),
		Content:  req.Msg.GetContent(),
	})
	if err != nil {
		return nil, h.translate("assets.UploadAsset", err)
	}
	return connect.NewResponse(&assetsv1.UploadAssetResponse{Asset: domainToProto(uploaded)}), nil
}

func (h *connectHandler) GetAsset(ctx context.Context, req *connect.Request[assetsv1.GetAssetRequest]) (*connect.Response[assetsv1.GetAssetResponse], error) {
	got, err := h.deps.Service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.translate("assets.GetAsset", err)
	}
	return connect.NewResponse(&assetsv1.GetAssetResponse{Asset: domainToProto(got)}), nil
}

func (h *connectHandler) DownloadAsset(ctx context.Context, req *connect.Request[assetsv1.DownloadAssetRequest]) (*connect.Response[assetsv1.DownloadAssetResponse], error) {
	content, err := h.deps.Service.Download(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.translate("assets.DownloadAsset", err)
	}
	return connect.NewResponse(&assetsv1.DownloadAssetResponse{
		Filename: content.Filename,
		MimeType: content.MimeType,
		Content:  content.Bytes,
	}), nil
}

func (h *connectHandler) DeleteAsset(ctx context.Context, req *connect.Request[assetsv1.DeleteAssetRequest]) (*connect.Response[assetsv1.DeleteAssetResponse], error) {
	if err := h.deps.Service.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, h.translate("assets.DeleteAsset", err)
	}
	return connect.NewResponse(&assetsv1.DeleteAssetResponse{}), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault 4xx-equivalent codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := assets.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
