package adoptions

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"react-component-library/internal/adoptions"

	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
)

// Deps wires the seams the Connect adoptions handler needs.
type Deps struct {
	Service adoptions.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.ListAdoptionsRequest]) (*connect.Response[adoptionsv1.ListAdoptionsResponse], error) {
	out, err := h.deps.Service.List(ctx, adoptions.ListQuery{
		ComponentID: req.Msg.ComponentId,
		Scenario:    req.Msg.Scenario,
		Limit:       int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("adoptions.ListAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.ListAdoptionsResponse{Adoptions: make([]*adoptionsv1.Adoption, 0, len(out))}
	for _, a := range out {
		resp.Adoptions = append(resp.Adoptions, domainToProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CreateAdoption(ctx context.Context, req *connect.Request[adoptionsv1.CreateAdoptionRequest]) (*connect.Response[adoptionsv1.CreateAdoptionResponse], error) {
	got, err := h.deps.Service.Create(ctx, adoptions.CreateInput{
		ComponentID:    req.Msg.ComponentId,
		Scenario:       req.Msg.Scenario,
		AdoptedPath:    req.Msg.AdoptedPath,
		AdoptedVersion: req.Msg.AdoptedVersion,
	})
	if err != nil {
		connectErr := adoptions.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("adoptions.CreateAdoption: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&adoptionsv1.CreateAdoptionResponse{Adoption: domainToProto(got)}), nil
}

func (h *connectHandler) DeleteAdoption(ctx context.Context, req *connect.Request[adoptionsv1.DeleteAdoptionRequest]) (*connect.Response[adoptionsv1.DeleteAdoptionResponse], error) {
	if err := h.deps.Service.Delete(ctx, req.Msg.Id); err != nil {
		connectErr := adoptions.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("adoptions.DeleteAdoption(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&adoptionsv1.DeleteAdoptionResponse{}), nil
}

func (h *connectHandler) RefreshAdoptions(ctx context.Context, req *connect.Request[adoptionsv1.RefreshAdoptionsRequest]) (*connect.Response[adoptionsv1.RefreshAdoptionsResponse], error) {
	rows, summary, err := h.deps.Service.Refresh(ctx, req.Msg.ComponentId)
	if err != nil {
		h.deps.Logger.Printf("adoptions.RefreshAdoptions: %v", err)
		return nil, adoptions.ToConnectError(err)
	}
	resp := &adoptionsv1.RefreshAdoptionsResponse{
		Adoptions: make([]*adoptionsv1.Adoption, 0, len(rows)),
		Current:   int32(summary.Current),
		Behind:    int32(summary.Behind),
		Modified:  int32(summary.Modified),
		Unknown:   int32(summary.Unknown),
	}
	for _, a := range rows {
		resp.Adoptions = append(resp.Adoptions, domainToProto(a))
	}
	return connect.NewResponse(resp), nil
}
