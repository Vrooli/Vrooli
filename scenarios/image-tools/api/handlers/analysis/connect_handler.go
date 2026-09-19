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
type connectHandler struct{}

// NewConnectHandler builds the AnalysisService discovery handler.
func NewConnectHandler() *connectHandler { return &connectHandler{} }

func (h *connectHandler) ListAnalysisOperations(_ context.Context, _ *connect.Request[analysisv1.ListAnalysisOperationsRequest]) (*connect.Response[analysisv1.ListAnalysisOperationsResponse], error) {
	resp := &analysisv1.ListAnalysisOperationsResponse{}
	for _, op := range internalanalysis.List() {
		resp.Operations = append(resp.Operations, &analysisv1.AnalysisOperationInfo{
			Name:           op.Name,
			Summary:        op.Summary,
			ModelBacked:    op.ModelBacked,
			DefaultModelId: op.DefaultModelID,
		})
	}
	return connect.NewResponse(resp), nil
}
