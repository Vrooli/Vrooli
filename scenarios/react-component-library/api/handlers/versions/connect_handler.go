package versions

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"react-component-library/internal/versions"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
)

// Deps wires the seams the Connect versions handler needs.
type Deps struct {
	Service versions.Service
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

func (h *connectHandler) ListVersions(ctx context.Context, req *connect.Request[versionsv1.ListVersionsRequest]) (*connect.Response[versionsv1.ListVersionsResponse], error) {
	out, err := h.deps.Service.List(ctx, versions.ListQuery{
		ComponentID: req.Msg.ComponentId,
		Limit:       int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("versions.ListVersions: %v", err)
		return nil, mapErr(err)
	}
	resp := &versionsv1.ListVersionsResponse{Versions: make([]*versionsv1.Version, 0, len(out))}
	for _, v := range out {
		resp.Versions = append(resp.Versions, versionToProto(v, false))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetVersion(ctx context.Context, req *connect.Request[versionsv1.GetVersionRequest]) (*connect.Response[versionsv1.GetVersionResponse], error) {
	v, err := h.deps.Service.Get(ctx, req.Msg.ComponentId, req.Msg.Version)
	if err != nil {
		connectErr := mapErr(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("versions.GetVersion(%q,%q): %v", req.Msg.ComponentId, req.Msg.Version, err)
		}
		return nil, connectErr
	}
	resp := &versionsv1.GetVersionResponse{Version: versionToProto(v, false)}
	if req.Msg.IncludeContent {
		resp.Content = v.Content
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DiffVersions(ctx context.Context, req *connect.Request[versionsv1.DiffVersionsRequest]) (*connect.Response[versionsv1.DiffVersionsResponse], error) {
	result, err := h.deps.Service.Diff(ctx, versions.DiffInput{
		ComponentID: req.Msg.ComponentId,
		From:        req.Msg.From,
		To:          req.Msg.To,
	})
	if err != nil {
		connectErr := mapErr(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("versions.DiffVersions: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(diffToProto(result)), nil
}

func mapErr(err error) *connect.Error {
	if mapped := versions.ToConnectError(err); mapped != nil {
		return mapped
	}
	return connect.NewError(connect.CodeInternal, err)
}
