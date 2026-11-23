package ai

import (
	"context"
	"errors"
	"net/http"

	internalai "image-tools/internal/ai"
	"image-tools/internal/models"
	"image-tools/internal/verdicts"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// connectHandler serves discovery and durable admission for owned blob refs.
// Uploading new image bytes remains the multipart REST boundary.
type connectHandler struct {
	// previewer is the selection `ModelsService.SelectModel` uses; discovery asks
	// it for each operation's default so the two cannot disagree.
	previewer *models.Previewer
	verdicts  *verdicts.Service
	admission *Deps
}

// NewConnectHandler builds discovery and stored-input admission over the same
// policy and durable-job owner as multipart uploads. previewer and verdictSvc
// may be nil: discovery then lists the operations without defaults or verdicts.
func NewConnectHandler(previewer *models.Previewer, verdictSvc *verdicts.Service, admission *Deps) *connectHandler {
	return &connectHandler{previewer: previewer, verdicts: verdictSvc, admission: admission}
}

// ListAIOperations lists the AI operations with the model a submit would run
// for each right now and that model's verdict. The default is the previewer's
// selection (operator pins, enabled overrides, rights and hardware fit), not the
// seed's static default_for.
func (h *connectHandler) ListAIOperations(ctx context.Context, _ *connect.Request[aiv1.ListAIOperationsRequest]) (*connect.Response[aiv1.ListAIOperationsResponse], error) {
	resp := &aiv1.ListAIOperationsResponse{}
	var preview *models.PreviewContext
	var board *verdicts.Board
	unavailable := "model selection is not wired"
	if h.previewer != nil {
		var err error
		if preview, err = h.previewer.Load(ctx); err != nil {
			unavailable = err.Error()
			preview = nil
		}
	}
	if h.verdicts != nil {
		if b, err := h.verdicts.Load(ctx); err == nil {
			board = b
		}
	}
	for _, op := range internalai.List() {
		info := &aiv1.AIOperationInfo{
			Name:              op.Name,
			Category:          op.Family,
			Summary:           op.Summary,
			RequiresImage:     op.RequiresImage,
			RequiresMask:      op.RequiresMask,
			RequiresReference: op.RequiresReference,
			PromptDriven:      op.PromptDriven,
		}
		if preview == nil {
			info.DefaultUnavailable = unavailable
		} else if sel, err := preview.Select(ctx, models.PreviewRequest{Operation: op.Name}); err != nil {
			info.DefaultUnavailable = err.Error()
		} else {
			info.DefaultModelId = sel.Model.ID
			if board != nil {
				info.DefaultVerdict = verdicts.ToProto(board.For(sel.Model.ID, op.Name))
			} else {
				info.DefaultVerdict = verdicts.ToProto(verdicts.UntestedVerdict(op.Name))
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
