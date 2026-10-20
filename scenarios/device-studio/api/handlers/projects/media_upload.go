package projects

import (
	"bytes"
	"device-studio/internal/httpx"
	domain "device-studio/internal/projects"
	"errors"
	"github.com/gorilla/mux"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"io"
	"log"
	"net/http"
	"strconv"
)

const maxMediaMultipartBytes int64 = domain.MaxSourceMediaBytes + (1 << 20)

type mediaUploadHandler struct {
	projects domain.Service
	media    domain.SourceMediaService
	logger   *log.Logger
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
		}
	})
}

func newMediaUploadHandler(projects domain.Service, media domain.SourceMediaService, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc((&mediaUploadHandler{projects: projects, media: media, logger: logger}).upload)
}

func (h *mediaUploadHandler) upload(w http.ResponseWriter, r *http.Request) {
	if h.media == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "source media is unavailable")
		return
	}
	projectID := mux.Vars(r)["project_id"]
	if _, err := h.projects.Get(r.Context(), projectID); err != nil {
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
	r.Body = http.MaxBytesReader(w, r.Body, maxMediaMultipartBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "multipart/form-data is required")
		return
	}
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
	data, err := io.ReadAll(io.LimitReader(part, domain.MaxSourceMediaBytes+1))
	_ = part.Close()
	if err != nil {
		writeMultipartError(w, err)
		return
	}
	if int64(len(data)) > domain.MaxSourceMediaBytes {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "source media exceeds the upload limit")
		return
	}
	if _, err = mr.NextPart(); err != io.EOF {
		if err != nil {
			writeMultipartError(w, err)
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "exactly one file field is required")
		return
	}
	created, err := h.media.Import(r.Context(), projectID, filename, int64(len(data)), bytes.NewReader(data))
	if err != nil {
		var invalid domain.ErrInvalidSourceMedia
		var tooLarge domain.ErrSourceMediaTooLarge
		var missing domain.ErrProjectNotFound
		var invalidProject domain.ErrInvalidProject
		switch {
		case errors.As(err, &invalid):
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, invalid.Error())
		case errors.As(err, &tooLarge):
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "source media exceeds the upload limit")
		case errors.As(err, &missing):
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "project not found")
		case errors.As(err, &invalidProject):
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid project")
		default:
			h.logger.Printf("projects source media import failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "source media import failed")
		}
		return
	}
	httpx.WriteProto(w, http.StatusCreated, &projectsv1.UploadSourceMediaResponse{SourceMedia: sourceMediaToProto(created)})
}

func writeMultipartError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "multipart request exceeds the upload envelope limit")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid multipart upload")
}
