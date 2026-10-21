package projects

import (
	"bufio"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"

	"device-studio/internal/httpx"
	domain "device-studio/internal/projects"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
)

const maxMediaMultipartBytes int64 = domain.MaxSourceMediaBytes + (1 << 20)

type mediaUploadHandler struct {
	projects   domain.Service
	media      domain.SourceMediaService
	inspection SourceInspectionCoordinator
	activator  RouteActivator
	db         *database.RoutedDB
	roots      *filerouting.RoutedRoots
	logger     *log.Logger
	maxBytes   int64
}

func newMediaContentHandler(media domain.SourceMediaService, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if media == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "source media is unavailable")
			return
		}
		vars := mux.Vars(r)
		row, body, err := media.Open(r.Context(), vars["project_id"], vars["media_id"])
		if err != nil {
			var missing domain.ErrProjectNotFound
			var mediaMissing domain.ErrSourceMediaNotFound
			var invalid domain.ErrInvalidProject
			switch {
			case errors.As(err, &missing), errors.As(err, &mediaMissing):
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "source media not found")
			case errors.As(err, &invalid):
				httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid project")
			default:
				logger.Printf("projects source media content failed: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "source media unavailable")
			}
			return
		}
		defer body.Close()
		w.Header().Set("Content-Type", row.MediaType)
		w.Header().Set("Content-Length", strconv.FormatInt(row.SizeBytes, 10))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, no-store")
		if _, err := io.Copy(w, body); err != nil {
			logger.Printf("projects source media response interrupted: %v", err)
			// Headers may already be on the wire. Abort the response so clients
			// cannot mistake a truncated, unverified body for a complete 200.
			panic(http.ErrAbortHandler)
		}
	})
}

func newMediaUploadHandler(projects domain.Service, media domain.SourceMediaService, logger *log.Logger) http.Handler {
	return newMediaUploadHandlerWithInspection(projects, media, nil, nil, nil, nil, logger)
}

func newMediaUploadHandlerWithInspection(projects domain.Service, media domain.SourceMediaService, inspection SourceInspectionCoordinator, activator RouteActivator, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc((&mediaUploadHandler{projects: projects, media: media, inspection: inspection, activator: activator, db: db, roots: roots, logger: logger, maxBytes: maxMediaMultipartBytes}).upload)
}

func (h *mediaUploadHandler) upload(w http.ResponseWriter, r *http.Request) {
	if h.media == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "source media is unavailable")
		return
	}
	ctx, routeErr := pinProjectRoute(r.Context(), h.db, h.roots)
	if routeErr != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "routed source media storage is unavailable")
		return
	}
	projectID := mux.Vars(r)["project_id"]
	if _, err := h.projects.Get(ctx, projectID); err != nil {
		var missing domain.ErrProjectNotFound
		var invalid domain.ErrInvalidProject
		if errors.As(err, &missing) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "project not found")
		} else if errors.As(err, &invalid) {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid project")
		} else {
			h.logger.Printf("projects source media project lookup failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "source media import failed")
		}
		return
	}
	limit := h.maxBytes
	if limit <= 0 {
		limit = maxMediaMultipartBytes
	}
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "multipart/form-data is required")
		return
	}
	boundedBody := http.MaxBytesReader(w, r.Body, limit)
	bufferedBody := bufio.NewReader(boundedBody)
	mr := multipart.NewReader(bufferedBody, params["boundary"])
	part, err := mr.NextRawPart()
	if err != nil {
		writeMultipartError(w, err)
		return
	}
	if part.FormName() != "file" || part.FileName() == "" || part.Header.Get("Content-Transfer-Encoding") != "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "exactly one binary file field is required")
		return
	}
	filename := part.FileName()
	declaredSize := int64(-1)
	if value := part.Header.Get("Content-Length"); value != "" {
		declaredSize, err = strconv.ParseInt(value, 10, 64)
		if err != nil || declaredSize < 0 {
			_ = part.Close()
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid file size")
			return
		}
	}
	staged, err := h.media.BeginImport(ctx, projectID, filename, declaredSize, part)
	if err != nil {
		writeMediaImportError(w, h.logger, err)
		return
	}
	if staged == nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "source media import failed")
		return
	}
	if err := part.Close(); err != nil {
		_ = staged.Abort(ctx)
		writeMultipartError(w, err)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = staged.Abort(ctx)
		}
	}()
	if _, err = mr.NextRawPart(); err != io.EOF {
		if err != nil {
			writeMultipartError(w, err)
		} else {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "exactly one file field is required")
		}
		return
	}
	// Multipart readers stop at the closing boundary. Consume the remaining
	// envelope through the same capped reader before committing metadata.
	if _, err = io.Copy(io.Discard, bufferedBody); err != nil {
		writeMultipartError(w, err)
		return
	}
	created, err := staged.Commit(ctx)
	if err != nil {
		writeMediaImportError(w, h.logger, err)
		return
	}
	committed = true
	responseMedia := sourceMediaToProto(created)
	if created.Kind == domain.MediaKindVideo && h.inspection != nil {
		job, _, inspectionErr := h.inspection.EnsureSourceInspection(ctx, projectID, created.ID)
		if inspectionErr != nil {
			h.logger.Printf("video imported but inspection submission failed for %s/%s: %v", projectID, created.ID, inspectionErr)
			responseMedia.InspectionIssueCode = "inspection_pending"
			responseMedia.InspectionIssueMessage = "Video was imported; source inspection is pending and can be retried."
		} else {
			responseMedia.InspectionJobId = job.ID
			if len(job.Attempts) > 0 {
				responseMedia.InspectionAttemptId = job.Attempts[len(job.Attempts)-1].ID
			}
			responseMedia.InspectionJob = inspectionJobToProto(job)
			if h.activator != nil {
				if activationErr := h.activator.ActivateTestRoute(ctx); activationErr != nil {
					h.logger.Printf("video inspection accepted but route activation failed for job %s: %v", job.ID, activationErr)
					responseMedia.InspectionIssueCode = "inspection_activation_pending"
					responseMedia.InspectionIssueMessage = "Video was imported and inspection was accepted, but progress activation is pending; retry inspection submission with the same media id."
				}
			}
		}
	}
	httpx.WriteProto(w, http.StatusCreated, &projectsv1.UploadSourceMediaResponse{SourceMedia: responseMedia})
}

func writeMediaImportError(w http.ResponseWriter, logger *log.Logger, err error) {
	var invalid domain.ErrInvalidSourceMedia
	var tooLarge domain.ErrSourceMediaTooLarge
	var missing domain.ErrProjectNotFound
	var invalidProject domain.ErrInvalidProject
	var requestTooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &requestTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "multipart request exceeds the upload envelope limit")
	case errors.As(err, &invalid):
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, invalid.Error())
	case errors.As(err, &tooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "source media exceeds the upload limit")
	case errors.As(err, &missing):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "project not found")
	case errors.As(err, &invalidProject):
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid project")
	default:
		logger.Printf("projects source media import failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "source media import failed")
	}
}

func writeMultipartError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "multipart request exceeds the upload envelope limit")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid multipart upload")
}
