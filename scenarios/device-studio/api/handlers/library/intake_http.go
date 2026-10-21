package library

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"

	"device-studio/handlers/revisionapi"
	"device-studio/internal/httpx"
	domain "device-studio/internal/library"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	"google.golang.org/protobuf/encoding/protojson"
)

const maxIntakeMultipartBytes int64 = domain.MaxGeometryBytes + (1 << 20)

type intakeOperations interface {
	BeginImport(context.Context, domain.IntakeClaims, string, int64, io.Reader) (domain.StagedIntakeImport, error)
	Open(context.Context, string, uint32) (domain.IntakeCandidate, io.ReadCloser, error)
}

type intakeHTTPHandler struct {
	service intakeOperations
	db      *database.RoutedDB
	roots   *filerouting.RoutedRoots
	logger  *log.Logger
}

func newIntakeUploadHandler(service intakeOperations, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	h := &intakeHTTPHandler{service: service, db: db, roots: roots, logger: logger}
	return http.HandlerFunc(h.upload)
}

func newIntakeContentHandler(service intakeOperations, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "geometry intake is unavailable")
			return
		}
		ctx, err := pinIntakeRoute(r.Context(), db, roots)
		if err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "routed intake storage is unavailable")
			return
		}
		id := mux.Vars(r)["candidate_id"]
		revision, err := parseIntakeRevision(mux.Vars(r)["revision"])
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "candidate revision is invalid")
			return
		}
		candidate, body, err := service.Open(ctx, id, revision)
		if err != nil {
			writeIntakeHTTPError(w, logger, err)
			return
		}
		defer body.Close()
		w.Header().Set("Content-Type", candidate.MIME)
		w.Header().Set("Content-Length", strconv.FormatInt(candidate.SizeBytes, 10))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, no-store")
		if _, err := io.Copy(w, body); err != nil {
			logger.Printf("library intake content response interrupted: %v", err)
			panic(http.ErrAbortHandler)
		}
	})
}

func (h *intakeHTTPHandler) upload(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "geometry intake is unavailable")
		return
	}
	ctx, err := pinIntakeRoute(r.Context(), h.db, h.roots)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "routed intake storage is unavailable")
		return
	}
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "multipart/form-data is required")
		return
	}
	limited := http.MaxBytesReader(w, r.Body, maxIntakeMultipartBytes)
	buffered := bufio.NewReader(limited)
	mr := multipart.NewReader(buffered, params["boundary"])
	meta, err := mr.NextRawPart()
	if err != nil || meta.FormName() != "metadata" || meta.FileName() != "" {
		writeIntakeMultipartError(w, err)
		return
	}
	claims, err := readIntakeClaims(meta)
	closeErr := meta.Close()
	if err != nil || closeErr != nil {
		writeIntakeMultipartError(w, errors.Join(err, closeErr))
		return
	}
	file, err := mr.NextRawPart()
	if err != nil || file.FormName() != "file" || file.FileName() == "" || file.Header.Get("Content-Transfer-Encoding") != "" {
		writeIntakeMultipartError(w, err)
		return
	}
	declaredSize := int64(-1)
	if raw := file.Header.Get("Content-Length"); raw != "" {
		declaredSize, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || declaredSize < 0 {
			_ = file.Close()
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file size is invalid")
			return
		}
	}
	staged, err := h.service.BeginImport(ctx, claims, file.FileName(), declaredSize, file)
	if err != nil {
		_ = file.Close()
		writeIntakeHTTPError(w, h.logger, err)
		return
	}
	if staged == nil {
		_ = file.Close()
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "geometry intake failed")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = staged.Abort(ctx)
		}
	}()
	if err := file.Close(); err != nil {
		writeIntakeMultipartError(w, err)
		return
	}
	if _, err := mr.NextRawPart(); err != io.EOF {
		writeIntakeMultipartError(w, err)
		return
	}
	if _, err := io.Copy(io.Discard, buffered); err != nil {
		writeIntakeMultipartError(w, err)
		return
	}
	candidate, err := staged.Commit(ctx)
	if err != nil {
		writeIntakeHTTPError(w, h.logger, err)
		return
	}
	committed = true
	projected, err := candidateProto(candidate)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "intake metadata projection failed")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	payload, err := protojson.Marshal(&v1.GetIntakeCandidateResponse{Candidate: projected})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "intake response encoding failed")
		return
	}
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(append(payload, '\n')); err != nil {
		h.logger.Printf("write intake response: %v", err)
	}
}

func readIntakeClaims(part *multipart.Part) (domain.IntakeClaims, error) {
	data, err := io.ReadAll(io.LimitReader(part, 64<<10+1))
	if err != nil {
		return domain.IntakeClaims{}, err
	}
	if len(data) > 64<<10 {
		return domain.IntakeClaims{}, errors.New("intake metadata exceeds its size limit")
	}
	var message v1.IntakeCandidateClaims
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &message); err != nil {
		return domain.IntakeClaims{}, err
	}
	return domain.IntakeClaims{CandidateID: message.CandidateId, DisplayName: message.DisplayName, SourceLabel: message.SourceLabel, LicenseClaim: message.LicenseClaim, Attribution: message.Attribution, DeclaredUses: message.DeclaredUses, ParentID: message.ParentCandidateId, ParentRevision: message.ParentRevision, ParentSHA256: message.ParentSha256}, nil
}

func candidateProto(candidate domain.IntakeCandidate) (*v1.IntakeCandidateRevision, error) {
	out := &v1.IntakeCandidateRevision{}
	if err := revisionapi.Encode(candidate, out); err != nil {
		return nil, err
	}
	return out, nil
}

func pinIntakeRoute(ctx context.Context, db *database.RoutedDB, roots *filerouting.RoutedRoots) (context.Context, error) {
	if !database.IsTestMode(ctx) {
		return ctx, nil
	}
	if db == nil || roots == nil {
		return nil, errors.New("test intake route is not configured")
	}
	leaseID := db.ActiveLeaseID()
	if leaseID == "" {
		return nil, errors.New("test intake database lease is unavailable")
	}
	pinned, err := db.PinTestPool(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	return roots.PinTestRoots(pinned, leaseID)
}

func writeIntakeHTTPError(w http.ResponseWriter, logger *log.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrIntakeTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "GLB exceeds the local geometry size limit")
	case errors.Is(err, domain.ErrInvalidIntake):
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
	case errors.Is(err, domain.ErrIntakeConflict):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeConflict, "candidate id is already bound to different bytes or claims")
	case errors.Is(err, domain.ErrIntakeNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "intake candidate not found")
	default:
		if logger != nil {
			logger.Printf("library local geometry intake failed: %v", err)
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "geometry intake failed")
	}
}

func writeIntakeMultipartError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "multipart request exceeds the intake envelope limit")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid multipart geometry intake")
}
