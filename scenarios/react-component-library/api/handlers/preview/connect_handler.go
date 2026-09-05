package preview

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"react-component-library/internal/components"
	"react-component-library/internal/preview"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
)

// Deps wires the seams the Connect preview handler needs.
type Deps struct {
	Service preview.Service
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

func (h *connectHandler) GetPreviewBundle(ctx context.Context, req *connect.Request[previewv1.GetPreviewBundleRequest]) (*connect.Response[previewv1.GetPreviewBundleResponse], error) {
	bundle, err := h.deps.Service.GetBundle(ctx, req.Msg.Id)
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("preview.GetPreviewBundle(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&previewv1.GetPreviewBundleResponse{
		Js:         bundle.JS,
		SourcePath: bundle.SourcePath,
		Sha256:     bundle.SHA256,
		Warnings:   append([]string(nil), bundle.Warnings...),
	}), nil
}

// toConnectError translates preview-domain failures into Connect codes.
// Components-domain errors are forwarded to that package's mapper so
// the wire codes stay consistent across both handlers.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var bundleErr preview.ErrBundle
	if errors.As(err, &bundleErr) {
		return connect.NewError(connect.CodeInvalidArgument, bundleErr)
	}
	// Anything else is a components-domain pass-through (NotFound,
	// PathEscape, …) — delegate to the canonical mapper.
	return components.ToConnectError(err)
}
