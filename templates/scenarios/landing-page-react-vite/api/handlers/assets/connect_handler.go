package assets

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalassets "landing-page-react-vite-api/internal/assets"
)

// Deps wires the Assets Connect handler.
type Deps struct {
	Service *internalassets.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the AssetsService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListAssets(_ context.Context, req *connect.Request[landingv1.ListAssetsRequest]) (*connect.Response[landingv1.ListAssetsResponse], error) {
	assets, err := h.deps.Service.List(req.Msg.Category)
	if err != nil {
		h.deps.Logger.Printf("assets.ListAssets: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*landingv1.Asset, 0, len(assets))
	for i := range assets {
		out = append(out, assetToProto(&assets[i]))
	}
	return connect.NewResponse(&landingv1.ListAssetsResponse{Assets: out}), nil
}

func (h *connectHandler) GetAsset(_ context.Context, req *connect.Request[landingv1.GetAssetRequest]) (*connect.Response[landingv1.AssetResponse], error) {
	asset, err := h.deps.Service.Get(int(req.Msg.Id))
	if err != nil {
		if errors.Is(err, internalassets.ErrAssetNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("assets.GetAsset: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.AssetResponse{Asset: assetToProto(asset)}), nil
}

func (h *connectHandler) DeleteAsset(_ context.Context, req *connect.Request[landingv1.DeleteAssetRequest]) (*connect.Response[landingv1.DeleteAssetResponse], error) {
	if err := h.deps.Service.Delete(int(req.Msg.Id)); err != nil {
		if errors.Is(err, internalassets.ErrAssetNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("assets.DeleteAsset: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.DeleteAssetResponse{Deleted: true}), nil
}

func assetToProto(a *internalassets.Asset) *landingv1.Asset {
	out := &landingv1.Asset{
		Id:               int64(a.ID),
		Filename:         a.Filename,
		OriginalFilename: a.OriginalFilename,
		MimeType:         a.MimeType,
		SizeBytes:        a.SizeBytes,
		StoragePath:      a.StoragePath,
		ThumbnailPath:    a.ThumbnailPath,
		AltText:          a.AltText,
		Category:         a.Category,
		UploadedBy:       a.UploadedBy,
		Url:              a.URL,
		Derivatives:      a.Derivatives,
	}
	if !a.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(a.CreatedAt)
	}
	return out
}
