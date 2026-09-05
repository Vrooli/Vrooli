package ops

import (
	"context"

	internalops "image-tools/internal/ops"

	"connectrpc.com/connect"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
)

// connectHandler implements OpsServiceHandler — the discovery surface. Op
// EXECUTION is the REST multipart edge (rest.go); this Connect service only
// answers "what operations and formats exist" so clients can build their UIs.
type connectHandler struct{}

// NewConnectHandler builds the OpsService discovery handler.
func NewConnectHandler() *connectHandler { return &connectHandler{} }

func (h *connectHandler) ListOperations(_ context.Context, _ *connect.Request[opsv1.ListOperationsRequest]) (*connect.Response[opsv1.ListOperationsResponse], error) {
	resp := &opsv1.ListOperationsResponse{
		DecodableFormats: internalops.DecodableFormats,
		EncodableFormats: internalops.EncodableFormats,
	}
	for _, op := range internalops.List() {
		resp.Operations = append(resp.Operations, &opsv1.OperationInfo{
			Name:     op.Name,
			Category: string(op.Category),
			Summary:  op.Summary,
		})
	}
	return connect.NewResponse(resp), nil
}
