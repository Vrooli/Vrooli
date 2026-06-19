package analysis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	internalanalysis "image-tools/internal/analysis"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
)

// Recorder records a synchronous op as a terminal durable job (satisfied by
// *internal/jobs.Manager). Analysis ops return structured data inline and are
// recorded for uniform observability.
type Recorder interface {
	Record(spec internaljobs.Spec, resultRef string, runErr error) (internaljobs.Job, error)
}

// Deps wires the analyze edge's seams.
type Deps struct {
	Service *internalanalysis.Service
	Jobs    Recorder
	Guard   storage.Guard
	Logger  *log.Logger
}

const maxMultipartMemory = 32 << 20

// analyzeHandler is POST /api/v1/analysis/{operation}: the multipart edge for
// the image→data ops (ocr / nsfw_classify / probe). The `file` part carries the
// image; the result is the proto-typed AnalyzeResponse. These run synchronously
// and are recorded as terminal jobs.
func (h *Deps) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	op := mux.Vars(r)["operation"]
	if !internalanalysis.Has(op) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, fmt.Sprintf("unknown analysis operation %q", op))
		return
	}
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "malformed multipart form")
		return
	}
	img, ok := h.readImage(w, r)
	if !ok {
		return
	}

	resp, runErr := h.analyze(r.Context(), op, img)
	if runErr != nil {
		_, _ = h.Jobs.Record(internaljobs.Spec{Operation: op, Lane: internaljobs.LaneCPU}, "", runErr)
		h.writeAnalyzeError(w, runErr)
		return
	}

	job, err := h.Jobs.Record(internaljobs.Spec{Operation: op, Lane: internaljobs.LaneCPU}, "", nil)
	if err != nil {
		h.logf("analysis.run record job: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to record job")
		return
	}
	resp.JobId = job.ID
	httpx.WriteProto(w, http.StatusOK, resp)
}

func (h *Deps) analyze(ctx context.Context, op string, img []byte) (*analysisv1.AnalyzeResponse, error) {
	switch op {
	case internalanalysis.OpProbe:
		res, err := internalanalysis.Probe(img)
		if err != nil {
			return nil, err
		}
		return &analysisv1.AnalyzeResponse{Result: &analysisv1.AnalyzeResponse_Probe{Probe: probeToProto(res)}}, nil
	case internalanalysis.OpOCR:
		res, err := h.Service.OCR(ctx, img)
		if err != nil {
			return nil, err
		}
		return &analysisv1.AnalyzeResponse{Result: &analysisv1.AnalyzeResponse_Ocr{Ocr: ocrToProto(res)}}, nil
	case internalanalysis.OpNSFW:
		res, err := h.Service.NSFW(ctx, img)
		if err != nil {
			return nil, err
		}
		return &analysisv1.AnalyzeResponse{Result: &analysisv1.AnalyzeResponse_Nsfw{Nsfw: nsfwToProto(res)}}, nil
	case internalanalysis.OpDuplicate:
		res, err := internalanalysis.DuplicateDetect(img)
		if err != nil {
			return nil, err
		}
		return &analysisv1.AnalyzeResponse{Result: &analysisv1.AnalyzeResponse_Duplicate{Duplicate: duplicateToProto(res)}}, nil
	case internalanalysis.OpQuality:
		res, err := internalanalysis.QualityAssess(img)
		if err != nil {
			return nil, err
		}
		return &analysisv1.AnalyzeResponse{Result: &analysisv1.AnalyzeResponse_Quality{Quality: qualityToProto(res)}}, nil
	default:
		return nil, fmt.Errorf("unsupported analysis operation %q", op)
	}
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

func (h *Deps) writeAnalyzeError(w http.ResponseWriter, err error) {
	if errors.Is(err, internalanalysis.ErrBackendUnavailable) {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, err.Error())
		return
	}
	httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, err.Error())
}

func (h *Deps) logf(format string, args ...any) {
	if h.Logger != nil {
		h.Logger.Printf(format, args...)
	}
}
