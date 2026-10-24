package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/safety"
)

// admissionError retains the existing error vocabulary across both transports.
// Core admission is shared: stored-input execution cannot bypass the deployment
// gate, consent elevation, typed validation or durable request identity.
type admissionError struct {
	status        int
	code, message string
	cause         error
}

func (e *admissionError) Error() string { return e.message }
func rejectAdmission(status int, code, message string) error {
	return &admissionError{status: status, code: code, message: message}
}

func (h *Deps) preflightAdmission(ctx context.Context, op string, params *aiv1.AIParams, operationID string) (bool, error) {
	if _, ok := internalai.Get(op); !ok {
		return false, rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, fmt.Sprintf("unknown AI operation %q", op))
	}
	for _, validate := range []func(string, *aiv1.AIParams) error{validateSegmentation, validateSceneGeometry, validateImageToObject, validateBackgroundRepair, validateInpaint} {
		if err := validate(op, params); err != nil {
			return false, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		}
	}
	if operationID != "" {
		if _, err := uuid.Parse(operationID); err != nil {
			return false, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, "operation identity must be a UUID")
		}
	}
	if h.Gate == nil {
		return false, nil
	}
	if !h.Gate.AllowRate() {
		return false, rejectAdmission(http.StatusTooManyRequests, httpx.CodeRateLimited, "rate limit exceeded for this deployment tier; retry shortly")
	}
	decision := h.Gate.Evaluate(op, params.GetConsentAffirmed())
	if !decision.Allowed {
		return false, rejectAdmission(http.StatusForbidden, httpx.CodeForbidden, decision.Reason+" — "+decision.RecoveryHint)
	}
	if decision.RecordConsent {
		if err := h.Gate.RecordConsent(ctx, op, decision.Weight); err != nil {
			h.logf("ai.submit consent log: %v", err)
		}
	}
	return decision.ForceNSFWScan, nil
}

// admitStored receives only owned blob references, never filesystem paths or
// remote URLs. A retained admission is reconciled before current model selection.
func (h *Deps) admitStored(ctx context.Context, op string, params *aiv1.AIParams, inputKey, maskKey, operationID string, forceScan bool) (*aiv1.SubmitAIResponse, error) {
	meta, ok := internalai.Get(op)
	if !ok {
		return nil, rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, "unknown AI operation")
	}
	if (meta.RequiresImage && inputKey == "") || (meta.RequiresMask && maskKey == "") {
		return nil, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, "operation requires its input image and removal mask references")
	}
	for _, ref := range []string{inputKey, maskKey} {
		if ref == "" {
			continue
		}
		reader, _, err := h.Store.Get(ctx, ref)
		if err != nil {
			return nil, rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, "input or mask does not identify an Image Tools artifact")
		}
		_ = reader.Close()
	}
	fingerprint := ""
	if operationID != "" {
		var err error
		fingerprint, err = h.requestFingerprint(ctx, op, params, inputKey, maskKey)
		if err != nil {
			h.logf("ai.submit fingerprint: %v", err)
			return nil, rejectAdmission(http.StatusInternalServerError, httpx.CodeInternal, "failed to inspect retained input identity")
		}
		prior, err := h.Jobs.GetByClientOperation(ctx, operationID)
		if err == nil {
			if prior.RequestFingerprint != fingerprint {
				return nil, rejectAdmission(http.StatusConflict, httpx.CodeInvalidRequest, "operation identity was already used for different request input")
			}
			return savedAdmission(prior, nil)
		}
		if !errors.Is(err, internaljobs.ErrNotFound) {
			return nil, rejectAdmission(http.StatusServiceUnavailable, httpx.CodeInternal, "failed to inspect retained operation")
		}
	}
	plan, err := h.Engine.Plan(ctx, internalai.PlanRequest{Operation: op, ModelOverride: params.GetModelOverride(), AllowBYOK: allowBYOKForParams(params), QualityPolicy: qualityPolicyForParams(params), FallbackPolicy: fallbackPolicyForParams(params), Priority: priorityForParams(params), AllowReclaim: allowReclaimForParams(params), Role: params.GetOpenrouterRole(), Adapters: adapterRequests(params)})
	if err != nil {
		return nil, planAdmissionError(err)
	}
	if params.GetInpaint() != nil && h.modelBackend(plan.ModelID) != "diffusers" {
		return nil, rejectAdmission(http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, "contextual inpainting requires a native diffusers inpaint model; choose one explicitly")
	}
	if params.GetBackgroundRepair() != nil && plan.ModelID != "big-lama" {
		return nil, rejectAdmission(http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, "background repair controls require the qualified big-lama model; select it explicitly")
	}
	if h.Gate != nil && len(plan.Adapters) > 0 && plan.Weight != "" {
		decision := h.Gate.EvaluateWeight(safety.Weight(plan.Weight), op, params.GetConsentAffirmed())
		if !decision.Allowed {
			return nil, rejectAdmission(http.StatusForbidden, httpx.CodeForbidden, decision.Reason+" — "+decision.RecoveryHint)
		}
		if decision.RecordConsent {
			if err := h.Gate.RecordConsent(ctx, op, decision.Weight); err != nil {
				h.logf("ai.submit consent log (adapter-elevated): %v", err)
			}
		}
	}
	payload := internalai.Payload{Operation: op, InputKey: inputKey, MaskKey: maskKey, ModelID: plan.ModelID, Backend: h.modelBackend(plan.ModelID), Tier: plan.Tier, GPU: plan.GPUViable, AllowBYOK: allowBYOKForParams(params), CapacityPriority: priorityForParams(params), AllowReclaim: allowReclaimForParams(params), AutoScanNSFW: params.GetAutoScanNsfw() || forceScan, Variations: int(params.GetVariations()), Params: paramsMap(params), Adapters: plan.Adapters}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, rejectAdmission(http.StatusInternalServerError, httpx.CodeInternal, "failed to prepare job")
	}
	timeout := 0
	if op == "segment" {
		timeout = 180
	} else if operationID != "" {
		timeout = 30 * 60
	}
	job, err := h.Jobs.Submit(ctx, internaljobs.Spec{Operation: op, Lane: internalai.LaneForTier(plan.Tier), Payload: raw, ClientOperationID: operationID, RequestFingerprint: fingerprint, ExecutionTimeoutSeconds: timeout, EstimatedSeconds: plan.EstimatedSeconds})
	if err != nil {
		h.logf("ai.submit enqueue: %v", err)
		if errors.Is(err, internaljobs.ErrIdempotencyConflict) {
			return nil, rejectAdmission(http.StatusConflict, httpx.CodeInvalidRequest, "operation identity was already used for different request input")
		}
		return nil, rejectAdmission(http.StatusServiceUnavailable, httpx.CodeInternal, "failed to enqueue job")
	}
	return savedAdmission(job, plan.Warnings)
}

func savedAdmission(job internaljobs.Job, warnings []string) (*aiv1.SubmitAIResponse, error) {
	var payload internalai.Payload
	if err := decodeSavedAIPayload(job, &payload); err != nil {
		return nil, rejectAdmission(http.StatusInternalServerError, httpx.CodeInternal, "retained operation payload is invalid")
	}
	return &aiv1.SubmitAIResponse{JobId: job.ID, EstimatedSeconds: int32(job.EstimatedSeconds), ModelId: payload.ModelID, Tier: payload.Tier, Warnings: warnings}, nil
}

func planAdmissionError(err error) error {
	status, code := http.StatusUnprocessableEntity, httpx.CodeInvalidRequest
	switch {
	case errors.Is(err, internalai.ErrModelNotInstalled):
		status = http.StatusConflict
	case errors.Is(err, backends.ErrNoneAvailable), errors.Is(err, backends.ErrNoProvider):
		status, code = http.StatusServiceUnavailable, httpx.CodeInternal
	case errors.Is(err, models.ErrUnknownOperation):
		status, code = http.StatusNotFound, httpx.CodeNotFound
	}
	return &admissionError{status: status, code: code, message: err.Error(), cause: err}
}

func (e *admissionError) Unwrap() error { return e.cause }

func (h *Deps) submitStored(ctx context.Context, request *aiv1.SubmitStoredAIRequest) (*aiv1.SubmitAIResponse, error) {
	params := request.GetParams()
	if params == nil {
		params = &aiv1.AIParams{}
	}
	operationID := strings.TrimSpace(request.GetOperationId())
	if operationID == "" {
		return nil, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, "stored-input submission requires a stable UUID operation_id")
	}
	forceScan, err := h.preflightAdmission(ctx, request.GetOperation(), params, operationID)
	if err != nil {
		return nil, err
	}
	return h.admitStored(ctx, request.GetOperation(), params, strings.TrimSpace(request.GetInputRef()), strings.TrimSpace(request.GetMaskRef()), operationID, forceScan)
}
