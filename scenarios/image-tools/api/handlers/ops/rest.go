package ops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	internalops "image-tools/internal/ops"
	"image-tools/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
)

// BlobStore is the storage surface the REST edge needs (satisfied by
// *internal/storage.Store). Declared at the consumer so tests inject a fake.
type BlobStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Write(ctx context.Context, target storage.OutputTarget, r io.Reader, mime string) (string, error)
}

// Recorder records a synchronous op as a terminal durable job (satisfied by
// *internal/jobs.Manager). Deterministic ops are instant, so they execute
// inline and are recorded for uniform observability rather than queued.
type Recorder interface {
	Record(spec internaljobs.Spec, resultRef string, runErr error) (internaljobs.Job, error)
}

// Deps wires the REST edge's seams.
type Deps struct {
	Store  BlobStore
	Jobs   Recorder
	Guard  storage.Guard
	Logger *log.Logger
}

// maxMultipartMemory bounds in-memory multipart buffering; larger parts spill
// to temp files. The Guard enforces the real per-image byte cap downstream.
const maxMultipartMemory = 32 << 20

// runHandler is POST /api/v1/ops/{operation}: the multipart execution edge for
// deterministic image operations. Image bytes ride multipart (a proto field
// can't carry them); the PARAMETERS stay proto-typed (OpParams as protojson in
// the `params` part) and the RESULT metadata is proto-typed (RunOpResponse).
//
// Output modes (query `output`):
//   - "bytes" (default): stream the result bytes with the result MIME and the
//     job metadata in X-Image-Tools-* headers (the one-round-trip CLI path);
//   - "blob": store the result and return the proto-typed RunOpResponse JSON.
//
// A `path` query writes the result directly to a caller-owned host path
// (server-side) instead of the managed blob store.
func (h *Deps) runHandler(w http.ResponseWriter, r *http.Request) {
	operation := mux.Vars(r)["operation"]
	if !internalops.Has(operation) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, fmt.Sprintf("unknown operation %q", operation))
		return
	}

	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "malformed multipart form")
		return
	}

	inputBytes, ok := h.readImagePart(w, r, "file")
	if !ok {
		return
	}

	params, ok := h.parseParams(w, operation, r.FormValue("params"))
	if !ok {
		return
	}
	// Output encoding is a property of the OUTPUT, orthogonal to the operation:
	// `?format=`/`?quality=` set the result format/quality for any op (the CLI
	// derives them from the --out extension) without bloating every op's typed
	// params. An op that owns these in its params (convert/compress) wins.
	applyOutputOverrides(params, r.URL.Query())
	if operation == "overlay" {
		if overlay, present := h.readOptionalImagePart(w, r, "overlay"); present {
			if overlay == nil {
				return // a read error was already written
			}
			params.OverlayImage = overlay
		}
	}

	res, runErr := internalops.Execute(operation, inputBytes, params)
	if runErr != nil {
		// Record the failure for observability, then surface it as a 422 — these
		// are client-side bad params / undecodable images, not server faults.
		_, _ = h.Jobs.Record(internaljobs.Spec{Operation: operation, Lane: internaljobs.LaneCPU}, "", runErr)
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, runErr.Error())
		return
	}

	ref, err := h.writeResult(r.Context(), r.URL.Query().Get("path"), res)
	if err != nil {
		h.logf("ops.run write result: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to write result")
		return
	}

	job, err := h.Jobs.Record(internaljobs.Spec{Operation: operation, Lane: internaljobs.LaneCPU}, ref, nil)
	if err != nil {
		h.logf("ops.run record job: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to record job")
		return
	}

	h.writeRunResponse(w, r.URL.Query().Get("output"), job, res, ref)
}

// blobHandler is GET /api/v1/blobs/{key}: serves a managed result blob's bytes.
// Browser-facing binary fetch with no generated client (RESTReasonOpsProbe).
func (h *Deps) blobHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	rc, mime, err := h.Store.Get(r.Context(), key)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "blob not found")
		return
	}
	defer func() { _ = rc.Close() }()
	if mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	if _, err := io.Copy(w, rc); err != nil {
		h.logf("ops.blob copy: %v", err)
	}
}

// --- helpers ---

func (h *Deps) readImagePart(w http.ResponseWriter, r *http.Request, field string) ([]byte, bool) {
	file, _, err := r.FormFile(field)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, fmt.Sprintf("missing %q image part", field))
		return nil, false
	}
	defer func() { _ = file.Close() }()
	inspected, err := h.Guard.Inspect(file)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrTooLarge):
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, err.Error())
		default:
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, err.Error())
		}
		return nil, false
	}
	return inspected.Bytes, true
}

// readOptionalImagePart returns (bytes, true) when the part is present and
// valid, (nil, true) when present but invalid (a response was already written),
// and (nil, false) when the part is absent.
func (h *Deps) readOptionalImagePart(w http.ResponseWriter, r *http.Request, field string) ([]byte, bool) {
	if r.MultipartForm == nil || len(r.MultipartForm.File[field]) == 0 {
		return nil, false
	}
	b, ok := h.readImagePart(w, r, field)
	if !ok {
		return nil, true
	}
	return b, true
}

func (h *Deps) parseParams(w http.ResponseWriter, operation, raw string) (*internalops.Params, bool) {
	var pb *opsv1.OpParams
	if raw != "" {
		pb = &opsv1.OpParams{}
		if err := protojson.Unmarshal([]byte(raw), pb); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid params: "+err.Error())
			return nil, false
		}
	}
	params, err := translateParams(operation, pb)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		return nil, false
	}
	return params, true
}

func (h *Deps) writeResult(ctx context.Context, outputPath string, res internalops.RunResult) (string, error) {
	if outputPath != "" {
		return h.Store.Write(ctx, storage.OutputTarget{LocalPath: outputPath}, bytes.NewReader(res.Bytes), res.Mime)
	}
	key := "out/" + uuid.NewString() + "." + extFor(res.Format)
	return h.Store.Write(ctx, storage.OutputTarget{BlobKey: key}, bytes.NewReader(res.Bytes), res.Mime)
}

func (h *Deps) writeRunResponse(w http.ResponseWriter, output string, job internaljobs.Job, res internalops.RunResult, ref string) {
	if output == "bytes" || output == "" {
		w.Header().Set("Content-Type", res.Mime)
		w.Header().Set("X-Image-Tools-Job-Id", job.ID)
		w.Header().Set("X-Image-Tools-Result-Ref", ref)
		w.Header().Set("X-Image-Tools-Format", res.Format)
		if res.Width > 0 {
			w.Header().Set("X-Image-Tools-Width", strconv.Itoa(res.Width))
			w.Header().Set("X-Image-Tools-Height", strconv.Itoa(res.Height))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res.Bytes)
		return
	}
	httpx.WriteProto(w, http.StatusOK, &opsv1.RunOpResponse{
		JobId: job.ID,
		Result: &opsv1.OpResult{
			Ref:       ref,
			Format:    res.Format,
			Mime:      res.Mime,
			Width:     int32(res.Width),
			Height:    int32(res.Height),
			SizeBytes: int64(len(res.Bytes)),
		},
	})
}

// applyOutputOverrides fills params.Format/Quality from the query when the op
// did not set them itself, so `--out result.webp` yields WebP for any op.
func applyOutputOverrides(params *internalops.Params, q url.Values) {
	if params.Format == "" {
		if f := q.Get("format"); f != "" {
			params.Format = f
		}
	}
	if params.Quality == 0 {
		if v, err := strconv.Atoi(q.Get("quality")); err == nil && v > 0 {
			params.Quality = v
		}
	}
}

func (h *Deps) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}

func extFor(format string) string {
	switch format {
	case internalops.FormatJPEG:
		return "jpg"
	case "json":
		return "json"
	default:
		return format
	}
}
