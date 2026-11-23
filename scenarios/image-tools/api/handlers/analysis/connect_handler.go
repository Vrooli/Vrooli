package analysis

import (
	"context"

	internalanalysis "image-tools/internal/analysis"

	"connectrpc.com/connect"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
)

// connectHandler implements AnalysisServiceHandler — the discovery surface. Op
// EXECUTION is the REST multipart analyze edge (rest.go); this Connect service
// only answers "what analysis operations exist".
type connectHandler struct {
	service *internalanalysis.Service
}

// NewConnectHandler builds the AnalysisService discovery handler. The service
// supplies each operation's live default model; nil lists the built-in table.
func NewConnectHandler(service *internalanalysis.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListAnalysisOperations(ctx context.Context, _ *connect.Request[analysisv1.ListAnalysisOperationsRequest]) (*connect.Response[analysisv1.ListAnalysisOperationsResponse], error) {
	resp := &analysisv1.ListAnalysisOperationsResponse{}
	for _, op := range h.service.Operations(ctx) {
		resp.Operations = append(resp.Operations, &analysisv1.AnalysisOperationInfo{
			Name:           op.Name,
			Summary:        op.Summary,
			ModelBacked:    op.ModelBacked,
			DefaultModelId: op.DefaultModelID,
		})
	}
	return connect.NewResponse(resp), nil
}
