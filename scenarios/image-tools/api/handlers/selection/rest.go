package selection

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	internalselection "image-tools/internal/selection"
	"image-tools/internal/storage"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

// BlobStore is the storage surface the segment edge needs to persist the
// produced mask (satisfied by *internal/storage.Store).
type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, mime string) error
}

// Recorder records the segmentation as a terminal durable job (uniform
// observability), satisfied by *internal/jobs.Manager.
type Recorder interface {
	Record(spec internaljobs.Spec, resultRef string, runErr error) (internaljobs.Job, error)
}

// Deps wires the segment edge's seams.
type Deps struct {
	Service *internalselection.Service
	Store   BlobStore
	Jobs    Recorder
	Guard   storage.Guard
	Logger  *log.Logger
}

const (
	maxMultipartMemory = 32 << 20
	segmentOp          = "segment"
)

// segmentHandler is POST /api/v1/selection/segment: the multipart smart-select
// edge. The `file` part carries the image; the `params` part carries
// SegmentParams (protojson). It runs the built-in region-grow synchronously,
// stores the produced mask as a blob, classifies the region, attaches the
// contextual edit menu, and returns SegmentResult.
func (h *Deps) segmentHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "malformed multipart form")
		return
	}
	params, ok := h.parseParams(w, r.FormValue("params"))
	if !ok {
		return
	}
	img, ok := h.readImage(w, r)
	if !ok {
		return
	}

	res, runErr := h.Service.Segment(r.Context(), img, paramsFromProto(params))
	if runErr != nil {
		_, _ = h.Jobs.Record(internaljobs.Spec{Operation: segmentOp, Lane: internaljobs.LaneCPU}, "", runErr)
		h.writeSegmentError(w, runErr)
		return
	}

	maskKey := "mask/" + uuid.NewString() + ".png"
	if err := h.Store.Put(r.Context(), maskKey, bytes.NewReader(res.MaskPNG), "image/png"); err != nil {
		h.logf("selection.segment store mask: %v", err)
		_, _ = h.Jobs.Record(internaljobs.Spec{Operation: segmentOp, Lane: internaljobs.LaneCPU}, "", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to store mask")
		return
	}

	job, err := h.Jobs.Record(internaljobs.Spec{Operation: segmentOp, Lane: internaljobs.LaneCPU}, maskKey, nil)
	if err != nil {
		h.logf("selection.segment record job: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to record job")
		return
	}

	httpx.WriteProto(w, http.StatusOK, &selectionv1.SegmentResult{
		JobId:          job.ID,
		MaskRef:        maskKey,
		Box:            boxToProto(res.Box),
		RegionClass:    res.RegionClass,
		Confidence:     res.Confidence,
		AreaFraction:   res.AreaFraction,
		Tier:           res.Tier,
		ModelId:        res.ModelID,
		SuggestedEdits: editsToProto(res.Edits),
		Warnings:       res.Warnings,
	})
}

func (h *Deps) parseParams(w http.ResponseWriter, raw string) (*selectionv1.SegmentParams, bool) {
	p := &selectionv1.SegmentParams{}
	if raw != "" {
		if err := protojson.Unmarshal([]byte(raw), p); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid params: "+err.Error())
			return nil, false
		}
	}
	return p, true
}

func (h *Deps) readImage(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	file, _, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "missing \"file\" image part")
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

// writeSegmentError maps a segmentation error to an HTTP status. A bad
// mode/seed is an invalid request; a decode failure is unprocessable.
func (h *Deps) writeSegmentError(w http.ResponseWriter, err error) {
	httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, fmt.Sprintf("segmentation failed: %v", err))
}

func (h *Deps) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}
