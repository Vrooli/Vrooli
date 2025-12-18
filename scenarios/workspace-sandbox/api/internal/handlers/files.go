package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/types"
)

// --- File Operations API ---
// These endpoints provide direct file access within sandboxes, enabling agents
// to read, write, and manage files without using the exec endpoint.

// FileInfo represents metadata about a file or directory.
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
}

// TreeResponse represents a directory listing.
type TreeResponse struct {
	Path    string     `json:"path"`
	Entries []FileInfo `json:"entries"`
	Total   int        `json:"total"`
}

// ReadFileResponse represents the content of a file.
type ReadFileResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "utf8" or "base64"
	Size     int64  `json:"size"`
	IsBinary bool   `json:"isBinary"`
}

// WriteFileRequest represents a request to write a file.
type WriteFileRequest struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty"` // "utf8" (default) or "base64"
	Mode     int    `json:"mode,omitempty"`     // File mode (default: 0644)
}

// WriteFileResponse represents the result of writing a file.
type WriteFileResponse struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Created bool   `json:"created"` // true if file was created, false if updated
}

// DeleteFileResponse represents the result of deleting a file.
type DeleteFileResponse struct {
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
}

// ListFiles handles listing files in a sandbox directory.
// GET /sandboxes/{id}/files?path=/&depth=1
func (h *Handlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to list files", http.StatusConflict)
		return
	}

	// Get path from query (default to root)
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		relPath = "/"
	}

	// Resolve and validate the path
	fullPath, err := h.resolveAndValidatePath(sb, relPath)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Read directory
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.JSONError(w, "path not found", http.StatusNotFound)
			return
		}
		h.JSONError(w, fmt.Sprintf("failed to read directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	resp := TreeResponse{
		Path:    relPath,
		Entries: make([]FileInfo, 0, len(entries)),
		Total:   len(entries),
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := filepath.Join(relPath, entry.Name())
		if relPath == "/" {
			entryPath = "/" + entry.Name()
		}

		resp.Entries = append(resp.Entries, FileInfo{
			Name:    entry.Name(),
			Path:    entryPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	h.JSONSuccess(w, resp)
}

// ReadFile handles reading a file from a sandbox.
// GET /sandboxes/{id}/files/content?path=/path/to/file
func (h *Handlers) ReadFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to read files", http.StatusConflict)
		return
	}

	// Get path from query
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		h.JSONError(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Resolve and validate the path
	fullPath, err := h.resolveAndValidatePath(sb, relPath)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if it's a file
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.JSONError(w, "file not found", http.StatusNotFound)
			return
		}
		h.JSONError(w, fmt.Sprintf("failed to stat file: %v", err), http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		h.JSONError(w, "path is a directory, use list endpoint", http.StatusBadRequest)
		return
	}

	// Check file size limit (10MB for API reads)
	const maxReadSize = 10 * 1024 * 1024
	if info.Size() > maxReadSize {
		h.JSONError(w, fmt.Sprintf("file too large (%.1f MB), max is 10 MB", float64(info.Size())/(1024*1024)), http.StatusBadRequest)
		return
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	// Detect if binary
	isBinary := isBinaryContent(content)

	resp := ReadFileResponse{
		Path:     relPath,
		Size:     info.Size(),
		IsBinary: isBinary,
	}

	if isBinary {
		resp.Encoding = "base64"
		resp.Content = base64.StdEncoding.EncodeToString(content)
	} else {
		resp.Encoding = "utf8"
		resp.Content = string(content)
	}

	h.JSONSuccess(w, resp)
}

// WriteFile handles writing a file to a sandbox.
// PUT /sandboxes/{id}/files/content?path=/path/to/file
func (h *Handlers) WriteFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to write files", http.StatusConflict)
		return
	}

	// Get path from query
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		h.JSONError(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req WriteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Resolve and validate the path
	fullPath, err := h.resolveAndValidatePath(sb, relPath)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Decode content
	var content []byte
	switch req.Encoding {
	case "base64":
		content, err = base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			h.JSONError(w, "invalid base64 content", http.StatusBadRequest)
			return
		}
	case "utf8", "":
		content = []byte(req.Content)
	default:
		h.JSONError(w, "encoding must be 'utf8' or 'base64'", http.StatusBadRequest)
		return
	}

	// Check if file exists (for created flag)
	_, statErr := os.Stat(fullPath)
	created := os.IsNotExist(statErr)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.JSONError(w, fmt.Sprintf("failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Set file mode
	mode := os.FileMode(0644)
	if req.Mode != 0 {
		mode = os.FileMode(req.Mode)
	}

	// Write the file
	if err := os.WriteFile(fullPath, content, mode); err != nil {
		h.JSONError(w, fmt.Sprintf("failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	resp := WriteFileResponse{
		Path:    relPath,
		Size:    int64(len(content)),
		Created: created,
	}

	if created {
		h.JSONCreated(w, resp)
	} else {
		h.JSONSuccess(w, resp)
	}
}

// DeleteFile handles deleting a file from a sandbox.
// DELETE /sandboxes/{id}/files/content?path=/path/to/file
func (h *Handlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to delete files", http.StatusConflict)
		return
	}

	// Get path from query
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		h.JSONError(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Prevent deleting root
	if relPath == "/" || relPath == "" {
		h.JSONError(w, "cannot delete root directory", http.StatusBadRequest)
		return
	}

	// Resolve and validate the path
	fullPath, err := h.resolveAndValidatePath(sb, relPath)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if it exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Idempotent - return success even if already deleted
			h.JSONSuccess(w, DeleteFileResponse{Path: relPath, Deleted: false})
			return
		}
		h.JSONError(w, fmt.Sprintf("failed to stat file: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete file or directory
	if info.IsDir() {
		// Check if recursive deletion is allowed
		if r.URL.Query().Get("recursive") != "true" {
			h.JSONError(w, "path is a directory, use recursive=true to delete", http.StatusBadRequest)
			return
		}
		err = os.RemoveAll(fullPath)
	} else {
		err = os.Remove(fullPath)
	}

	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to delete: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, DeleteFileResponse{Path: relPath, Deleted: true})
}

// MkdirRequest represents a request to create a directory.
type MkdirRequest struct {
	Path string `json:"path"`
}

// Mkdir handles creating a directory in a sandbox.
// POST /sandboxes/{id}/files/mkdir
func (h *Handlers) Mkdir(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to create directories", http.StatusConflict)
		return
	}

	// Parse request body
	var req MkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		h.JSONError(w, "path is required", http.StatusBadRequest)
		return
	}

	// Resolve and validate the path
	fullPath, err := h.resolveAndValidatePath(sb, req.Path)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create directory (and parents)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		h.JSONError(w, fmt.Sprintf("failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSONCreated(w, map[string]interface{}{
		"path":    req.Path,
		"created": true,
	})
}

// --- Helper Functions ---

// resolveAndValidatePath resolves a relative path within a sandbox and validates it.
// It prevents path traversal attacks and ensures the path is within the sandbox.
func (h *Handlers) resolveAndValidatePath(sb *types.Sandbox, relPath string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(relPath)

	// Remove leading slash for joining
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	// Prevent path traversal
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	// The sandbox's merged directory is the root for file operations
	if sb.MergedDir == "" {
		return "", fmt.Errorf("sandbox has no merged directory (mount may be unhealthy)")
	}

	// Join with merged dir
	fullPath := filepath.Join(sb.MergedDir, cleanPath)

	// Verify the resolved path is still within the merged dir (defense in depth)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	absMergedDir, err := filepath.Abs(sb.MergedDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve merged dir: %w", err)
	}

	// Ensure path is within merged dir
	if !strings.HasPrefix(absFullPath, absMergedDir) {
		return "", fmt.Errorf("path escapes sandbox")
	}

	return fullPath, nil
}

// isBinaryContent checks if content appears to be binary (contains null bytes).
func isBinaryContent(content []byte) bool {
	// Check first 8000 bytes for null bytes (common heuristic)
	checkLen := len(content)
	if checkLen > 8000 {
		checkLen = 8000
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// DownloadFile serves a file as a download (binary-safe).
// GET /sandboxes/{id}/files/download?path=/path/to/file
func (h *Handlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to download files", http.StatusConflict)
		return
	}

	// Get path from query
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		h.JSONError(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	// Resolve and validate the path
	fullPath, err := h.resolveAndValidatePath(sb, relPath)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Open file
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.JSONError(w, "file not found", http.StatusNotFound)
			return
		}
		h.JSONError(w, fmt.Sprintf("failed to open file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to stat file: %v", err), http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		h.JSONError(w, "cannot download a directory", http.StatusBadRequest)
		return
	}

	// Set headers for download
	filename := filepath.Base(relPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	// Stream file content
	io.Copy(w, file)
}
