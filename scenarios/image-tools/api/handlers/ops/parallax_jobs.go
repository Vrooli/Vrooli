package ops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
	"image-tools/internal/httpx"
	"image-tools/internal/jobrunner"
	internaljobs "image-tools/internal/jobs"
	internalops "image-tools/internal/ops"
	"image-tools/internal/storage"
)

const parallaxJobOperation = "parallax"

type parallaxQueue interface {
	Submit(context.Context, internaljobs.Spec) (internaljobs.Job, error)
	GetByClientOperation(context.Context, string) (internaljobs.Job, error)
}
type parallaxPayload struct {
	Params *internalops.Params `json:"params"`
	Input  string              `json:"input"`
	Plates []string            `json:"plates"`
}

// RegisterParallaxRunner reuses the existing CPU lane, job cancellation and blob
// store. No render loop or job state lives in an HTTP request or in the CLI.
func RegisterParallaxRunner(dispatcher *jobrunner.Dispatcher, store BlobStore) {
	dispatcher.Register(parallaxJobOperation, func(ctx context.Context, job internaljobs.Job, emit func(int, string)) (internaljobs.Result, error) {
		var payload parallaxPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return internaljobs.Result{}, err
		}
		if payload.Params == nil || len(payload.Plates) != len(payload.Params.Plates) {
			return internaljobs.Result{}, fmt.Errorf("invalid saved parallax input closure")
		}
		read := func(key string) ([]byte, error) {
			r, _, err := store.Get(ctx, key)
			if err != nil {
				return nil, err
			}
			defer r.Close()
			data, err := io.ReadAll(io.LimitReader(r, 32<<20+1))
			if len(data) > 32<<20 {
				return nil, fmt.Errorf("parallax input exceeds 32MiB")
			}
			return data, err
		}
		source, err := read(payload.Input)
		if err != nil {
			return internaljobs.Result{}, err
		}
		for i, key := range payload.Plates {
			data, err := read(key)
			if err != nil {
				return internaljobs.Result{}, err
			}
			payload.Params.Plates[i].Image = data
		}
		emit(5, "Rendering prepared depth layers")
		result, err := internalops.ExecuteContext(ctx, "parallax", source, payload.Params)
		if err != nil {
			return internaljobs.Result{}, err
		}
		emit(95, "Publishing lossless frames and provenance")
		sum := sha256.Sum256(result.Bytes)
		key := "out/parallax-" + hex.EncodeToString(sum[:]) + ".zip"
		ref, err := store.Write(ctx, storage.OutputTarget{BlobKey: key}, bytes.NewReader(result.Bytes), result.Mime)
		if err != nil {
			return internaljobs.Result{}, err
		}
		return internaljobs.Result{Ref: ref, Meta: map[string]string{"format": "zip", "mime": "application/zip", "width": strconv.Itoa(result.Width), "height": strconv.Itoa(result.Height), "sha256": hex.EncodeToString(sum[:])}}, nil
	})
}

func (h *Deps) submitParallax(w http.ResponseWriter, r *http.Request, input []byte, p *internalops.Params) {
	queue, ok := h.Jobs.(parallaxQueue)
	if !ok {
		httpx.WriteError(w, 503, httpx.CodeInternal, "Parallax job runner is unavailable")
		return
	}
	operationID := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if operationID == "" || len(operationID) > 128 {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "Parallax requires an Idempotency-Key of 1..128 characters")
		return
	}
	if r.URL.Query().Get("path") != "" {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "Parallax publishes a managed job result; download it after completion")
		return
	}
	// Image pixels are excluded from JSON and enter the fingerprint explicitly,
	// in declared order. A retried operation can never adopt different layers.
	raw, marshalErr := json.Marshal(p)
	if marshalErr != nil {
		httpx.WriteError(w, 422, httpx.CodeInvalidRequest, "Parallax parameters must be finite and serializable")
		return
	}
	sum := sha256.New()
	sum.Write(raw)
	inputHash := sha256.Sum256(input)
	sum.Write(inputHash[:])
	for _, plate := range p.Plates {
		digest := sha256.Sum256(plate.Image)
		sum.Write(digest[:])
	}
	fingerprint := hex.EncodeToString(sum.Sum(nil))
	respond := func(job internaljobs.Job) {
		body, _ := protojson.Marshal(&opsv1.RunOpResponse{JobId: job.ID})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Image-Tools-Job-Id", job.ID)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(body)
	}
	if prior, err := queue.GetByClientOperation(r.Context(), operationID); err == nil {
		if prior.RequestFingerprint != fingerprint {
			httpx.WriteError(w, 409, httpx.CodeInvalidRequest, "Operation id already names different parallax inputs")
			return
		}
		respond(prior)
		return
	} else if !errors.Is(err, internaljobs.ErrNotFound) {
		httpx.WriteError(w, 500, httpx.CodeInternal, "Could not inspect the saved operation")
		return
	}
	// Validate the complete scene/path bounds before accepting expensive work.
	if err := internalops.ValidateParallaxRequest(p); err != nil {
		httpx.WriteError(w, 422, httpx.CodeInvalidRequest, err.Error())
		return
	}
	save := func(data []byte) (string, error) {
		s := sha256.Sum256(data)
		key := "in/parallax-" + hex.EncodeToString(s[:])
		return h.Store.Write(r.Context(), storage.OutputTarget{BlobKey: key}, bytes.NewReader(data), "application/octet-stream")
	}
	inputRef, err := save(input)
	if err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "Could not retain parallax source")
		return
	}
	payload := parallaxPayload{Params: p, Input: inputRef}
	for _, plate := range p.Plates {
		ref, err := save(plate.Image)
		if err != nil {
			httpx.WriteError(w, 500, httpx.CodeInternal, "Could not retain parallax layer")
			return
		}
		payload.Plates = append(payload.Plates, ref)
	}
	payloadJSON, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "Could not retain parallax parameters")
		return
	}
	job, err := queue.Submit(r.Context(), internaljobs.Spec{Operation: parallaxJobOperation, Lane: internaljobs.LaneCPU, Payload: payloadJSON, ClientOperationID: operationID, RequestFingerprint: fingerprint, ExecutionTimeoutSeconds: 180})
	if errors.Is(err, internaljobs.ErrIdempotencyConflict) {
		httpx.WriteError(w, 409, httpx.CodeInvalidRequest, err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, "Could not submit the retained parallax operation")
		return
	}
	respond(job)
}
