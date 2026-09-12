package generation

import (
	"context"
	"log"

	"brand-manager/internal/generation"

	"connectrpc.com/connect"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
)

// Deps wires the seams the Connect generation handler needs.
type Deps struct {
	Service generation.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC generation handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetProviderStatus(ctx context.Context, _ *connect.Request[generationv1.GetProviderStatusRequest]) (*connect.Response[generationv1.GetProviderStatusResponse], error) {
	available, statuses := h.deps.Service.ProviderStatus(ctx)
	return connect.NewResponse(&generationv1.GetProviderStatusResponse{
		Available: available,
		Providers: statusesToProto(statuses),
	}), nil
}

func (h *connectHandler) GetImageBackendStatus(ctx context.Context, _ *connect.Request[generationv1.GetImageBackendStatusRequest]) (*connect.Response[generationv1.GetImageBackendStatusResponse], error) {
	status := h.deps.Service.ImageBackendStatus(ctx)
	return connect.NewResponse(imageBackendStatusToProto(status)), nil
}

func (h *connectHandler) GenerateBrandElements(ctx context.Context, req *connect.Request[generationv1.GenerateBrandElementsRequest]) (*connect.Response[generationv1.GenerateBrandElementsResponse], error) {
	result, err := h.deps.Service.GenerateElements(ctx, req.Msg.GetBrandId(), req.Msg.GetElements(), req.Msg.GetModel())
	if err != nil {
		return nil, h.translate("generation.GenerateBrandElements", err)
	}
	return connect.NewResponse(elementsResultToProto(result)), nil
}

func (h *connectHandler) GenerateBrandImage(ctx context.Context, req *connect.Request[generationv1.GenerateBrandImageRequest]) (*connect.Response[generationv1.BrandImageAsset], error) {
	result, err := h.deps.Service.GenerateImage(ctx, generation.GenerateImageInput{
		BrandID:        req.Msg.GetBrandId(),
		Type:           req.Msg.GetType(),
		ModelOverride:  req.Msg.GetModelOverride(),
		AllowBYOK:      req.Msg.GetAllowByok(),
		QualityPolicy:  req.Msg.GetQualityPolicy(),
		FallbackPolicy: req.Msg.GetFallbackPolicy(),
		Priority:       req.Msg.GetPriority(),
		AllowReclaim:   req.Msg.AllowReclaim,
		Seed:           req.Msg.GetSeed(),
		SetCanonical:   req.Msg.GetSetCanonical(),
	})
	if err != nil {
		return nil, h.translate("generation.GenerateBrandImage", err)
	}
	return connect.NewResponse(imageResultToProto(result)), nil
}

func (h *connectHandler) EditBrandImage(ctx context.Context, req *connect.Request[generationv1.EditBrandImageRequest]) (*connect.Response[generationv1.BrandImageAsset], error) {
	result, err := h.deps.Service.EditImage(ctx, generation.EditImageInput{
		BrandID:        req.Msg.GetBrandId(),
		SourceAssetID:  req.Msg.GetSourceAssetId(),
		Instruction:    req.Msg.GetInstruction(),
		ModelOverride:  req.Msg.GetModelOverride(),
		AllowBYOK:      req.Msg.GetAllowByok(),
		QualityPolicy:  req.Msg.GetQualityPolicy(),
		FallbackPolicy: req.Msg.GetFallbackPolicy(),
		Priority:       req.Msg.GetPriority(),
		AllowReclaim:   req.Msg.AllowReclaim,
		Seed:           req.Msg.GetSeed(),
		SetCanonical:   req.Msg.GetSetCanonical(),
	})
	if err != nil {
		return nil, h.translate("generation.EditBrandImage", err)
	}
	return connect.NewResponse(imageResultToProto(result)), nil
}

func (h *connectHandler) RemoveBrandImageBackground(ctx context.Context, req *connect.Request[generationv1.RemoveBrandImageBackgroundRequest]) (*connect.Response[generationv1.BrandImageAsset], error) {
	result, err := h.deps.Service.RemoveBackground(ctx, generation.RemoveBackgroundInput{
		BrandID:       req.Msg.GetBrandId(),
		SourceAssetID: req.Msg.GetSourceAssetId(),
		ModelOverride: req.Msg.GetModelOverride(),
		AllowBYOK:     req.Msg.GetAllowByok(),
		SetCanonical:  req.Msg.GetSetCanonical(),
	})
	if err != nil {
		return nil, h.translate("generation.RemoveBrandImageBackground", err)
	}
	return connect.NewResponse(imageResultToProto(result)), nil
}

func (h *connectHandler) DeriveBrandIcons(ctx context.Context, req *connect.Request[generationv1.DeriveBrandIconsRequest]) (*connect.Response[generationv1.DeriveBrandIconsResponse], error) {
	icons, warnings, err := h.deps.Service.DeriveIcons(ctx, generation.DeriveIconsInput{
		BrandID:           req.Msg.GetBrandId(),
		SourceAssetID:     req.Msg.GetSourceAssetId(),
		IncludeMaskable:   req.Msg.GetIncludeMaskable(),
		IncludeAppleTouch: req.Msg.GetIncludeAppleTouch(),
		IncludeFavicon:    req.Msg.GetIncludeFavicon(),
	})
	if err != nil {
		return nil, h.translate("generation.DeriveBrandIcons", err)
	}
	return connect.NewResponse(deriveIconsResultToProto(icons, warnings)), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := generation.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
