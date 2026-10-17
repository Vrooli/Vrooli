package diff

import (
	"context"

	internaldiff "image-tools/internal/diff"

	"connectrpc.com/connect"

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
)

// connectHandler implements DiffServiceHandler — the discovery surface.
// Comparison EXECUTION is the REST multipart edge (rest.go); this Connect
// service answers "what comparison modes exist" without touching pixels.
type connectHandler struct{}

// NewConnectHandler builds the DiffService discovery handler.
func NewConnectHandler() *connectHandler { return &connectHandler{} }

func (h *connectHandler) ListDiffModes(_ context.Context, _ *connect.Request[diffv1.ListDiffModesRequest]) (*connect.Response[diffv1.ListDiffModesResponse], error) {
	resp := &diffv1.ListDiffModesResponse{}
	for _, m := range internaldiff.Modes() {
		resp.Modes = append(resp.Modes, &diffv1.DiffModeInfo{Name: m.Name, Summary: m.Summary})
	}
	return connect.NewResponse(resp), nil
}
