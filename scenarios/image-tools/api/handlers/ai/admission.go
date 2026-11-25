package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	"image-tools/internal/adapters"
	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	internalquarantine "image-tools/internal/quarantine"
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

func (h *Deps) preflightAdmission(ctx context.Context, op string, params *aiv1.AIParams, operationID string) (safety.Decision, error) {
	if _, ok := internalai.Get(op); !ok {
		return safety.Decision{}, rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, fmt.Sprintf("unknown AI operation %q", op))
	}
	for _, validate := range []func(string, *aiv1.AIParams) error{validateOpParams, validateSegmentationSemantics} {
		if err := validate(op, params); err != nil {
			return safety.Decision{}, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		}
	}
	if operationID != "" {
		if _, err := uuid.Parse(operationID); err != nil {
			return safety.Decision{}, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, "operation identity must be a UUID")
		}
	}
	if h.Gate == nil {
		return safety.Decision{Allowed: true}, nil
	}
	if !h.Gate.AllowRate() {
		return safety.Decision{}, rejectAdmission(http.StatusTooManyRequests, httpx.CodeRateLimited, "rate limit exceeded for this deployment tier; retry shortly")
	}
	// The consent rules that depend on where the work runs are applied after the
	// route is planned (admitStored); this first pass carries the tier-wide
	// refusals (minor-safety, a public deployment's tier rule) that need no plan.
	decision := h.Gate.Evaluate(op, params.GetConsentAffirmed())
	if !decision.Allowed {
		return safety.Decision{}, decisionAdmissionError(decision)
	}
	return decision, nil
}

// decisionAdmissionError maps a gate decision to the wire status. A technical
// reviewer failure (or a required reviewer that is not configured) is 503; a
// genuine policy refusal is 403. The two must not look identical to a caller.
func decisionAdmissionError(decision safety.Decision) error {
	status, code := http.StatusForbidden, httpx.CodeForbidden
	if decision.Unavailable {
		status, code = http.StatusServiceUnavailable, httpx.CodeUnavailable
	}
	return rejectAdmission(status, code, decision.Reason+" — "+decision.RecoveryHint)
}

// admitStored receives only owned blob references, never filesystem paths or
// remote URLs. A retained admission is reconciled before current model selection.
func (h *Deps) admitStored(ctx context.Context, op string, params *aiv1.AIParams, inputKey, maskKey, referenceKey, operationID string, gate safety.Decision) (*aiv1.SubmitAIResponse, error) {
	meta, ok := internalai.Get(op)
	if !ok {
		return nil, rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, "unknown AI operation")
	}
	if (meta.RequiresImage && inputKey == "") || (meta.RequiresMask && maskKey == "") || (meta.RequiresReference && referenceKey == "") {
		return nil, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, "operation requires its input image, mask and reference image references")
	}
	for _, ref := range []string{inputKey, maskKey, referenceKey} {
		if ref == "" {
			continue
		}
		if internalquarantine.IsKey(ref) {
			return nil, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, "a quarantined refusal cannot be used as AI input")
		}
		reader, _, err := h.Store.Get(ctx, ref)
		if err != nil {
			return nil, rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, "input or mask does not identify an Image Tools artifact")
		}
		_ = reader.Close()
	}
	if err := h.reviewMinorInput(ctx, op, gate, inputKey, referenceKey); err != nil {
		return nil, err
	}
	// A preset is admitted as the operation it runs: its plan, consent, capacity and
	// output review are that operation's. The fingerprint below is that request's.
	op, params = presetRequest(op, params, inputKey, referenceKey)
	fingerprint := ""
	if operationID != "" {
		var err error
		fingerprint, err = h.requestFingerprint(ctx, op, params, inputKey, maskKey, referenceKey)
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
	faces := h.newInputFaces(inputKey, referenceKey)
	requestParams := paramsMap(params)
	modelOverride, err := h.extentDefaults(op, params, requestParams)
	if err != nil {
		return nil, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
	}
	plan, err := h.Engine.Plan(ctx, internalai.PlanRequest{Operation: op, ModelOverride: modelOverride, AllowBYOK: allowBYOKForParams(params), QualityPolicy: qualityPolicyForParams(params), FallbackPolicy: fallbackPolicyForParams(params), Priority: priorityForParams(params), AllowReclaim: allowReclaimForParams(params), RequireGPU: params.GetRequireGpu(), ForceCPU: params.GetForceCpu(), Role: params.GetOpenrouterRole(), Adapters: adapterRequests(params), AutoAcquire: autoAcquireForParams(params), Prompt: params.GetPrompt(), Variations: int(params.GetVariations()), EditStyle: params.GetEditStyle(), MaxWaitSeconds: maxWaitForParams(params), MaxCostUSD: maxCostForParams(params), ConsentAffirmed: params.GetConsentAffirmed(), InputFaces: faces.Func()})
	if err != nil {
		return nil, planAdmissionError(err)
	}
	if err := bindPosePreprocessorInput(plan.Adapters, inputKey); err != nil {
		return nil, rejectAdmission(http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
	}
	if hasInpaintControls(params) && h.modelBackend(plan.ModelID) != "diffusers" {
		return nil, rejectAdmission(http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, "contextual inpainting requires a native diffusers inpaint model; choose one explicitly")
	}
	if hasBackgroundRepair(params) && plan.ModelID != "big-lama" {
		return nil, rejectAdmission(http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, "background repair controls require the qualified big-lama model; select it explicitly")
	}
	consentBasis := ""
	if h.Gate != nil {
		// The route is known now: the consent rules apply to what will actually
		// run. The weight is the plan's (raised by any adapter stack, decision C4)
		// and never below the operation's own.
		weight := safety.MaxWeight(safety.OpWeight(op), safety.Weight(plan.Weight))
		decision := h.Gate.EvaluateRouteInput(ctx, weight, op, plan.Tier, params.GetConsentAffirmed(), faces.Func())
		if !decision.Allowed {
			return nil, decisionAdmissionError(decision)
		}
		consentBasis = decision.ConsentBasis
		if decision.RecordConsent {
			if err := h.Gate.RecordConsent(ctx, op, decision.Weight, plan.Tier+":"+plan.ModelID); err != nil {
				h.logf("ai.submit consent log: %v", err)
			}
		}
	}
	// A look that found nobody is remembered on the job, so a re-route chosen
	// after it was queued does not have to ask again.
	consentWaived := faces.Clear()
	payload := internalai.Payload{Operation: op, InputKey: inputKey, MaskKey: maskKey, ReferenceKey: referenceKey, ConsentAffirmed: params.GetConsentAffirmed(), ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, GPU: plan.GPUViable, RequireGPU: params.GetRequireGpu(), AllowBYOK: plan.AllowBYOK, CapacityPriority: priorityForParams(params), AllowReclaim: allowReclaimForParams(params), AutoScanNSFW: params.GetAutoScanNsfw() || gate.ForceNSFWScan, ConsentWaived: consentWaived, ConsentBasis: consentBasis, Variations: int(params.GetVariations()), Params: paramsWithPlan(requestParams, plan.Params), Adapters: plan.Adapters, AutoAcquire: plan.AutoAcquire, Route: plan.Route, PreserveIdentity: params.PreserveIdentity}
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
	job, err := h.Jobs.Submit(ctx, internaljobs.Spec{Operation: op, Lane: internalai.LaneForTier(plan.Tier), Priority: internaljobs.Priority(payload.CapacityPriority), Payload: raw, ClientOperationID: operationID, RequestFingerprint: fingerprint, ExecutionTimeoutSeconds: timeout, EstimatedSeconds: plan.EstimatedSeconds})
	if err != nil {
		h.logf("ai.submit enqueue: %v", err)
		if errors.Is(err, internaljobs.ErrIdempotencyConflict) {
			return nil, rejectAdmission(http.StatusConflict, httpx.CodeInvalidRequest, "operation identity was already used for different request input")
		}
		return nil, rejectAdmission(http.StatusServiceUnavailable, httpx.CodeInternal, "failed to enqueue job")
	}
	return savedAdmission(job, plan.Warnings)
}

// extentDefaults applies what a swap extent pins to the request: the model a
// stage needs (a general editor for the whole head) when the caller named none,
// and the instruction that stage is given when the caller wrote none. It fails
// for an extent the operation does not declare, before any job exists.
func (h *Deps) extentDefaults(op string, params *aiv1.AIParams, requestParams map[string]string) (string, error) {
	modelOverride := params.GetModelOverride()
	model, err := internalai.ExtentModel(op, requestParams)
	if err != nil {
		return "", err
	}
	if modelOverride == "" {
		modelOverride = model
	}
	if requestParams["prompt"] == "" {
		prompt, err := internalai.ExtentPrompt(op, requestParams)
		if err != nil {
			return "", err
		}
		if prompt != "" && modelOverride == model {
			requestParams["prompt"] = prompt
		}
	}
	return modelOverride, nil
}

// bindPosePreprocessorInput links a multipart source image to a ControlNet pose
// preprocessor when the caller requested it without a separately stored image
// key. This lets the UI and CLI submit one durable job containing both the raw
// pose source and the generated image request. A supplied conditioning key is
// preserved, and a missing source fails before a job is queued.
func bindPosePreprocessorInput(resolved []adapters.ResolvedAdapter, inputKey string) error {
	for i := range resolved {
		a := &resolved[i]
		if a.Kind != adapters.KindControlNet || a.Preprocessor != adapters.PreprocessorPose || a.ConditioningImageKey != "" {
			continue
		}
		if inputKey == "" {
			return fmt.Errorf("pose-conditioned generation requires a source image")
		}
		a.ConditioningImageKey = inputKey
	}
	return nil
}

// reviewMinorInput runs the deployment's reviewer on the source image before any
// model is admitted. It is skipped entirely on a local deployment and for
// operations whose policy does not require it.
func (h *Deps) reviewMinorInput(ctx context.Context, op string, gate safety.Decision, refs ...string) error {
	if h.Gate == nil || !gate.RequireMinorReview {
		return nil
	}
	const maxReviewBytes = 64 << 20
	images := make([][]byte, 0, len(refs))
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		reader, _, err := h.Store.Get(ctx, ref)
		if err != nil {
			return rejectAdmission(http.StatusNotFound, httpx.CodeNotFound, "input does not identify an Image Tools artifact")
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, maxReviewBytes+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			return rejectAdmission(http.StatusInternalServerError, httpx.CodeInternal, "failed to read input for safety review")
		}
		if len(data) > maxReviewBytes {
			return rejectAdmission(http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "input is too large for the required safety review")
		}
		images = append(images, data)
	}
	decision := h.Gate.ReviewMinor(ctx, op, images)
	if !decision.Allowed {
		return decisionAdmissionError(decision)
	}
	if decision.Shadow {
		h.logf("ai.submit shadow safety refusal recorded: %s", decision.Reason)
	}
	return nil
}

func savedAdmission(job internaljobs.Job, warnings []string) (*aiv1.SubmitAIResponse, error) {
	var payload internalai.Payload
	if err := decodeSavedAIPayload(job, &payload); err != nil {
		return nil, rejectAdmission(http.StatusInternalServerError, httpx.CodeInternal, "retained operation payload is invalid")
	}
	return &aiv1.SubmitAIResponse{JobId: job.ID, EstimatedSeconds: int32(job.EstimatedSeconds), ModelId: payload.ModelID, Tier: payload.Tier, Warnings: warnings}, nil
}

// autoAcquireForParams returns the caller's explicit auto-acquire choice, or
// nil to use the service default.
func autoAcquireForParams(params *aiv1.AIParams) *bool {
	if params == nil || params.AutoAcquire == nil {
		return nil
	}
	v := params.GetAutoAcquire()
	return &v
}

func planAdmissionError(err error) error {
	status, code := http.StatusUnprocessableEntity, httpx.CodeInvalidRequest
	switch {
	case errors.Is(err, internalai.ErrModelNotInstalled):
		status = http.StatusConflict
	case errors.Is(err, internalai.ErrSpendCap):
		status = http.StatusPaymentRequired
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
	gate, err := h.preflightAdmission(ctx, request.GetOperation(), params, operationID)
	if err != nil {
		return nil, err
	}
	return h.admitStored(ctx, request.GetOperation(), params, strings.TrimSpace(request.GetInputRef()), strings.TrimSpace(request.GetMaskRef()), strings.TrimSpace(request.GetReferenceRef()), operationID, gate)
}
