// Package assets owns HTTP transport for the assets domain. Multipart upload
// and public byte serving are explicit REST exceptions to the unary proto API.
package assets

import (
	"context"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

const (
	MaxUploadSize      = 10 * 1024 * 1024
	maxMultipartMemory = 1 * 1024 * 1024
)

type UploadInput struct {
	File       multipart.File
	Header     *multipart.FileHeader
	Category   string
	AltText    string
	UploadedBy string
}

type UploadResult struct {
	Payload  any
	ID       int
	Filename string
	Category string
	Size     int64
}

// Dependencies adapt the root composition package's service and common HTTP
// helpers without importing package main into this domain package.
type Dependencies struct {
	Upload             func(context.Context, UploadInput) (UploadResult, error)
	List               func(string) (any, bool, error)
	Get                func(int) (any, error)
	Delete             func(context.Context, int) error
	ResolveStoragePath func(context.Context, string) (string, error)
	PathInt            func(http.ResponseWriter, *http.Request, string) (int, bool)
	Path               func(*http.Request, string) (string, bool)
	WriteError         func(http.ResponseWriter, int, string, string)
	WriteJSON          func(http.ResponseWriter, any)
	Log                func(string, map[string]any)
	IsNotFound         func(error) bool
	DetectMimeType     func(string) string
}

func Upload(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
		// #nosec G120 -- MaxBytesReader above caps the full request at MaxUploadSize;
		// ParseMultipartForm keeps at most maxMultipartMemory in memory and spills the rest to disk.
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "File too large or invalid form data", "validation")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, "No file provided in 'file' field", "validation")
			return
		}
		defer file.Close()

		asset, err := deps.Upload(r.Context(), UploadInput{File: file, Header: header, Category: r.FormValue("category"), AltText: r.FormValue("alt_text"), UploadedBy: r.FormValue("uploaded_by")})
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "invalid file type") {
				status = http.StatusBadRequest
			} else if strings.Contains(err.Error(), "file exceeds") {
				status = http.StatusRequestEntityTooLarge
			}
			deps.Log("asset_upload_failed", map[string]any{"error": err.Error(), "filename": header.Filename})
			deps.WriteError(w, status, err.Error(), map[bool]string{true: "validation", false: "server_error"}[status != http.StatusInternalServerError])
			return
		}
		deps.Log("asset_uploaded", map[string]any{"id": asset.ID, "filename": asset.Filename, "category": asset.Category, "size": asset.Size})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		deps.WriteJSON(w, asset.Payload)
	}
}

func List(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assets, empty, err := deps.List(r.URL.Query().Get("category"))
		if err != nil {
			deps.Log("list_assets_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to list assets", "server_error")
			return
		}
		if empty {
			assets = []any{}
		}
		deps.WriteJSON(w, map[string]any{"assets": assets})
	}
}

func Get(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt(w, r, "id")
		if !ok {
			return
		}
		asset, err := deps.Get(id)
		if err != nil {
			if deps.IsNotFound(err) {
				deps.WriteError(w, http.StatusNotFound, "Asset not found", "not_found")
				return
			}
			deps.Log("get_asset_failed", map[string]any{"id": id, "error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get asset", "server_error")
			return
		}
		deps.WriteJSON(w, asset)
	}
}

func Delete(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt(w, r, "id")
		if !ok {
			return
		}
		if err := deps.Delete(r.Context(), id); err != nil {
			if deps.IsNotFound(err) {
				deps.WriteError(w, http.StatusNotFound, "Asset not found", "not_found")
				return
			}
			deps.Log("delete_asset_failed", map[string]any{"id": id, "error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to delete asset", "server_error")
			return
		}
		deps.Log("asset_deleted", map[string]any{"id": id})
		w.WriteHeader(http.StatusNoContent)
	}
}

func Serve(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storagePath, ok := deps.Path(r, "path")
		if !ok || storagePath == "" {
			deps.WriteError(w, http.StatusBadRequest, "File path required", "validation")
			return
		}
		fullPath, err := deps.ResolveStoragePath(r.Context(), storagePath)
		if err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid path", "validation")
			return
		}
		stat, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			deps.WriteError(w, http.StatusNotFound, "File not found", "not_found")
			return
		}
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Failed to access file", "server_error")
			return
		}
		if stat.IsDir() {
			deps.WriteError(w, http.StatusBadRequest, "Cannot serve directory", "validation")
			return
		}
		w.Header().Set("Content-Type", deps.DetectMimeType(fullPath))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, fullPath)
	}
}
