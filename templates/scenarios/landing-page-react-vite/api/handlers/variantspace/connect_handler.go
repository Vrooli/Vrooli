// Package variantspace is the variant_space domain's API contribution: it
// serves the authored variant-space catalog verbatim as JSON bytes. The catalog
// is loaded and validated by internal/variantspace.
package variantspace

import (
	"context"
	"log"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalvariantspace "landing-page-react-vite-api/internal/variantspace"
)

// Deps wires the variant_space Connect handler.
type Deps struct {
	Space  *internalvariantspace.Space
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the VariantSpaceService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Space == nil {
		d.Space = internalvariantspace.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetVariantSpace(ctx context.Context, _ *connect.Request[landingv1.GetVariantSpaceRequest]) (*connect.Response[landingv1.GetVariantSpaceResponse], error) {
	return connect.NewResponse(&landingv1.GetVariantSpaceResponse{RawJson: h.deps.Space.JSONBytes()}), nil
}
