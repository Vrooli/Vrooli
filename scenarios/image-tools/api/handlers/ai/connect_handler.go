package ai

import (
	"context"

	internalai "image-tools/internal/ai"
	"image-tools/internal/models"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// connectHandler implements AIServiceHandler — the discovery surface. Op
// EXECUTION is the REST multipart submit edge (rest.go); this Connect service
// only answers "what model-backed operations exist + what inputs they need".
type connectHandler struct {
	registry *models.Registry
}

// NewConnectHandler builds the AIService discovery handler. registry supplies
// each op's seeded default model id for display.
func NewConnectHandler(registry *models.Registry) *connectHandler {
	return &connectHandler{registry: registry}
}

func (h *connectHandler) ListAIOperations(_ context.Context, _ *connect.Request[aiv1.ListAIOperationsRequest]) (*connect.Response[aiv1.ListAIOperationsResponse], error) {
	resp := &aiv1.ListAIOperationsResponse{}
	for _, op := range internalai.List() {
		info := &aiv1.AIOperationInfo{
			Name:          op.Name,
			Category:      string(op.Category),
			Summary:       op.Summary,
			RequiresImage: op.RequiresImage,
			RequiresMask:  op.RequiresMask,
			PromptDriven:  op.PromptDriven,
		}
		if h.registry != nil {
			if m, ok := h.registry.DefaultFor(op.Name); ok {
				info.DefaultModelId = m.ID
			}
		}
		resp.Operations = append(resp.Operations, info)
	}
	return connect.NewResponse(resp), nil
}
