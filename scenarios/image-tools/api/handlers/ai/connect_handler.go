package ai

import (
	"context"
	"errors"
	"net/http"

	internalai "image-tools/internal/ai"
	"image-tools/internal/models"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// connectHandler serves discovery and durable admission for owned blob refs.
// Uploading new image bytes remains the multipart REST boundary.
type connectHandler struct {
	registry  *models.Registry
	admission *Deps
}

// NewConnectHandler builds discovery and stored-input admission over the same
// policy and durable-job owner as multipart uploads.
func NewConnectHandler(registry *models.Registry, admission *Deps) *connectHandler {
	return &connectHandler{registry: registry, admission: admission}
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

func (h *connectHandler) SubmitStoredAI(ctx context.Context, req *connect.Request[aiv1.SubmitStoredAIRequest]) (*connect.Response[aiv1.SubmitAIResponse], error) {
	if h.admission == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("AI admission is unavailable"))
	}
	result, err := h.admission.submitStored(ctx, req.Msg)
	if err != nil {
		code := connect.CodeInternal
		var rejection *admissionError
		if errors.As(err, &rejection) {
			switch rejection.status {
			case http.StatusBadRequest:
				code = connect.CodeInvalidArgument
			case http.StatusNotFound:
				code = connect.CodeNotFound
			case http.StatusForbidden:
				code = connect.CodePermissionDenied
			case http.StatusTooManyRequests:
				code = connect.CodeResourceExhausted
			case http.StatusConflict:
				code = connect.CodeAlreadyExists
			case http.StatusUnprocessableEntity:
				code = connect.CodeFailedPrecondition
			case http.StatusServiceUnavailable:
				code = connect.CodeUnavailable
			}
		}
		if errors.Is(err, internalai.ErrModelNotInstalled) {
			code = connect.CodeFailedPrecondition
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(result), nil
}
