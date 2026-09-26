package diff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	internaldiff "image-tools/internal/diff"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
)

// BlobStore is the storage surface the compare edge needs to persist the
// produced heat-map (satisfied by *internal/storage.Store).
type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, mime string) error
}

// Recorder records the comparison as a terminal durable job (uniform
// observability), satisfied by *internal/jobs.Manager.
type Recorder interface {
	Record(spec internaljobs.Spec, resultRef string, runErr error) (internaljobs.Job, error)
}

// Deps wires the compare edge's seams.
type Deps struct {
	Store  BlobStore
	Jobs   Recorder
	Guard  storage.Guard
	Logger *log.Logger
}

const (
	maxMultipartMemory = 32 << 20
	compareOp          = "image_diff"
)

// compareHandler is POST /api/v1/diff/compare: the multipart visual-comparison
// edge. The `base` and `compare` parts carry the two images; the `params` part
// carries DiffParams (protojson). It runs the pure-Go pixel + perceptual
// comparison synchronously, stores the heat-map as a blob (when requested), and
// returns DiffResult with the full metric set.
func (h *Deps) compareHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "malformed multipart form")
		return
	}
	params, ok := h.parseParams(w, r.FormValue("params"))
	if !ok {
		return
	}
	baseImg, ok := h.readImage(w, r, "base")
	if !ok {
		return
	}
	cmpImg, ok := h.readImage(w, r, "compare")
	if !ok {
		return
	}

	// IncludeHeatmap defaults true unless the caller explicitly sent it false.
	internalParams := paramsFromProto(params)

	res, runErr := internaldiff.Compare(baseImg, cmpImg, internalParams)
	if runErr != nil {
		_, _ = h.Jobs.Record(internaljobs.Spec{Operation: compareOp, Lane: internaljobs.LaneCPU}, "", runErr)
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, fmt.Sprintf("comparison failed: %v", runErr))
		return
	}

	var heatmapRef string
	if len(res.HeatmapPNG) > 0 {
		heatmapRef = "diff/" + uuid.NewString() + ".png"
		if err := h.Store.Put(r.Context(), heatmapRef, bytes.NewReader(res.HeatmapPNG), "image/png"); err != nil {
			h.logf("diff.compare store heatmap: %v", err)
			_, _ = h.Jobs.Record(internaljobs.Spec{Operation: compareOp, Lane: internaljobs.LaneCPU}, "", err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to store heat-map")
			return
		}
	}

	job, err := h.Jobs.Record(internaljobs.Spec{Operation: compareOp, Lane: internaljobs.LaneCPU}, heatmapRef, nil)
	if err != nil {
		h.logf("diff.compare record job: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to record job")
		return
	}

	httpx.WriteProto(w, http.StatusOK, resultToProto(job.ID, heatmapRef, res))
}

func (h *Deps) parseParams(w http.ResponseWriter, raw string) (*diffv1.DiffParams, bool) {
	// Default the heat-map on; an explicit params body may turn it off.
	p := &diffv1.DiffParams{IncludeHeatmap: true}
	if raw != "" {
		if err := protojson.Unmarshal([]byte(raw), p); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid params: "+err.Error())
			return nil, false
		}
	}
	return p, true
}

func (h *Deps) readImage(w http.ResponseWriter, r *http.Request, part string) ([]byte, bool) {
	file, _, err := r.FormFile(part)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, fmt.Sprintf("missing %q image part", part))
		return nil, false
	}
	defer func() { _ = file.Close() }()
	inspected, err := h.Guard.Inspect(file)
	if err != nil {
		if errors.Is(err, storage.ErrTooLarge) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, err.Error())
		} else {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, err.Error())
		}
		return nil, false
	}
	return inspected.Bytes, true
}

func (h *Deps) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}
