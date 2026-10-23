package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"image-tools/internal/ai"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/safety"
	"image-tools/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	GetByClientOperation(ctx context.Context, operationID string) (internaljobs.Job, error)
}

// Gate is the Responsible-Use deployment gate the submit edge enforces
// (satisfied by *internal/safety.Gate). Nil disables the gate (the local
// behaviour: unrestricted).
type Gate interface {
	Evaluate(op string, consentAffirmed bool) safety.Decision
	EvaluateWeight(weight safety.Weight, op string, consentAffirmed bool) safety.Decision
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
	defer r.MultipartForm.RemoveAll()

	params, ok := h.parseParams(w, r.FormValue("params"))
	if !ok {
		return
	}
	operationID := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	forceScan, err := h.preflightAdmission(r.Context(), op, params, operationID)
	if err != nil {
		writeAdmissionError(w, err)
		return
	}

	inputRef := strings.TrimSpace(r.FormValue("input_ref"))
	inputKey, ok := h.storeOptionalInput(w, r, "file", meta.RequiresImage && inputRef == "", "input")
	if !ok {
		return
	}
	if inputRef != "" {
		if inputKey != "" {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "provide either file or input_ref, not both")
			return
		}
		if !meta.RequiresImage {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "input_ref is only valid for image-input operations")
			return
		}
		// input_ref names an already-owned Image Tools blob. Cross-scenario
		// callers never provide filesystem paths or proxy the image bytes.
		reader, _, getErr := h.Store.Get(r.Context(), inputRef)
		if getErr != nil {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "input_ref does not identify an Image Tools artifact")
			return
		}
		_ = reader.Close()
		inputKey = inputRef
	}
	maskKey, ok := h.storeOptionalInput(w, r, "mask", meta.RequiresMask, "mask")
	if !ok {
		return
	}
	response, err := h.admitStored(r.Context(), op, params, inputKey, maskKey, operationID, forceScan)
	if err != nil {
		writeAdmissionError(w, err)
		return
	}
	httpx.WriteProto(w, http.StatusAccepted, response)
}

func writeAdmissionError(w http.ResponseWriter, err error) {
	var rejection *admissionError
	if errors.As(err, &rejection) {
		httpx.WriteError(w, rejection.status, rejection.code, rejection.message)
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "AI admission unavailable")
}

func (h *Deps) requestFingerprint(ctx context.Context, op string, params *aiv1.AIParams, inputKey, maskKey string) (string, error) {
	canonical, err := (proto.MarshalOptions{Deterministic: true}).Marshal(params)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write([]byte(op + "\n"))
	hash.Write(canonical)
	hash.Write([]byte{0})
	if inputKey != "" || maskKey != "" {
		// Fixed-size per-role digests are unambiguous and independent of random
		// upload keys, multipart boundaries, filenames and filename extensions.
		for _, key := range []string{inputKey, maskKey} {
			if key == "" {
				hash.Write(make([]byte, sha256.Size))
				continue
			}
			r, _, err := h.Store.Get(ctx, key)
			if err != nil {
				return "", err
			}
			digest := sha256.New()
			const maxInput = 64 << 20
			n, err := io.Copy(digest, io.LimitReader(r, maxInput+1))
			closeErr := r.Close()
			if err != nil || closeErr != nil {
				return "", errors.Join(err, closeErr)
			}
			if n > maxInput {
				return "", fmt.Errorf("retained input exceeds 64MiB")
			}
			hash.Write(digest.Sum(nil))
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

type operationStatus struct {
	JobID      string            `json:"job_id"`
	Operation  string            `json:"operation"`
	ModelID    string            `json:"model_id,omitempty"`
	Tier       string            `json:"tier,omitempty"`
	State      string            `json:"state"`
	Progress   int               `json:"progress"`
	Message    string            `json:"message,omitempty"`
	Error      string            `json:"error,omitempty"`
	ResultRef  string            `json:"result_ref,omitempty"`
	ResultMeta map[string]string `json:"result_meta,omitempty"`
}

// Retained admissions must keep their original operation/model identity even
// while that model is currently unavailable. Empty or mismatched saved JSON is
// corruption, not a successful operation with missing presentation fields.
func decodeSavedAIPayload(job internaljobs.Job, payload *ai.Payload) error {
	if err := json.Unmarshal(job.Payload, payload); err != nil {
		return err
	}
	if payload.ModelID == "" || payload.Operation == "" || payload.Operation != job.Operation {
		return fmt.Errorf("saved AI operation identity is incomplete or inconsistent")
	}
	return nil
}

func (h *Deps) operationHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["operation_id"])
	if _, err := uuid.Parse(id); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid operation id")
		return
	}
	job, err := h.Jobs.GetByClientOperation(r.Context(), id)
	if err != nil {
		if errors.Is(err, internaljobs.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "operation was not accepted")
		} else {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "operation status unavailable")
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var payload ai.Payload
	if err := decodeSavedAIPayload(job, &payload); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "saved operation metadata is invalid")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(operationStatus{JobID: job.ID, Operation: job.Operation, ModelID: payload.ModelID, Tier: payload.Tier, State: string(job.State), Progress: job.Progress, Message: job.Message, Error: job.Error, ResultRef: job.ResultRef, ResultMeta: job.Meta})
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
	digest := sha256.Sum256(inspected.Bytes)
	key := fmt.Sprintf("%s/%x%s", prefix, digest, extOf(header.Filename))
	if err := h.Store.Put(r.Context(), key, bytes.NewReader(inspected.Bytes), ""); err != nil {
		h.logf("ai.submit store %s: %v", field, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to store input")
		return "", false
	}
	return key, true
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
