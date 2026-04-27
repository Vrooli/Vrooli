package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxFilePreviewBytes int64 = 256 * 1024

type resolveFileReferenceRequest struct {
	Path string `json:"path"`
}

type fileReferenceResolveResponse struct {
	InputPath       string `json:"input_path"`
	ResolvedPath    string `json:"resolved_path"`
	Line            *int   `json:"line,omitempty"`
	Exists          bool   `json:"exists"`
	ResolutionBasis string `json:"resolution_basis"`
	Category        string `json:"category"`
	CanPreview      bool   `json:"can_preview"`
}

type fileReferenceContentResponse struct {
	Path        string `json:"path"`
	Line        *int   `json:"line,omitempty"`
	Category    string `json:"category"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
	Truncated   bool   `json:"truncated"`
}

type fileReferenceResolution struct {
	inputPath       string
	resolvedPath    string
	line            *int
	exists          bool
	resolutionBasis string
	category        string
	canPreview      bool
	sizeBytes       int64
}

func (s *Server) handleResolveFileReference(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	var req resolveFileReferenceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		writeCatalogError(w, "file_reference_invalid", "path is required")
		return
	}

	resolved, err := s.resolveFileReference(r.Context(), sess, req.Path)
	if err != nil {
		s.writeFileReferenceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, fileReferenceResolveResponse{
		InputPath:       resolved.inputPath,
		ResolvedPath:    resolved.resolvedPath,
		Line:            resolved.line,
		Exists:          resolved.exists,
		ResolutionBasis: resolved.resolutionBasis,
		Category:        resolved.category,
		CanPreview:      resolved.canPreview,
	})
}

func (s *Server) handleGetFileReferenceContent(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if rawPath == "" {
		writeCatalogError(w, "file_reference_invalid", "path query parameter is required")
		return
	}

	resolved, err := s.resolveFileReference(r.Context(), sess, rawPath)
	if err != nil {
		s.writeFileReferenceError(w, err)
		return
	}
	if !resolved.canPreview {
		writeCatalogError(w, "file_reference_not_previewable", "File type cannot be previewed")
		return
	}
	if resolved.sizeBytes > maxFilePreviewBytes {
		writeCatalogError(w, "file_reference_too_large", fmt.Sprintf("File exceeds preview limit of %d bytes", maxFilePreviewBytes))
		return
	}

	data, err := os.ReadFile(resolved.resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeCatalogError(w, "file_reference_not_found", "Referenced file was not found")
			return
		}
		writeCatalogError(w, "internal_error", "Failed to read referenced file")
		return
	}
	if !utf8.Valid(data) || bytesContainNull(data) {
		writeCatalogError(w, "file_reference_not_previewable", "File is not valid UTF-8 text")
		return
	}

	writeJSON(w, http.StatusOK, fileReferenceContentResponse{
		Path:        resolved.resolvedPath,
		Line:        resolved.line,
		Category:    resolved.category,
		ContentType: fileReferenceContentType(resolved.category),
		Content:     string(data),
		Truncated:   false,
	})
}

func (s *Server) writeFileReferenceError(w http.ResponseWriter, err error) {
	var refErr *fileReferenceError
	if !asFileReferenceError(err, &refErr) {
		writeCatalogError(w, "internal_error", "Failed to resolve file reference")
		return
	}
	writeCatalogError(w, refErr.code, refErr.message)
}

type fileReferenceError struct {
	code    string
	message string
}

func (e *fileReferenceError) Error() string { return e.message }

func asFileReferenceError(err error, target **fileReferenceError) bool {
	refErr, ok := err.(*fileReferenceError)
	if !ok {
		return false
	}
	*target = refErr
	return true
}

func newFileReferenceError(code, message string) error {
	return &fileReferenceError{code: code, message: message}
}

func (s *Server) resolveFileReference(ctx context.Context, sess *Session, rawPath string) (*fileReferenceResolution, error) {
	inputPath, line, normalizeErr := normalizeFileReferencePath(rawPath)
	if normalizeErr != nil {
		return nil, normalizeErr
	}
	if inputPath == "" {
		return nil, newFileReferenceError("file_reference_invalid", "Referenced path is empty")
	}

	projectRoot, err := filepath.Abs(resolveWorkingDir())
	if err != nil {
		projectRoot = filepath.Clean(resolveWorkingDir())
	}
	sessionUploadRoot, err := filepath.Abs(filepath.Join(resolveUploadDir(), sess.ID))
	if err != nil {
		sessionUploadRoot = filepath.Clean(filepath.Join(resolveUploadDir(), sess.ID))
	}

	liveCwd, cwdErr := sess.CurrentDir(ctx)
	if cwdErr == nil && liveCwd != "" {
		if absCwd, absErr := filepath.Abs(liveCwd); absErr == nil {
			liveCwd = absCwd
		} else {
			liveCwd = filepath.Clean(liveCwd)
		}
	}

	candidates := make([]struct {
		path  string
		basis string
	}, 0, 3)

	if filepath.IsAbs(inputPath) {
		candidates = append(candidates, struct {
			path  string
			basis string
		}{path: filepath.Clean(inputPath), basis: "absolute_allowed"})
	} else {
		if liveCwd != "" {
			candidates = append(candidates, struct {
				path  string
				basis string
			}{path: filepath.Join(liveCwd, inputPath), basis: "session_cwd"})
		}
		candidates = append(candidates, struct {
			path  string
			basis string
		}{path: filepath.Join(projectRoot, inputPath), basis: "project_root"})
	}

	var lastDenied bool
	for _, candidate := range candidates {
		resolvedPath, basis, ok, resolutionErr := evaluateCandidate(candidate.path, candidate.basis, projectRoot, sessionUploadRoot)
		if resolutionErr != nil {
			lastDenied = true
			continue
		}
		if !ok {
			continue
		}

		info, statErr := os.Stat(resolvedPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, newFileReferenceError("internal_error", "Failed to stat referenced file")
		}
		if info.IsDir() {
			return nil, newFileReferenceError("file_reference_not_previewable", "Directories cannot be previewed")
		}

		category, canPreview := classifyFileReference(resolvedPath, info.Size())
		return &fileReferenceResolution{
			inputPath:       rawPath,
			resolvedPath:    resolvedPath,
			line:            line,
			exists:          true,
			resolutionBasis: basis,
			category:        category,
			canPreview:      canPreview,
			sizeBytes:       info.Size(),
		}, nil
	}

	if lastDenied && filepath.IsAbs(inputPath) {
		return nil, newFileReferenceError("file_reference_not_allowed", "Referenced path is outside allowed roots")
	}
	if cwdErr != nil && !filepath.IsAbs(inputPath) {
		return nil, newFileReferenceError("file_reference_unresolvable", "Relative path could not be resolved from the current session context")
	}
	return nil, newFileReferenceError("file_reference_not_found", "Referenced file was not found")
}

func evaluateCandidate(candidatePath, basis, projectRoot, sessionUploadRoot string) (string, string, bool, error) {
	absCandidate, err := filepath.Abs(candidatePath)
	if err != nil {
		return "", "", false, newFileReferenceError("file_reference_invalid", "Referenced path is invalid")
	}
	absCandidate = filepath.Clean(absCandidate)
	if isWithinRoot(absCandidate, projectRoot) {
		return absCandidate, basis, true, nil
	}
	if isWithinRoot(absCandidate, sessionUploadRoot) {
		return absCandidate, "session_upload", true, nil
	}
	return "", "", false, newFileReferenceError("file_reference_not_allowed", "Referenced path is outside allowed roots")
}

func normalizeFileReferencePath(raw string) (string, *int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, nil
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "file://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", nil, newFileReferenceError("file_reference_invalid", "Referenced file URL is invalid")
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", nil, newFileReferenceError("file_reference_not_allowed", "Only local file URLs can be previewed")
		}
		decoded, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", nil, newFileReferenceError("file_reference_invalid", "Referenced file URL path is invalid")
		}
		path, line := splitFileReferenceLine(decoded)
		return strings.TrimSpace(path), line, nil
	}

	if decoded, err := url.PathUnescape(trimmed); err == nil {
		trimmed = decoded
	}
	path, line := splitFileReferenceLine(trimmed)
	return strings.TrimSpace(path), line, nil
}

func splitFileReferenceLine(raw string) (string, *int) {
	trimmed := strings.TrimSpace(raw)
	lastColon := strings.LastIndex(trimmed, ":")
	if lastColon <= 0 || lastColon == len(trimmed)-1 {
		return trimmed, nil
	}
	lineValue, err := strconv.Atoi(trimmed[lastColon+1:])
	if err != nil || lineValue <= 0 {
		return trimmed, nil
	}
	return trimmed[:lastColon], &lineValue
}

func isWithinRoot(candidate, root string) bool {
	if root == "" {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func classifyFileReference(path string, sizeBytes int64) (category string, canPreview bool) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".mdx", ".markdown":
		return "markdown", true
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp", ".tiff", ".pdf":
		return "binary", false
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".json", ".yml", ".yaml", ".sh", ".bash", ".zsh", ".sql", ".css", ".html", ".txt", ".proto", ".toml", ".ini", ".env":
		return "code", true
	default:
		if sizeBytes > maxFilePreviewBytes {
			return "text", true
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "text", false
		}
		if !utf8.Valid(data) || bytesContainNull(data) {
			return "binary", false
		}
		return "text", true
	}
}

func fileReferenceContentType(category string) string {
	switch category {
	case "markdown":
		return "text/markdown; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func bytesContainNull(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}
