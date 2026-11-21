package analysis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	internalanalysis "image-tools/internal/analysis"
	"image-tools/internal/httpx"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
)

// Scheduler owns analysis admission, execution and terminal persistence. The
// multipart API waits once for the job while client disconnects leave it owned.
type Scheduler interface {
	SubmitTask(context.Context, internaljobs.Spec, internaljobs.Runner) (internaljobs.Job, error)
	Wait(context.Context, string) (internaljobs.Job, error)
}

// Deps wires the analyze edge's seams.
type Deps struct {
	Service *internalanalysis.Service
	Jobs    Scheduler
	Guard   storage.Guard
	Store   *storage.Store
	Logger  *log.Logger
}

const maxMultipartMemory = 32 << 20

// analyzeHandler is POST /api/v1/analysis/{operation}: the multipart edge for
// the image→data ops (ocr / nsfw_classify / probe). The `file` part carries the
// image; the result is the proto-typed AnalyzeResponse. The response waits for the common
// scheduler and includes its durable job identity.
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
	defer r.MultipartForm.RemoveAll()
	img, ok := h.readImage(w, r)
	if !ok {
		return
	}

	// Keep queued inputs in the blob owner rather than retaining large request
	// byte slices for every waiting job.
	inputRef := ""
	if h.Store != nil {
		inputRef = "inputs/analysis/" + uuid.NewString()
		if err := h.Store.Put(r.Context(), inputRef, bytes.NewReader(img), "application/octet-stream"); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to retain analysis input")
			return
		}
		img = nil
	}
	type completion struct {
		response *analysisv1.AnalyzeResponse
		err      error
	}
	completed := make(chan completion, 1)
	job, err := h.Jobs.SubmitTask(r.Context(), internaljobs.Spec{Operation: op, Lane: internaljobs.LaneCPU, Priority: internaljobs.PriorityInteractive}, func(ctx context.Context, job internaljobs.Job, emit func(int, string)) (internaljobs.Result, error) {
		source := img
		if inputRef != "" {
			rc, _, err := h.Store.Get(ctx, inputRef)
			if err != nil {
				return internaljobs.Result{}, err
			}
			source, err = io.ReadAll(io.LimitReader(rc, maxMultipartMemory+1))
			_ = rc.Close()
			if err != nil {
				return internaljobs.Result{}, err
			}
		}
		emit(10, "analyzing")
		resp, runErr := h.analyze(ctx, op, source)
		completed <- completion{resp, runErr}
		if runErr != nil {
			return internaljobs.Result{}, runErr
		}
		resp.JobId = job.ID
		if h.Store == nil {
			return internaljobs.Result{}, nil
		}
		encoded, err := protojson.Marshal(resp)
		if err != nil {
			return internaljobs.Result{}, err
		}
		ref := "outputs/" + job.ID + "/analysis.json"
		err = h.Store.Put(ctx, ref, bytes.NewReader(encoded), "application/json")
		return internaljobs.Result{Ref: ref}, err
	})
	if err != nil {
		h.logf("analysis submit: %v", err)
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "failed to schedule analysis")
		return
	}
	w.Header().Set("X-Image-Tools-Job-ID", job.ID)
	job, err = h.Jobs.Wait(r.Context(), job.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, err.Error())
		return
	}
	var result completion
	select {
	case result = <-completed:
	default:
	}
	if job.State != internaljobs.StateSucceeded {
		if result.err == nil {
			result.err = errors.New(job.Error)
			if strings.Contains(job.Error, internalanalysis.ErrBackendUnavailable.Error()) {
				result.err = fmt.Errorf("%w: %s", internalanalysis.ErrBackendUnavailable, job.Error)
			}
		}
		h.writeAnalyzeError(w, result.err)
		return
	}
	if result.response == nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "analysis completed without a response")
		return
	}
	result.response.JobId = job.ID
	httpx.WriteProto(w, http.StatusOK, result.response)
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
	inputRef := r.FormValue("input_ref")
	file, _, err := r.FormFile("file")
	if inputRef != "" && err == nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "provide either \"file\" or \"input_ref\", not both")
		_ = file.Close()
		return nil, false
	}
	if inputRef != "" {
		if h.Store == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "stored input references are unavailable")
			return nil, false
		}
		rc, _, getErr := h.Store.Get(r.Context(), inputRef)
		if getErr != nil {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, fmt.Sprintf("input_ref %q not found", inputRef))
			return nil, false
		}
		defer func() { _ = rc.Close() }()
		inspected, inspectErr := h.Guard.Inspect(io.LimitReader(rc, maxMultipartMemory+1))
		if inspectErr != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, inspectErr.Error())
			return nil, false
		}
		return inspected.Bytes, true
	}
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
