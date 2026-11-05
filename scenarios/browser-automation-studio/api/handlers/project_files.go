package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
)

// normalizeProjectRelPath cleans a project-root-relative path and rejects
// anything that escapes the root. Kept in this package because peer
// handlers (projects.go) still use it for legacy REST paths; the proto-first
// project_files handler has its own copy.
func normalizeProjectRelPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" || raw == "." {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", false
	}
	return clean, true
}

func safeJoinProjectPath(projectRoot, relPath string) (string, error) {
	projectRoot = filepath.Clean(strings.TrimSpace(projectRoot))
	if projectRoot == "" || projectRoot == "." {
		return "", errors.New("invalid project root")
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		return "", errors.New("invalid project relative path")
	}
	abs := filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(relPath)))
	rootWithSep := projectRoot + string(filepath.Separator)
	if abs != projectRoot && !strings.HasPrefix(abs, rootWithSep) {
		return "", errors.New("path escapes project root")
	}
	return abs, nil
}

// ServeProjectFile handles GET /api/v1/projects/{id}/files/*.
//
// RESTException: streams arbitrary file bytes with MIME types decided by
// extension; consumed by the browser via <img>, <a download>, and file
// viewers. Cannot be expressed as a Connect-RPC method.
// RESTReason: third_party_shape (browser-native binary streaming).
// See docs/internal/REST_EXCEPTIONS.md.
func (h *Handler) ServeProjectFile(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidProjectID)
	if !ok {
		return
	}

	filePath := chi.URLParam(r, "*")
	if filePath == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "file_path"}))
		return
	}
	filePath = strings.TrimPrefix(filePath, "/")
	relPath, ok := normalizeProjectRelPath(filePath)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid file path"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.repo.GetProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrProjectNotFound)
			return
		}
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_project"}))
		return
	}

	absPath, joinErr := safeJoinProjectPath(project.FolderPath, relPath)
	if joinErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": joinErr.Error()}))
		return
	}

	info, statErr := os.Stat(absPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			h.respondError(w, ErrProjectFileNotFound)
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stat_file"}))
		return
	}
	if info.IsDir() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "path is a directory, not a file"}))
		return
	}

	file, openErr := os.Open(absPath)
	if openErr != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "open_file"}))
		return
	}
	defer file.Close()

	ext := filepath.Ext(absPath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Cache-Control", "private, max-age=60")

	if _, err := io.Copy(w, file); err != nil {
		h.log.WithError(err).WithField("path", relPath).Error("Failed to stream project file")
	}
}
