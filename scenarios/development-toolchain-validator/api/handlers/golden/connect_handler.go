package golden

import (
	"context"
	"log"

	"development-toolchain-validator/internal/golden"

	"connectrpc.com/connect"

	goldenv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/golden"
)

// Deps wires the seams the Connect golden handler needs.
type Deps struct {
	Service golden.Service
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

func (h *connectHandler) ListGoldens(ctx context.Context, _ *connect.Request[goldenv1.ListGoldensRequest]) (*connect.Response[goldenv1.ListGoldensResponse], error) {
	results, err := h.deps.Service.List(ctx)
	if err != nil {
		h.deps.Logger.Printf("golden.ListGoldens: %v", err)
		return nil, golden.ToConnectError(err)
	}
	resp := &goldenv1.ListGoldensResponse{Goldens: make([]*goldenv1.Golden, 0, len(results))}
	for _, g := range results {
		resp.Goldens = append(resp.Goldens, domainToProto(g))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetGolden(ctx context.Context, req *connect.Request[goldenv1.GetGoldenRequest]) (*connect.Response[goldenv1.GetGoldenResponse], error) {
	g, err := h.deps.Service.Get(ctx, req.Msg.Slug)
	if err != nil {
		connectErr := golden.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("golden.GetGolden(%q): %v", req.Msg.Slug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&goldenv1.GetGoldenResponse{Golden: domainToProto(g)}), nil
}

func (h *connectHandler) RegisterGolden(ctx context.Context, req *connect.Request[goldenv1.RegisterGoldenRequest]) (*connect.Response[goldenv1.RegisterGoldenResponse], error) {
	g, err := h.deps.Service.Register(ctx, golden.RegisterInput{
		Slug:            req.Msg.Slug,
		TemplateID:      req.Msg.TemplateId,
		TemplateVersion: req.Msg.TemplateVersion,
		Path:            req.Msg.Path,
	})
	if err != nil {
		connectErr := golden.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("golden.RegisterGolden: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&goldenv1.RegisterGoldenResponse{Golden: domainToProto(g)}), nil
}

func (h *connectHandler) UpdateGolden(ctx context.Context, req *connect.Request[goldenv1.UpdateGoldenRequest]) (*connect.Response[goldenv1.UpdateGoldenResponse], error) {
	g, err := h.deps.Service.Update(ctx, golden.UpdateInput{
		Slug:            req.Msg.Slug,
		Path:            req.Msg.Path,
		TemplateVersion: req.Msg.TemplateVersion,
	})
	if err != nil {
		connectErr := golden.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("golden.UpdateGolden(%q): %v", req.Msg.Slug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&goldenv1.UpdateGoldenResponse{Golden: domainToProto(g)}), nil
}

func (h *connectHandler) DeleteGolden(ctx context.Context, req *connect.Request[goldenv1.DeleteGoldenRequest]) (*connect.Response[goldenv1.DeleteGoldenResponse], error) {
	if err := h.deps.Service.Delete(ctx, req.Msg.Slug); err != nil {
		connectErr := golden.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("golden.DeleteGolden(%q): %v", req.Msg.Slug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&goldenv1.DeleteGoldenResponse{}), nil
}

func (h *connectHandler) RegenerateGolden(ctx context.Context, req *connect.Request[goldenv1.RegenerateGoldenRequest]) (*connect.Response[goldenv1.RegenerateGoldenResponse], error) {
	g, err := h.deps.Service.Regenerate(ctx, req.Msg.Slug)
	if err != nil {
		connectErr := golden.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("golden.RegenerateGolden(%q): %v", req.Msg.Slug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&goldenv1.RegenerateGoldenResponse{Golden: domainToProto(g)}), nil
}
