package selection

import (
	"context"

	internalselection "image-tools/internal/selection"

	"connectrpc.com/connect"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

// connectHandler implements SelectionServiceHandler — the discovery + pure
// contextual-edit compiler surface. Segmentation EXECUTION is the REST multipart
// edge (rest.go); this Connect service answers "what region classes exist" and
// "what edits make sense for this class" without touching pixels.
type connectHandler struct{}

// NewConnectHandler builds the SelectionService discovery handler.
func NewConnectHandler() *connectHandler { return &connectHandler{} }

func (h *connectHandler) ListRegionClasses(_ context.Context, _ *connect.Request[selectionv1.ListRegionClassesRequest]) (*connect.Response[selectionv1.ListRegionClassesResponse], error) {
	resp := &selectionv1.ListRegionClassesResponse{}
	for _, c := range internalselection.ListClasses() {
		resp.Classes = append(resp.Classes, classToProto(c))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) SuggestEdits(_ context.Context, req *connect.Request[selectionv1.SuggestEditsRequest]) (*connect.Response[selectionv1.SuggestEditsResponse], error) {
	resolved, edits := internalselection.Suggest(req.Msg.GetRegionClass())
	return connect.NewResponse(&selectionv1.SuggestEditsResponse{
		RegionClass: resolved,
		Edits:       editsToProto(edits),
	}), nil
}
