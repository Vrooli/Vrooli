package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/safety"
	"image-tools/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	"google.golang.org/protobuf/encoding/protojson"
)

// BlobStore is the storage surface the submit edge needs (satisfied by
// *internal/storage.Store).
type BlobStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Put(ctx context.Context, key string, r io.Reader, mime string) error
}

// Submitter submits a durable job (satisfied by *internal/jobs.Manager).
type Submitter interface {
	Submit(ctx context.Context, spec internaljobs.Spec) (internaljobs.Job, error)
}

// Gate is the Responsible-Use deployment gate the submit edge enforces
// (satisfied by *internal/safety.Gate). Nil disables the gate (the local
// behaviour: unrestricted).
type Gate interface {
	Evaluate(op string, consentAffirmed bool) safety.Decision
	AllowRate() bool
	RecordConsent(ctx context.Context, op string, weight safety.Weight) error
}

// Deps wires the AI submit edge's seams.
type Deps struct {
	Engine *ai.Engine
	Store  BlobStore
	Jobs   Submitter
	Guard  storage.Guard
	Gate   Gate
	Logger *log.Logger
}

const maxMultipartMemory = 32 << 20

// submitHandler is POST /api/v1/ai/{operation}: the multipart submit edge for
// the model-backed generation/enhancement ops. The optional `file`/`mask` parts
// carry image bytes; the `params` part carries AIParams as protojson. The op
// runs ASYNC: this edge pre-flights selection (so it can reject an unrunnable
// request and surface model/tier/ETA), stores the inputs, submits a durable job,
// and returns SubmitAIResponse immediately. Callers wait via JobsService.
func (h *Deps) submitHandler(w http.ResponseWriter, r *http.Request) {
	op := mux.Vars(r)["operation"]
	meta, ok := ai.Get(op)
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, fmt.Sprintf("unknown AI operation %q", op))
		return
	}

	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "malformed multipart form")
		return
	}

	params, ok := h.parseParams(w, r.FormValue("params"))
	if !ok {
		return
	}

	// Responsible-Use deployment gate (IMG-P1-015). On the public tier this
	// enforces the abuse throttle, requires a consent affirmation for
	// identity-altering ops, forces the NSFW output scan, and audits consent.
	// On the local tier it is a no-op (personal use is unrestricted).
	forceScan := false
	if h.Gate != nil {
		if !h.Gate.AllowRate() {
			httpx.WriteError(w, http.StatusTooManyRequests, httpx.CodeRateLimited, "rate limit exceeded for this deployment tier; retry shortly")
			return
		}
		decision := h.Gate.Evaluate(op, params.GetConsentAffirmed())
		if !decision.Allowed {
			httpx.WriteError(w, http.StatusForbidden, httpx.CodeForbidden, decision.Reason+" — "+decision.RecoveryHint)
			return
		}
		forceScan = decision.ForceNSFWScan
		if decision.RecordConsent {
			if err := h.Gate.RecordConsent(r.Context(), op, decision.Weight); err != nil {
				h.logf("ai.submit consent log: %v", err)
			}
		}
	}

	inputKey, ok := h.storeOptionalInput(w, r, "file", meta.RequiresImage, "input")
	if !ok {
		return
	}
	maskKey, ok := h.storeOptionalInput(w, r, "mask", meta.RequiresMask, "mask")
	if !ok {
		return
	}

	plan, err := h.Engine.Plan(r.Context(), ai.PlanRequest{
		Operation:     op,
		ModelOverride: params.GetModelOverride(),
		AllowBYOK:     params.GetAllowByok(),
	})
	if err != nil {
		h.writePlanError(w, err)
		return
	}

	payload := ai.Payload{
		Operation:    op,
		InputKey:     inputKey,
		MaskKey:      maskKey,
		ModelID:      plan.ModelID,
		Backend:      h.modelBackend(plan.ModelID),
		Tier:         plan.Tier,
		GPU:          plan.GPUViable,
		AllowBYOK:    params.GetAllowByok(),
		AutoScanNSFW: params.GetAutoScanNsfw() || forceScan,
		Variations:   int(params.GetVariations()),
		Params:       paramsMap(params),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		h.logf("ai.submit marshal payload: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to prepare job")
		return
	}

	job, err := h.Jobs.Submit(r.Context(), internaljobs.Spec{
		Operation:        op,
		Lane:             ai.Lane(op),
		Payload:          raw,
		EstimatedSeconds: plan.EstimatedSeconds,
	})
	if err != nil {
		h.logf("ai.submit enqueue: %v", err)
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "failed to enqueue job")
		return
	}

	httpx.WriteProto(w, http.StatusAccepted, &aiv1.SubmitAIResponse{
		JobId:            job.ID,
		EstimatedSeconds: int32(plan.EstimatedSeconds),
		ModelId:          plan.ModelID,
		Tier:             plan.Tier,
		Warnings:         plan.Warnings,
	})
}

func (h *Deps) modelBackend(modelID string) string {
	if h.Engine == nil {
		return ""
	}
	model, ok := h.Engine.ModelByID(modelID)
	if !ok {
		return ""
	}
	return model.Backend
}

func (h *Deps) parseParams(w http.ResponseWriter, raw string) (*aiv1.AIParams, bool) {
	p := &aiv1.AIParams{}
	if raw != "" {
		if err := protojson.Unmarshal([]byte(raw), p); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid params: "+err.Error())
			return nil, false
		}
	}
	return p, true
}

// storeOptionalInput reads a multipart image part and stores it as a blob,
// returning its key. When required is false and the part is absent it returns
// ("", true). A required-but-missing or invalid part writes an error and returns
// ("", false).
func (h *Deps) storeOptionalInput(w http.ResponseWriter, r *http.Request, field string, required bool, prefix string) (string, bool) {
	present := r.MultipartForm != nil && len(r.MultipartForm.File[field]) > 0
	if !present {
		if required {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, fmt.Sprintf("operation requires a %q image part", field))
			return "", false
		}
		return "", true
	}
	file, header, err := r.FormFile(field)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, fmt.Sprintf("missing %q image part", field))
		return "", false
	}
	defer func() { _ = file.Close() }()
	inspected, err := h.Guard.Inspect(file)
	if err != nil {
		if errors.Is(err, storage.ErrTooLarge) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, err.Error())
		} else {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, err.Error())
		}
		return "", false
	}
	key := prefix + "/" + uuid.NewString() + extOf(header.Filename)
	if err := h.Store.Put(r.Context(), key, bytes.NewReader(inspected.Bytes), ""); err != nil {
		h.logf("ai.submit store %s: %v", field, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to store input")
		return "", false
	}
	return key, true
}

func (h *Deps) writePlanError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ai.ErrModelNotInstalled):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeInvalidRequest, err.Error())
	case errors.Is(err, backends.ErrNoneAvailable), errors.Is(err, backends.ErrNoProvider):
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, err.Error())
	case errors.Is(err, models.ErrUnknownOperation):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
	default:
		// ErrNoEnabledModel / ErrNotRunnable / ErrOverrideInvalid / probe failure
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, err.Error())
	}
}

func (h *Deps) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}

// extOf returns the lowercased file extension (with dot) of name, or "" .
func extOf(name string) string {
	for i := len(name) - 1; i >= 0 && name[i] != '/'; i-- {
		if name[i] == '.' {
			return name[i:]
		}
	}
	return ""
}
