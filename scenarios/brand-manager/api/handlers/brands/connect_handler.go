package brands

import (
	"context"
	"log"

	"brand-manager/internal/brands"

	"connectrpc.com/connect"

	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
)

// Deps wires the seams the Connect brands handler needs.
type Deps struct {
	Service brands.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC brands handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListBrands(ctx context.Context, req *connect.Request[brandsv1.ListBrandsRequest]) (*connect.Response[brandsv1.ListBrandsResponse], error) {
	results, err := h.deps.Service.List(ctx, brands.ListFilter{
		NameContains: req.Msg.GetNameContains(),
		Limit:        int(req.Msg.GetLimit()),
		Offset:       int(req.Msg.GetOffset()),
	})
	if err != nil {
		h.deps.Logger.Printf("brands.ListBrands: %v", err)
		return nil, brands.ToConnectError(err)
	}
	resp := &brandsv1.ListBrandsResponse{Brands: make([]*brandsv1.Brand, 0, len(results))}
	for _, b := range results {
		resp.Brands = append(resp.Brands, domainToProto(b))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CreateBrand(ctx context.Context, req *connect.Request[brandsv1.CreateBrandRequest]) (*connect.Response[brandsv1.CreateBrandResponse], error) {
	created, err := h.deps.Service.Create(ctx, brands.CreateInput{
		Name:        req.Msg.GetName(),
		Description: req.Msg.GetDescription(),
		Notes:       req.Msg.GetNotes(),
		Identity:    identityFromProto(req.Msg.GetIdentity()),
		Colors:      colorsFromProto(req.Msg.GetColors()),
		Typography:  typographyFromProto(req.Msg.GetTypography()),
		Voice:       voiceFromProto(req.Msg.GetVoice()),
	})
	if err != nil {
		return nil, h.translate("brands.CreateBrand", err)
	}
	return connect.NewResponse(&brandsv1.CreateBrandResponse{Brand: domainToProto(created)}), nil
}

func (h *connectHandler) GetBrand(ctx context.Context, req *connect.Request[brandsv1.GetBrandRequest]) (*connect.Response[brandsv1.GetBrandResponse], error) {
	got, err := h.deps.Service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.translate("brands.GetBrand", err)
	}
	return connect.NewResponse(&brandsv1.GetBrandResponse{Brand: domainToProto(got)}), nil
}

func (h *connectHandler) UpdateBrand(ctx context.Context, req *connect.Request[brandsv1.UpdateBrandRequest]) (*connect.Response[brandsv1.UpdateBrandResponse], error) {
	updated, err := h.deps.Service.Update(ctx, brands.UpdateInput{
		ID:              req.Msg.GetId(),
		Name:            req.Msg.GetName(),
		Description:     req.Msg.GetDescription(),
		Notes:           req.Msg.GetNotes(),
		Identity:        identityFromProto(req.Msg.GetIdentity()),
		Colors:          colorsFromProto(req.Msg.GetColors()),
		Typography:      typographyFromProto(req.Msg.GetTypography()),
		Voice:           voiceFromProto(req.Msg.GetVoice()),
		ExpectedVersion: int(req.Msg.GetExpectedVersion()),
	})
	if err != nil {
		return nil, h.translate("brands.UpdateBrand", err)
	}
	return connect.NewResponse(&brandsv1.UpdateBrandResponse{Brand: domainToProto(updated)}), nil
}

func (h *connectHandler) DeleteBrand(ctx context.Context, req *connect.Request[brandsv1.DeleteBrandRequest]) (*connect.Response[brandsv1.DeleteBrandResponse], error) {
	if err := h.deps.Service.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, h.translate("brands.DeleteBrand", err)
	}
	return connect.NewResponse(&brandsv1.DeleteBrandResponse{}), nil
}

func (h *connectHandler) ListBrandVersions(ctx context.Context, req *connect.Request[brandsv1.ListBrandVersionsRequest]) (*connect.Response[brandsv1.ListBrandVersionsResponse], error) {
	versions, err := h.deps.Service.ListVersions(ctx, req.Msg.GetBrandId())
	if err != nil {
		h.deps.Logger.Printf("brands.ListBrandVersions: %v", err)
		return nil, brands.ToConnectError(err)
	}
	resp := &brandsv1.ListBrandVersionsResponse{Versions: make([]*brandsv1.BrandVersion, 0, len(versions))}
	for _, v := range versions {
		resp.Versions = append(resp.Versions, versionToProto(v))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetTokens(ctx context.Context, req *connect.Request[brandsv1.GetTokensRequest]) (*connect.Response[brandsv1.GetTokensResponse], error) {
	brand, err := h.deps.Service.Get(ctx, req.Msg.GetBrandId())
	if err != nil {
		return nil, h.translate("brands.GetTokens", err)
	}
	colors := brand.Colors
	values := []*brandsv1.Token{{Name: "$brand.primary", Value: colors.Primary}, {Name: "$brand.secondary", Value: colors.Secondary}, {Name: "$brand.accent", Value: colors.Accent}, {Name: "$brand.background", Value: colors.Background}, {Name: "$brand.surface", Value: colors.Surface}, {Name: "$brand.text", Value: colors.Text}, {Name: "$brand.error", Value: colors.Error}}
	return connect.NewResponse(&brandsv1.GetTokensResponse{Tokens: values}), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault 4xx-equivalent codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := brands.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
