package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/storage"
)

const maxUploadSize = 20 << 20 // 20 MB

func resolveUploadDir() string {
	return mustResolveScenarioStorageDir(storage.ClassCache, "uploads")
}

// resolveUploadDirFor resolves the uploads root for one request. Under a test
// lease the request context routes to the leased Cache root, so a BAS upload
// never lands in the operator's real uploads tree; without a lease it resolves
// to exactly the same path as resolveUploadDir.
func (s *Server) resolveUploadDirFor(ctx context.Context) string {
	if s.roots == nil {
		return resolveUploadDir()
	}
	root, err := s.roots.Pick(ctx, storage.ClassCache)
	if err != nil || strings.TrimSpace(root) == "" {
		return resolveUploadDir()
	}
	return ensureDir(filepath.Join(root, "uploads"))
}

// allowedImageTypes maps accepted MIME types for image uploads.
var allowedImageTypes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// unsafeFilenameChars matches characters unsafe for filenames.
var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// sanitizeFilename strips path components and dangerous characters from a
// user-supplied filename to prevent path traversal and other attacks.
func sanitizeFilename(name string) string {
	// Strip directory components
	name = filepath.Base(name)
	// Replace unsafe characters
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	// Collapse runs of underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	name = strings.Trim(name, "_.")
	if name == "" || name == "." || name == ".." {
		name = "upload"
	}
	// Limit length
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}
	return name
}

// uniquePath returns a path that doesn't conflict with existing files by
// appending a numeric suffix before the extension when necessary.
func uniquePath(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 10*1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return candidate
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	if sess.IsDead() {
		writeCatalogError(w, "session_terminated", "Cannot upload to terminated session "+sanitizeID(sess.ID))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			writeCatalogError(w, "upload_too_large", "File exceeds maximum upload size of 20MB")
			return
		}
		writeCatalogError(w, "invalid_body", "Failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeCatalogError(w, "invalid_body", "Missing file field")
		return
	}
	defer file.Close()

	// Validate MIME type from Content-Type header
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	// Normalize: take only the media type, not parameters
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	if !allowedImageTypes[ct] {
		writeCatalogError(w, "invalid_upload_type", "Only image files are accepted (png, jpeg, gif, webp, svg)")
		return
	}

	// Create session-scoped upload directory
	sessionDir := filepath.Join(s.resolveUploadDirFor(r.Context()), sess.ID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		writeCatalogError(w, "internal_error", "Failed to create upload directory")
		return
	}

	safeName := sanitizeFilename(header.Filename)
	destPath := uniquePath(sessionDir, safeName)

	dst, err := os.Create(destPath)
	if err != nil {
		writeCatalogError(w, "internal_error", "Failed to create file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(destPath)
		writeCatalogError(w, "internal_error", "Failed to write file")
		return
	}
	// Records which root the write actually landed in. The lease reports a
	// primary write during test mode as an isolation leak, so this call is
	// what makes the evidence real rather than assumed.
	s.roots.RecordWrite(r.Context())

	writeJSON(w, http.StatusOK, map[string]string{"path": destPath})
}
